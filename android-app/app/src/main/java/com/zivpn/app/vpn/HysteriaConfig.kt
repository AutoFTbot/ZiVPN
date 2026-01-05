package com.zivpn.app.vpn

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.google.gson.annotations.SerializedName

/**
 * Hysteria2 Config Generator
 * 
 * Generate config file untuk Hysteria2 client binary.
 * Referensi: https://v2.hysteria.network/docs/advanced/Full-Client-Config/
 */
object HysteriaConfig {

    private val gson: Gson = GsonBuilder().setPrettyPrinting().create()

    /**
     * Generate Hysteria2 client config
     */
    fun generate(
        serverAddress: String,
        serverPort: Int,
        password: String,
        obfs: String = "",
        sni: String = ""
    ): String {
        val config = Hysteria2Config(
            server = "$serverAddress:$serverPort",
            auth = password,
            obfs = if (obfs.isNotEmpty()) ObfsConfig(type = "salamander", salamander = SalamanderPassword(password = obfs)) else null,
            tls = TlsConfig(
                sni = sni.ifEmpty { serverAddress },
                insecure = true // Allow self-signed certs
            ),
            quic = QuicConfig(
                initStreamReceiveWindow = 8388608,
                maxStreamReceiveWindow = 8388608,
                initConnReceiveWindow = 20971520,
                maxConnReceiveWindow = 20971520
            ),
            bandwidth = BandwidthConfig(
                up = "100 mbps",
                down = "100 mbps"
            ),
            socks5 = Socks5Config(listen = "127.0.0.1:10808"),
            http = HttpConfig(listen = "127.0.0.1:10809")
        )

        return gson.toJson(config)
    }

    /**
     * Generate config untuk TUN mode (VPN)
     */
    fun generateTunConfig(
        serverAddress: String,
        serverPort: Int,
        password: String,
        obfs: String = "",
        sni: String = ""
    ): String {
        val config = Hysteria2TunConfig(
            server = "$serverAddress:$serverPort",
            auth = password,
            obfs = if (obfs.isNotEmpty()) ObfsConfig(type = "salamander", salamander = SalamanderPassword(password = obfs)) else null,
            tls = TlsConfig(
                sni = sni.ifEmpty { serverAddress },
                insecure = true
            ),
            quic = QuicConfig(
                initStreamReceiveWindow = 8388608,
                maxStreamReceiveWindow = 8388608,
                initConnReceiveWindow = 20971520,
                maxConnReceiveWindow = 20971520
            ),
            bandwidth = BandwidthConfig(
                up = "100 mbps",
                down = "100 mbps"
            ),
            tcpForwarding = listOf(
                ForwardRule(listen = "127.0.0.1:10850", remote = "127.0.0.1:1")
            )
        )

        return gson.toJson(config)
    }
}

// Hysteria2 Config Data Classes
data class Hysteria2Config(
    val server: String,
    val auth: String,
    val obfs: ObfsConfig?,
    val tls: TlsConfig,
    val quic: QuicConfig? = null,
    val bandwidth: BandwidthConfig? = null,
    val socks5: Socks5Config? = null,
    val http: HttpConfig? = null
)

data class Hysteria2TunConfig(
    val server: String,
    val auth: String,
    val obfs: ObfsConfig?,
    val tls: TlsConfig,
    val quic: QuicConfig? = null,
    val bandwidth: BandwidthConfig? = null,
    @SerializedName("tcpForwarding") val tcpForwarding: List<ForwardRule>? = null
)

data class ObfsConfig(
    val type: String,
    val salamander: SalamanderPassword?
)

data class SalamanderPassword(
    val password: String
)

data class TlsConfig(
    val sni: String,
    val insecure: Boolean = false
)

data class QuicConfig(
    val initStreamReceiveWindow: Int? = null,
    val maxStreamReceiveWindow: Int? = null,
    val initConnReceiveWindow: Int? = null,
    val maxConnReceiveWindow: Int? = null
)

data class BandwidthConfig(
    val up: String,
    val down: String
)

data class Socks5Config(
    val listen: String
)

data class HttpConfig(
    val listen: String
)

data class ForwardRule(
    val listen: String,
    val remote: String
)
