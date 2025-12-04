# ZiVPN UDP Tunnel

Repository ini berisi installer dan API untuk menjalankan ZiVPN UDP Tunnel.

## Fitur

*   **Tanpa UI**: Manajemen user dilakukan sepenuhnya via API (headless).
*   **API Golang**: Termasuk API server untuk Create, Delete, Renew, dan List User.
*   **Auto Install**: Script installer otomatis menginstall Golang, setup API, dan service systemd.
*   **Support**: Linux AMD64 (x86_64) Only.

## Instalasi

Jalankan perintah berikut di terminal VPS Anda (sebagai root):

```bash
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/install.sh && chmod +x install.sh && ./install.sh
```

Saat instalasi, Anda akan diminta memasukkan **Domain**. Domain ini digunakan untuk generate sertifikat SSL.

## API Documentation

Setelah instalasi selesai, API akan berjalan di port `8080`.

**Base URL**: `http://<IP-VPS>:8080`
**Auth Header**: `X-API-Key: zivpn-secret-token` (Default)

### 1. Create User
*   **Endpoint**: `/api/user/create`
*   **Method**: `POST`
*   **Body**:
    ```json
    {
        "password": "user123",
        "days": 30
    }
    ```

### 2. Delete User
*   **Endpoint**: `/api/user/delete`
*   **Method**: `POST`
*   **Body**:
    ```json
    {
        "password": "user123"
    }
    ```

### 3. Renew User
*   **Endpoint**: `/api/user/renew`
*   **Method**: `POST`
*   **Body**:
    ```json
    {
        "password": "user123",
        "days": 30
    }
    ```

### 4. List Users
*   **Endpoint**: `/api/users`
*   **Method**: `GET`

### 5. System Info
*   **Endpoint**: `/api/info`
*   **Method**: `GET`

## Uninstall

Untuk menghapus ZiVPN dan API-nya:

```bash
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/uninstall.sh && chmod +x uninstall.sh && ./uninstall.sh
```