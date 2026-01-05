package com.zivpn.app.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import android.util.Log
import androidx.core.app.NotificationCompat
import com.zivpn.app.R
import com.zivpn.app.data.model.VpnState
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import java.io.*
import java.net.InetSocketAddress
import java.net.Socket

/**
 * ZiVPN Service menggunakan Hysteria2 + tun2socks
 * 
 * Flow:
 * 1. Start Hysteria2 -> SOCKS5 proxy di 127.0.0.1:10808
 * 2. Setup VPN TUN interface
 * 3. Start tun2socks -> route traffic dari TUN ke SOCKS5
 */
class ZiVpnService : VpnService() {

    companion object {
        private const val TAG = "ZiVpnService"

        const val ACTION_CONNECT = "com.zivpn.app.CONNECT"
        const val ACTION_DISCONNECT = "com.zivpn.app.DISCONNECT"
        const val EXTRA_SERVER_DOMAIN = "server_domain"
        const val EXTRA_SERVER_PORT = "server_port"
        const val EXTRA_PASSWORD = "password"
        const val EXTRA_OBFS = "obfs"

        private const val CHANNEL_ID = "zivpn_channel"
        private const val NOTIFICATION_ID = 1
        private const val SOCKS_PORT = 10808
        private const val TUN_MTU = 1500

        private val _vpnState = MutableStateFlow(VpnState.DISCONNECTED)
        val vpnState: StateFlow<VpnState> = _vpnState

        private val _connectionInfo = MutableStateFlow<String?>(null)
        val connectionInfo: StateFlow<String?> = _connectionInfo
    }

