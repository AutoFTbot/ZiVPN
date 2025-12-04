# ZiVPN Tunnel UDP

Repository ini berisi script installer untuk ZiVPN UDP Tunnel yang telah dimodifikasi oleh **AutoFTbot**.

## Fitur
- **Bahasa Indonesia**: Seluruh script menggunakan Bahasa Indonesia.
- **Tanpa UI**: Instalasi ringan tanpa panel menu berbasis shell. Manajemen user dilakukan langsung melalui API binary Golang.
- **API Golang**: Tersedia source code API (`zivpn-api.go`) untuk manajemen user secara programatik.
- **Otomatisasi**: Script instalasi otomatis mengatur service systemd, iptables, dan sertifikat SSL.

## Instalasi

### Untuk VPS AMD64 (Umum)
```bash
wget https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/install-amd.sh && chmod +x install-amd.sh && ./install-amd.sh
```

### Untuk VPS ARM64
```bash
wget https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/install-arm.sh && chmod +x install-arm.sh && ./install-arm.sh
```

## Penggunaan API (Golang)

Anda dapat menjalankan API service untuk mengelola pengguna secara remote atau lokal.

### 1. Build API
Pastikan Go sudah terinstal di VPS Anda.
```bash
cd /root/zivpn-tunnel-udp # Sesuaikan dengan lokasi direktori
go build -o zivpn-api zivpn-api.go
```

### 2. Jalankan API
```bash
./zivpn-api
# Output: ZiVPN API berjalan di port :8080
```
*Disarankan untuk menjalankan ini sebagai service systemd agar berjalan di background.*

### 3. Dokumentasi Endpoint

**Header Wajib:**
`X-API-Key: zivpn-secret-token` (Ganti token di source code jika perlu)

#### A. Buat User Baru
*   **URL**: `/api/user/create`
*   **Method**: `POST`
*   **Body**:
    ```json
    {
        "password": "user123",
        "days": 30
    }
    ```

#### B. Hapus User
*   **URL**: `/api/user/delete`
*   **Method**: `POST`
*   **Body**:
    ```json
    {
        "password": "user123"
    }
    ```

#### C. Perpanjang User
*   **URL**: `/api/user/renew`
*   **Method**: `POST`
*   **Body**:
    ```json
    {
        "password": "user123",
        "days": 30
    }
    ```

#### D. List Semua User
*   **URL**: `/api/users`
*   **Method**: `GET`

#### E. Info System
*   **URL**: `/api/info`
*   **Method**: `GET`

## Uninstall
Untuk menghapus instalasi:
```bash
wget https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/uninstall.sh && chmod +x uninstall.sh && ./uninstall.sh
```