package com.zivpn.app.ui.server

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.zivpn.app.data.model.Server
import com.zivpn.app.data.model.ServerInfo
import com.zivpn.app.data.repository.ServerRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ServerUiState(
    val servers: List<Server> = emptyList(),
    val selectedServer: Server? = null,
    val serverInfo: ServerInfo? = null,
    val isLoading: Boolean = false,
    val error: String? = null,
    val message: String? = null
)

@HiltViewModel
class ServerViewModel @Inject constructor(
    private val repository: ServerRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(ServerUiState())
    val uiState: StateFlow<ServerUiState> = _uiState.asStateFlow()

    init {
        viewModelScope.launch {
            repository.allServers.collect { servers ->
                _uiState.update { it.copy(servers = servers) }
            }
        }
    }

    fun addServer(name: String, domain: String, port: Int, apiPort: Int, apiKey: String, obfs: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val server = Server(
                    name = name,
                    domain = domain,
                    port = port,
                    apiPort = apiPort,
                    apiKey = apiKey,
                    obfs = obfs
                )
                
                // Test connection first
                val isConnected = repository.testConnection(server)
                if (isConnected) {
                    repository.addServer(server)
                    _uiState.update { it.copy(isLoading = false, message = "Server berhasil ditambahkan") }
                } else {
                    _uiState.update { it.copy(isLoading = false, error = "Gagal terhubung ke server") }
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun deleteServer(server: Server) {
        viewModelScope.launch {
            repository.deleteServer(server)
            _uiState.update { it.copy(message = "Server dihapus") }
        }
    }

    fun selectServer(server: Server) {
        viewModelScope.launch {
            _uiState.update { it.copy(selectedServer = server, isLoading = true) }
            repository.getServerInfo(server)
                .onSuccess { info ->
                    _uiState.update { it.copy(serverInfo = info, isLoading = false) }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message, isLoading = false) }
                }
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }

    fun clearMessage() {
        _uiState.update { it.copy(message = null) }
    }
}
