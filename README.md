# ZiVPN UDP Tunnel

**ZiVPN UDP Tunnel** adalah solusi tunneling UDP premium dengan manajemen yang mudah, aman, dan otomatis. Dilengkapi dengan **API Server** dan **Telegram Bot** untuk pengelolaan user tanpa ribet.

---

## 🌟 Fitur Utama

*   **Minimalist CLI**: Installer dengan tampilan modern, bersih, dan elegan.
*   **Headless Management**: Manajemen user sepenuhnya via API atau Bot (tanpa menu CLI jadul).
*   **Telegram Bot Integration**: Kelola user (Create, Delete, Renew, List) langsung dari Telegram.
*   **Dynamic Security**: API Key dan sertifikat SSL digenerate otomatis saat instalasi.
*   **High Performance**: Menggunakan core UDP ZiVPN yang dioptimalkan untuk Linux AMD64.

---

## 📥 Instalasi

Jalankan perintah berikut di terminal VPS Anda (sebagai root):

```bash
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/install.sh && chmod +x install.sh && ./install.sh
```

### Konfigurasi Saat Instalasi
Saat script berjalan, Anda akan diminta memasukkan:
1.  **Domain**: Wajib diisi untuk generate sertifikat SSL (contoh: `vpn.domain.com`).
2.  **API Key**:
    *   Tekan **Enter** untuk menggunakan key acak yang aman (Recommended).
    *   Atau ketik key manual jika diinginkan.
3.  **Telegram Bot** (Opsional):
    *   **Bot Token**: Token dari @BotFather.
    *   **Admin ID**: ID Telegram Anda (cek di @userinfobot).
    *   *Kosongkan jika tidak ingin mengaktifkan bot.*

---

## 🤖 Telegram Bot Usage

Jika Anda mengaktifkan bot, Anda bisa mengelola VPN langsung dari chat Telegram.

*   **/start**: Menampilkan Menu Utama dengan tombol interaktif.
*   **Create User**: Membuat user baru (Input Username -> Input Durasi).
*   **Delete User**: Menghapus user (Input Username).
*   **Renew User**: Memperpanjang masa aktif user.
*   **List Users**: Melihat daftar user aktif dan expired.
*   **System Info**: Cek IP, Domain, dan status service.

> **Note**: Bot hanya merespon perintah dari **Admin ID** yang didaftarkan saat instalasi.

---

## 🔌 API Documentation

API berjalan di port `8080`. Gunakan **API Key** yang Anda atur saat instalasi pada header `X-API-Key`.

**Base URL**: `http://<IP-VPS>:8080`
**Header**: `X-API-Key: <YOUR-API-KEY>`

### 1. Create User
Membuat user baru.
*   **Endpoint**: `/api/user/create`
*   **Method**: `POST`
*   **Body**:
    ```json
    { "password": "user123", "days": 30 }
    ```
*   **Response**:
    ```json
    {
        "success": true,
        "message": "User berhasil dibuat",
        "data": {
            "password": "user123",
            "expired": "2024-12-31",
            "domain": "vpn.domain.com"
        }
    }
    ```

### 2. Delete User
Menghapus user.
*   **Endpoint**: `/api/user/delete`
*   **Method**: `POST`
*   **Body**:
    ```json
    { "password": "user123" }
    ```

### 3. Renew User
Memperpanjang durasi user.
*   **Endpoint**: `/api/user/renew`
*   **Method**: `POST`
*   **Body**:
    ```json
    { "password": "user123", "days": 30 }
    ```

### 4. List Users
Melihat semua user.
*   **Endpoint**: `/api/users`
*   **Method**: `GET`

### 5. System Info
Melihat informasi server.
*   **Endpoint**: `/api/info`
*   **Method**: `GET`

---

## 🛠️ Pemecahan Masalah (Troubleshooting)

### 1. Log "TCP error" di Jurnal
Jika Anda melihat log seperti:
`ERROR TCP error {"addr": "140.213.xx.xx:..."}`

*   **Penyebab**: Koneksi client tidak stabil (sering terjadi pada jaringan seluler/Indosat) atau masalah MTU.
*   **Solusi**:
    *   Ini biasanya **bukan error server**. Jika user masih bisa connect, abaikan saja.
    *   Jika user sering disconnect, sarankan user menurunkan **MTU** di aplikasi client mereka (coba `1100` atau `1200`).

### 2. Bot Telegram Tidak Merespon
*   Pastikan service berjalan: `systemctl status zivpn-bot`
*   Cek log error: `journalctl -u zivpn-bot -f`
*   Pastikan **Bot Token** dan **Admin ID** benar di `/etc/zivpn/bot-config.json`.
*   Restart bot: `systemctl restart zivpn-bot`

### 3. API Error "Unauthorized"
*   Pastikan Anda menggunakan **API Key** yang benar di header `X-API-Key`.
*   Cek key yang aktif di server: `cat /etc/zivpn/apikey`

### 4. Service Gagal Start
*   Cek status: `systemctl status zivpn`
*   Pastikan port `5667` (UDP) dan `8080` (TCP) tidak terpakai aplikasi lain.
*   Cek config: `cat /etc/zivpn/config.json`

---

## 🗑️ Uninstall

Untuk menghapus ZiVPN, API, Bot, dan semua konfigurasi:

```bash
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/uninstall.sh && chmod +x uninstall.sh && ./uninstall.sh
```