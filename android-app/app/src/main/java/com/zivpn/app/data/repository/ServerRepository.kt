package com.zivpn.app.data.repository

import com.zivpn.app.data.api.ApiClient
import com.zivpn.app.data.local.ServerDao
import com.zivpn.app.data.model.*
import kotlinx.coroutines.flow.Flow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class ServerRepository @Inject constructor(
    private val serverDao: ServerDao
) {
    val allServers: Flow<List<Server>> = serverDao.getAllServers()

    suspend fun addServer(server: Server): Long {
        return serverDao.insertServer(server)
    }

    suspend fun updateServer(server: Server) {
        serverDao.updateServer(server)
    }

    suspend fun deleteServer(server: Server) {
        ApiClient.clearCache(server.id)
        serverDao.deleteServer(server)
    }

    suspend fun getServerById(id: Long): Server? {
        return serverDao.getServerById(id)
    }

    // API calls for a specific server
    suspend fun getServerInfo(server: Server): Result<ServerInfo> {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.getServerInfo()
            if (response.isSuccessful && response.body()?.success == true) {
                Result.success(response.body()!!.data!!)
            } else {
                Result.failure(Exception(response.body()?.message ?: "Unknown error"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun testConnection(server: Server): Boolean {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.getServerInfo()
            response.isSuccessful && response.body()?.success == true
        } catch (e: Exception) {
            false
        }
    }
}
