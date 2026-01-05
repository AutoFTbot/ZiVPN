package com.zivpn.app.data.model

import androidx.room.Entity
import androidx.room.PrimaryKey
import com.google.gson.annotations.SerializedName

// Server yang disimpan di local database
@Entity(tableName = "servers")
data class Server(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    val name: String,
    val domain: String,
    val port: Int = 5667,
    val apiPort: Int = 8080,
    val apiKey: String,
    val obfs: String = "zivpn",
    val createdAt: Long = System.currentTimeMillis()
)

// User/Akun dari API
data class User(
    val password: String,
    val expired: String,
    val status: String
)

// Response dari API
data class ApiResponse<T>(
    val success: Boolean,
    val message: String,
    val data: T?
)

// Request untuk create/delete/renew user
data class UserRequest(
    val password: String,
    val days: Int = 0
)

// Info server dari API
data class ServerInfo(
    val domain: String,
    @SerializedName("public_ip") val publicIp: String,
    @SerializedName("private_ip") val privateIp: String,
    val port: String,
    val service: String
)

// Cleanup response
data class CleanupResult(
    @SerializedName("deleted_count") val deletedCount: Int,
    @SerializedName("deleted_users") val deletedUsers: List<String>
)

// VPN Connection State
enum class VpnState {
    DISCONNECTED,
    CONNECTING,
    CONNECTED,
    DISCONNECTING,
    ERROR
}

data class VpnConfig(
    val server: Server,
    val password: String
)
