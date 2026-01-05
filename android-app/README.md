# ZiVPN Android App

Aplikasi Android untuk mengelola server ZiVPN dan koneksi UDP VPN menggunakan Hysteria2 protocol.

## Fitur
- 🖥️ Add/List/Delete Server
- 👤 List/Delete/Renew Akun di Server  
- 🧹 Cleanup akun expired
- 🔌 Koneksi VPN dengan Hysteria2 + tun2socks
- 🔐 Autentikasi dengan API Key

## Tech Stack
- Kotlin + Coroutines
- Hilt (Dependency Injection)
- Room (Local Database)
- Retrofit (HTTP Client)
- Navigation Component
- VpnService API
- Hysteria2 binary (bundled)
- tun2socks (bundled)

## Cara Kerja VPN

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Android   │────▶│  tun2socks   │────▶│  Hysteria2  │
│  TUN Device │     │ (TUN→SOCKS)  │     │   Client    │
└─────────────┘     └──────────────┘     └─────────────┘
                                                │
                                                ▼
                                         ┌─────────────┐
                                         │   ZiVPN     │
                                         │   Server    │
                                         │ (Hysteria2) │
                                         └─────────────┘
```

1. **Hysteria2** - Connect ke server, buat SOCKS5 proxy di `127.0.0.1:10808`
2. **VPN TUN** - Android VpnService buat virtual network interface
3. **tun2socks** - Route semua traffic dari TUN ke SOCKS5 proxy

## Bundled Binaries

Binary sudah di-bundle di `app/src/main/assets/bin/`:
- `hysteria-arm64` - Hysteria2 client untuk ARM64
- `hysteria-arm` - Hysteria2 client untuk ARMv7
- `tun2socks-arm64` - tun2socks untuk ARM64
- `tun2socks-arm` - tun2socks untuk ARMv7

## Build APK

### Menggunakan Android Studio
1. Buka folder `android-app` di Android Studio
2. Tunggu Gradle sync selesai
3. Build > Build Bundle(s) / APK(s) > Build APK(s)

### Menggunakan Command Line
```bash
# Pastikan ANDROID_HOME sudah di-set
export ANDROID_HOME=~/Android/Sdk

# Build debug APK
./gradlew assembleDebug

# APK ada di: app/build/outputs/apk/debug/app-debug.apk
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/user/create` | POST | Buat akun baru |
| `/api/user/delete` | POST | Hapus akun |
| `/api/user/renew` | POST | Perpanjang akun |
| `/api/users` | GET | List semua akun |
| `/api/info` | GET | Info server |
| `/api/cron/cleanup` | POST | Hapus akun expired |

## Konfigurasi Server

Untuk menambah server di app:
- **Nama**: Label untuk server
- **Domain/IP**: Alamat server ZiVPN
- **Port VPN**: Port Hysteria (default: 5667)
- **Port API**: Port API management (default: 8080)
- **API Key**: Key untuk autentikasi API
- **Obfs Key**: Obfuscation key (sama dengan config server)

## Troubleshooting

### VPN tidak connect
1. Pastikan server ZiVPN berjalan
2. Cek port 5667 terbuka di firewall
3. Pastikan password/obfs key benar
4. Cek log di Logcat dengan tag `ZiVpnService`

### API error
1. Pastikan port API (8080) terbuka
2. Cek API key benar
3. Pastikan `zivpn-api` service berjalan di server

## License

MIT
