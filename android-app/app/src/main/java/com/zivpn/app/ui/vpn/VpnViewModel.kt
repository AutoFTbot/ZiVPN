package com.zivpn.app.ui.vpn

import android.app.Application
import android.content.Intent
import android.net.VpnService
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.zivpn.app.data.model.Server
import com.zivpn.app.data.model.VpnState
import com.zivpn.app.vpn.ZiVpnService
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class VpnUiState(
    val vpnState: VpnState = VpnState.DISCONNECTED,
    val connectedServer: Server? = null,
    val connectedPassword: String? = null,
    val error: String? = null
)

@HiltViewModel
class VpnViewModel @Inject constructor(
    private val application: Application
) : AndroidViewModel(application) {

    private val _uiState = MutableStateFlow(VpnUiState())
    val uiState: StateFlow<VpnUiState> = _uiState.asStateFlow()

    init {
        viewModelScope.launch {
            ZiVpnService.vpnState.collect { state ->
                _uiState.update { it.copy(vpnState = state) }
            }
        }
    }

    fun checkVpnPermission(): Intent? {
        return VpnService.prepare(application)
    }

    fun connect(server: Server, password: String) {
        val intent = Intent(application, ZiVpnService::class.java).apply {
            action = ZiVpnService.ACTION_CONNECT
            putExtra(ZiVpnService.EXTRA_SERVER_DOMAIN, server.domain)
            putExtra(ZiVpnService.EXTRA_SERVER_PORT, server.port)
            putExtra(ZiVpnService.EXTRA_PASSWORD, password)
            putExtra(ZiVpnService.EXTRA_OBFS, server.obfs)
        }
        application.startService(intent)
        _uiState.update { it.copy(connectedServer = server, connectedPassword = password) }
    }

    fun disconnect() {
        val intent = Intent(application, ZiVpnService::class.java).apply {
            action = ZiVpnService.ACTION_DISCONNECT
        }
        application.startService(intent)
        _uiState.update { it.copy(connectedServer = null, connectedPassword = null) }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }
}
