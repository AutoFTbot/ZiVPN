package com.zivpn.app.data.repository

import com.zivpn.app.data.api.ApiClient
import com.zivpn.app.data.model.*
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class UserRepository @Inject constructor() {

    suspend fun getUsers(server: Server): Result<List<User>> {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.listUsers()
            if (response.isSuccessful && response.body()?.success == true) {
                Result.success(response.body()!!.data ?: emptyList())
            } else {
                Result.failure(Exception(response.body()?.message ?: "Unknown error"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createUser(server: Server, password: String, days: Int): Result<User> {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.createUser(UserRequest(password, days))
            if (response.isSuccessful && response.body()?.success == true) {
                Result.success(response.body()!!.data!!)
            } else {
                Result.failure(Exception(response.body()?.message ?: "Gagal membuat user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteUser(server: Server, password: String): Result<Unit> {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.deleteUser(UserRequest(password))
            if (response.isSuccessful && response.body()?.success == true) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.body()?.message ?: "Gagal menghapus user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun renewUser(server: Server, password: String, days: Int): Result<User> {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.renewUser(UserRequest(password, days))
            if (response.isSuccessful && response.body()?.success == true) {
                Result.success(response.body()!!.data!!)
            } else {
                Result.failure(Exception(response.body()?.message ?: "Gagal memperpanjang user"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun cleanupExpired(server: Server): Result<CleanupResult> {
        return try {
            val api = ApiClient.getApi(server)
            val response = api.cleanupExpired()
            if (response.isSuccessful && response.body()?.success == true) {
                Result.success(response.body()!!.data ?: CleanupResult(0, emptyList()))
            } else {
                Result.failure(Exception(response.body()?.message ?: "Gagal cleanup"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
