package com.zivpn.app.data.api

import com.zivpn.app.data.model.*
import retrofit2.Response
import retrofit2.http.*

interface ZiVpnApi {

    @POST("user/create")
    suspend fun createUser(
        @Body request: UserRequest
    ): Response<ApiResponse<User>>

    @POST("user/delete")
    suspend fun deleteUser(
        @Body request: UserRequest
    ): Response<ApiResponse<Nothing>>

    @POST("user/renew")
    suspend fun renewUser(
        @Body request: UserRequest
    ): Response<ApiResponse<User>>

    @GET("users")
    suspend fun listUsers(): Response<ApiResponse<List<User>>>

    @GET("info")
    suspend fun getServerInfo(): Response<ApiResponse<ServerInfo>>

    @POST("cron/cleanup")
    suspend fun cleanupExpired(): Response<ApiResponse<CleanupResult>>

    @POST("cron/expire")
    suspend fun checkExpiration(): Response<ApiResponse<Nothing>>
}