    private var vpnInterface: ParcelFileDescriptor? = null
    private var hysteriaProcess: Process? = null
    private var tun2socksProcess: Process? = null
    private var tunnelJob: Job? = null
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    private var serverDomain: String = ""
    private var serverPort: Int = 5667
    private var password: String = ""
    private var obfsKey: String = ""

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_CONNECT -> {
                serverDomain = intent.getStringExtra(EXTRA_SERVER_DOMAIN) ?: ""
                serverPort = intent.getIntExtra(EXTRA_SERVER_PORT, 5667)
                password = intent.getStringExtra(EXTRA_PASSWORD) ?: ""
                obfsKey = intent.getStringExtra(EXTRA_OBFS) ?: "zivpn"
                startVpn()
            }
            ACTION_DISCONNECT -> stopVpn()
        }
        return START_STICKY
    }

    private fun startVpn() {
        if (_vpnState.value == VpnState.CONNECTED || _vpnState.value == VpnState.CONNECTING) {
            return
        }

        _vpnState.value = VpnState.CONNECTING
        _connectionInfo.value = "Connecting to $serverDomain..."

        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification("Connecting..."))

        tunnelJob = scope.launch {
            try {
                // 1. Extract binaries
                val hysteriaPath = extractBinary("hysteria")
                val tun2socksPath = extractBinary("tun2socks")

                if (hysteriaPath == null) {
                    _vpnState.value = VpnState.ERROR
                    _connectionInfo.value = "Failed to extract hysteria binary"
                    return@launch
                }

                // 2. Generate Hysteria config
                val configFile = File(filesDir, "hysteria-config.json")
                val config = HysteriaConfig.generate(
                    serverAddress = serverDomain,
                    serverPort = serverPort,
                    password = password,
                    obfs = obfsKey
                )
                configFile.writeText(config)
                Log.d(TAG, "Hysteria config: $config")

                // 3. Start Hysteria process
                _connectionInfo.value = "Starting Hysteria..."
                val hysteriaStarted = startHysteria(hysteriaPath, configFile.absolutePath)
                if (!hysteriaStarted) {
                    _vpnState.value = VpnState.ERROR
                    _connectionInfo.value = "Failed to start Hysteria"
                    return@launch
                }

                // 4. Wait for SOCKS proxy to be ready
                _connectionInfo.value = "Waiting for proxy..."
                if (!waitForSocksProxy()) {
                    _vpnState.value = VpnState.ERROR
                    _connectionInfo.value = "Hysteria failed to start proxy"
                    stopProcesses()
                    return@launch
                }

                // 5. Setup VPN interface
                _connectionInfo.value = "Setting up VPN..."
                vpnInterface = Builder()
                    .setSession("ZiVPN - $serverDomain")
                    .addAddress("10.255.0.1", 30)
                    .addRoute("0.0.0.0", 0)
                    .addDnsServer("8.8.8.8")
                    .addDnsServer("1.1.1.1")
                    .setMtu(TUN_MTU)
                    .setBlocking(false)
                    .establish()

                if (vpnInterface == null) {
                    _vpnState.value = VpnState.ERROR
                    _connectionInfo.value = "Failed to create VPN interface"
                    stopProcesses()
                    return@launch
                }

                // 6. Start tun2socks
                if (tun2socksPath != null) {
                    _connectionInfo.value = "Starting tunnel..."
                    val tunFd = vpnInterface!!.fd
                    startTun2Socks(tun2socksPath, tunFd)
                }

                _vpnState.value = VpnState.CONNECTED
                _connectionInfo.value = "Connected to $serverDomain"
                updateNotification("Connected to $serverDomain")

                // Monitor processes
                monitorProcesses()

            } catch (e: Exception) {
                Log.e(TAG, "VPN Error", e)
                _vpnState.value = VpnState.ERROR
                _connectionInfo.value = "Error: ${e.message}"
                stopProcesses()
            }
        }
    }

    private fun extractBinary(name: String): String? {
        return try {
            val arch = when {
                Build.SUPPORTED_ABIS.any { it.contains("arm64") } -> "arm64"
                Build.SUPPORTED_ABIS.any { it.contains("arm") } -> "arm"
                else -> "arm64"
            }

            val assetName = "$name-$arch"
            val targetFile = File(filesDir, name)

            // Check if already extracted and executable
            if (targetFile.exists() && targetFile.canExecute()) {
                Log.d(TAG, "$name already extracted")
                return targetFile.absolutePath
            }

            // Extract from assets
            try {
                assets.open("bin/$assetName").use { input ->
                    FileOutputStream(targetFile).use { output ->
                        input.copyTo(output)
                    }
                }
                targetFile.setExecutable(true)
                Log.d(TAG, "Extracted $name to ${targetFile.absolutePath}")
                targetFile.absolutePath
            } catch (e: FileNotFoundException) {
                Log.w(TAG, "Binary $assetName not found in assets")
                null
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to extract $name", e)
            null
        }
    }

    private fun startHysteria(binaryPath: String, configPath: String): Boolean {
        return try {
            val cmd = arrayOf(binaryPath, "client", "-c", configPath)
            Log.d(TAG, "Starting Hysteria: ${cmd.joinToString(" ")}")

            val processBuilder = ProcessBuilder(*cmd)
                .directory(filesDir)
                .redirectErrorStream(true)

            hysteriaProcess = processBuilder.start()

            // Log output in background
            scope.launch {
                try {
                    hysteriaProcess?.inputStream?.bufferedReader()?.use { reader ->
                        reader.lineSequence().forEach { line ->
                            Log.d(TAG, "Hysteria: $line")
                            // Check for errors
                            if (line.contains("error", ignoreCase = true) || 
                                line.contains("failed", ignoreCase = true)) {
                                Log.e(TAG, "Hysteria error: $line")
                            }
                        }
                    }
                } catch (e: Exception) {
                    if (_vpnState.value == VpnState.CONNECTED) {
                        Log.e(TAG, "Error reading hysteria output", e)
                    }
                }
            }

            true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start hysteria", e)
            false
        }
    }

    private fun startTun2Socks(binaryPath: String, tunFd: Int): Boolean {
        return try {
            // tun2socks v2 command format
            val cmd = arrayOf(
                binaryPath,
                "-device", "fd://$tunFd",
                "-proxy", "socks5://127.0.0.1:$SOCKS_PORT",
                "-interface", "lo",
                "-loglevel", "warning"
            )
            Log.d(TAG, "Starting tun2socks: ${cmd.joinToString(" ")}")

            val processBuilder = ProcessBuilder(*cmd)
                .directory(filesDir)
                .redirectErrorStream(true)

            tun2socksProcess = processBuilder.start()

            // Log output
            scope.launch {
                try {
                    tun2socksProcess?.inputStream?.bufferedReader()?.use { reader ->
                        reader.lineSequence().forEach { line ->
                            Log.d(TAG, "tun2socks: $line")
                        }
                    }
                } catch (e: Exception) {
                    if (_vpnState.value == VpnState.CONNECTED) {
                        Log.e(TAG, "Error reading tun2socks output", e)
                    }
                }
            }

            true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to start tun2socks", e)
            false
        }
    }

    private suspend fun waitForSocksProxy(timeoutMs: Long = 15000): Boolean {
        val startTime = System.currentTimeMillis()
        while (System.currentTimeMillis() - startTime < timeoutMs) {
            // Check if hysteria is still running
            if (hysteriaProcess?.isAlive != true) {
                Log.e(TAG, "Hysteria process died while waiting for proxy")
                return false
            }

            try {
                Socket().use { socket ->
                    socket.connect(InetSocketAddress("127.0.0.1", SOCKS_PORT), 1000)
                    Log.d(TAG, "SOCKS proxy is ready on port $SOCKS_PORT")
                    return true
                }
            } catch (e: Exception) {
                delay(500)
            }
        }
        Log.e(TAG, "Timeout waiting for SOCKS proxy")
        return false
    }

    private suspend fun monitorProcesses() {
        while (_vpnState.value == VpnState.CONNECTED) {
            delay(3000)

            // Check if hysteria is still running
            if (hysteriaProcess?.isAlive != true) {
                Log.w(TAG, "Hysteria process died")
                withContext(Dispatchers.Main) {
                    _vpnState.value = VpnState.ERROR
                    _connectionInfo.value = "Connection lost"
                }
                break
            }
        }
    }

    private fun stopProcesses() {
        try {
            tun2socksProcess?.destroy()
            tun2socksProcess?.waitFor()
        } catch (e: Exception) {
            Log.e(TAG, "Error stopping tun2socks", e)
        }
        tun2socksProcess = null

        try {
            hysteriaProcess?.destroy()
            hysteriaProcess?.waitFor()
        } catch (e: Exception) {
            Log.e(TAG, "Error stopping hysteria", e)
        }
        hysteriaProcess = null
    }

    private fun stopVpn() {
        _vpnState.value = VpnState.DISCONNECTING
        _connectionInfo.value = "Disconnecting..."

        tunnelJob?.cancel()
        tunnelJob = null

        stopProcesses()

        try {
            vpnInterface?.close()
        } catch (e: Exception) {
            Log.e(TAG, "Error closing VPN interface", e)
        }
        vpnInterface = null

        _vpnState.value = VpnState.DISCONNECTED
        _connectionInfo.value = null

        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "ZiVPN Connection",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Shows VPN connection status"
                setShowBadge(false)
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }

    private fun createNotification(text: String): Notification {
        val disconnectIntent = Intent(this, ZiVpnService::class.java).apply {
            action = ACTION_DISCONNECT
        }
        val disconnectPendingIntent = PendingIntent.getService(
            this, 0, disconnectIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("ZiVPN")
            .setContentText(text)
            .setSmallIcon(R.drawable.ic_vpn)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setOngoing(true)
            .addAction(R.drawable.ic_vpn, "Disconnect", disconnectPendingIntent)
            .build()
    }

    private fun updateNotification(text: String) {
        val manager = getSystemService(NotificationManager::class.java)
        manager.notify(NOTIFICATION_ID, createNotification(text))
    }

    override fun onDestroy() {
        stopVpn()
        scope.cancel()
        super.onDestroy()
    }

    override fun onRevoke() {
        stopVpn()
        super.onRevoke()
    }
}
