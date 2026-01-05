package com.zivpn.app.ui.user

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.zivpn.app.data.model.CleanupResult
import com.zivpn.app.data.model.Server
import com.zivpn.app.data.model.User
import com.zivpn.app.data.repository.UserRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class UserUiState(
    val users: List<User> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
    val message: String? = null,
    val cleanupResult: CleanupResult? = null
)

@HiltViewModel
class UserViewModel @Inject constructor(
    private val repository: UserRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(UserUiState())
    val uiState: StateFlow<UserUiState> = _uiState.asStateFlow()

    fun loadUsers(server: Server) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            repository.getUsers(server)
                .onSuccess { users ->
                    _uiState.update { it.copy(users = users, isLoading = false) }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message, isLoading = false) }
                }
        }
    }

    fun createUser(server: Server, password: String, days: Int) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            repository.createUser(server, password, days)
                .onSuccess { user ->
                    _uiState.update { it.copy(isLoading = false, message = "User ${user.password} berhasil dibuat") }
                    loadUsers(server)
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message, isLoading = false) }
                }
        }
    }

    fun deleteUser(server: Server, password: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            repository.deleteUser(server, password)
                .onSuccess {
                    _uiState.update { it.copy(isLoading = false, message = "User $password berhasil dihapus") }
                    loadUsers(server)
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message, isLoading = false) }
                }
        }
    }

    fun renewUser(server: Server, password: String, days: Int) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            repository.renewUser(server, password, days)
                .onSuccess { user ->
                    _uiState.update { it.copy(isLoading = false, message = "User ${user.password} diperpanjang sampai ${user.expired}") }
                    loadUsers(server)
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message, isLoading = false) }
                }
        }
    }

    fun cleanupExpired(server: Server) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            repository.cleanupExpired(server)
                .onSuccess { result ->
                    val msg = if (result.deletedCount > 0) {
                        "Berhasil menghapus ${result.deletedCount} akun expired"
                    } else {
                        "Tidak ada akun expired"
                    }
                    _uiState.update { it.copy(isLoading = false, message = msg, cleanupResult = result) }
                    loadUsers(server)
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
        _uiState.update { it.copy(message = null, cleanupResult = null) }
    }
}
