package com.zivpn.app.data.api

import com.zivpn.app.data.model.Server
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

object ApiClient {

    private val clients = mutableMapOf<Long, ZiVpnApi>()

    fun getApi(server: Server): ZiVpnApi {
        return clients.getOrPut(server.id) {
            createApi(server)
        }
    }

    private fun createApi(server: Server): ZiVpnApi {
        val authInterceptor = Interceptor { chain ->
            val request = chain.request().newBuilder()
                .addHeader("X-API-Key", server.apiKey)
                .addHeader("Content-Type", "application/json")
                .build()
            chain.proceed(request)
        }

        val loggingInterceptor = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        val client = OkHttpClient.Builder()
            .addInterceptor(authInterceptor)
            .addInterceptor(loggingInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()

        val baseUrl = "http://${server.domain}:${server.apiPort}/api/"

        return Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
            .create(ZiVpnApi::class.java)
    }

    fun clearCache(serverId: Long) {
        clients.remove(serverId)
    }

    fun clearAllCache() {
        clients.clear()
    }
}
