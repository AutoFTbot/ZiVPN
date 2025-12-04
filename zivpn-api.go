package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Konfigurasi path file
const (
	ConfigFile = "/etc/zivpn/config.json"
	UserDB     = "/etc/zivpn/users.db"
	DomainFile = "/etc/zivpn/domain"
	AuthToken  = "ASdfaASFaSrofasufasf34qwdas4q2rq++ASdasfawerwDF#Q" // Ganti dengan token yang aman
	Port       = ":8080"
)

// Struktur Config.json
type Config struct {
	Listen string `json:"listen"`
	Cert   string `json:"cert"`
	Key    string `json:"key"`
	Obfs   string `json:"obfs"`
	Auth   struct {
		Mode   string   `json:"mode"`
		Config []string `json:"config"`
	} `json:"auth"`
}

// Struktur Request/Response
type UserRequest struct {
	Password string `json:"password"`
	Days     int    `json:"days"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var mutex = &sync.Mutex{}

func main() {
	http.HandleFunc("/api/user/create", authMiddleware(createUser))
	http.HandleFunc("/api/user/delete", authMiddleware(deleteUser))
	http.HandleFunc("/api/user/renew", authMiddleware(renewUser))
	http.HandleFunc("/api/users", authMiddleware(listUsers))
	http.HandleFunc("/api/info", authMiddleware(getSystemInfo))

	fmt.Printf("ZiVPN API berjalan di port %s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}

// Middleware untuk autentikasi sederhana
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Key")
		if token != AuthToken {
			jsonResponse(w, http.StatusUnauthorized, false, "Unauthorized", nil)
			return
		}
		next(w, r)
	}
}

// Helper untuk mengirim respon JSON
func jsonResponse(w http.ResponseWriter, status int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: success,
		Message: message,
		Data:    data,
	})
}

// Handler: Buat User
func createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, false, "Method not allowed", nil)
		return
	}

	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if req.Password == "" || req.Days <= 0 {
		jsonResponse(w, http.StatusBadRequest, false, "Password dan days harus valid", nil)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// 1. Cek apakah user sudah ada di config.json
	config, err := loadConfig()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca config", nil)
		return
	}

	for _, p := range config.Auth.Config {
		if p == req.Password {
			jsonResponse(w, http.StatusConflict, false, "User sudah ada", nil)
			return
		}
	}

	// 2. Tambah ke config.json
	config.Auth.Config = append(config.Auth.Config, req.Password)
	if err := saveConfig(config); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan config", nil)
		return
	}

	// 3. Tambah ke users.db
	expDate := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour).Format("2006-01-02")
	entry := fmt.Sprintf("%s | %s\n", req.Password, expDate)
	
	f, err := os.OpenFile(UserDB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membuka database user", nil)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menulis database user", nil)
		return
	}

	// 4. Restart Service
	if err := restartService(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal merestart service", nil)
		return
	}

	jsonResponse(w, http.StatusOK, true, "User berhasil dibuat", map[string]string{
		"password": req.Password,
		"expired":  expDate,
	})
}

// Handler: Hapus User
func deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, false, "Method not allowed", nil)
		return
	}

	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// 1. Hapus dari config.json
	config, err := loadConfig()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca config", nil)
		return
	}

	found := false
	newConfigAuth := []string{}
	for _, p := range config.Auth.Config {
		if p == req.Password {
			found = true
		} else {
			newConfigAuth = append(newConfigAuth, p)
		}
	}

	if !found {
		jsonResponse(w, http.StatusNotFound, false, "User tidak ditemukan", nil)
		return
	}

	config.Auth.Config = newConfigAuth
	if err := saveConfig(config); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan config", nil)
		return
	}

	// 2. Hapus dari users.db
	users, err := loadUsers()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca database user", nil)
		return
	}

	newUsers := []string{}
	for _, line := range users {
		parts := strings.Split(line, "|")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) == req.Password {
			continue
		}
		newUsers = append(newUsers, line)
	}

	if err := saveUsers(newUsers); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan database user", nil)
		return
	}

	// 3. Restart Service
	if err := restartService(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal merestart service", nil)
		return
	}

	jsonResponse(w, http.StatusOK, true, "User berhasil dihapus", nil)
}

// Handler: Perpanjang User
func renewUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, false, "Method not allowed", nil)
		return
	}

	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	users, err := loadUsers()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca database user", nil)
		return
	}

	found := false
	newUsers := []string{}
	var newExpDate string

	for _, line := range users {
		parts := strings.Split(line, "|")
		if len(parts) >= 2 && strings.TrimSpace(parts[0]) == req.Password {
			found = true
			currentExpStr := strings.TrimSpace(parts[1])
			currentExp, err := time.Parse("2006-01-02", currentExpStr)
			if err != nil {
				// Jika format tanggal salah, anggap hari ini
				currentExp = time.Now()
			}
			
			// Jika sudah expired, mulai dari hari ini. Jika belum, tambah dari tanggal expired.
			if currentExp.Before(time.Now()) {
				currentExp = time.Now()
			}

			newExp := currentExp.Add(time.Duration(req.Days) * 24 * time.Hour)
			newExpDate = newExp.Format("2006-01-02")
			newUsers = append(newUsers, fmt.Sprintf("%s | %s", req.Password, newExpDate))
		} else {
			newUsers = append(newUsers, line)
		}
	}

	if !found {
		jsonResponse(w, http.StatusNotFound, false, "User tidak ditemukan di database", nil)
		return
	}

	if err := saveUsers(newUsers); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan database user", nil)
		return
	}

	// Restart service mungkin tidak diperlukan untuk renew, tapi bagus untuk memastikan konsistensi
	if err := restartService(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal merestart service", nil)
		return
	}

	jsonResponse(w, http.StatusOK, true, "User berhasil diperpanjang", map[string]string{
		"password": req.Password,
		"expired":  newExpDate,
	})
}

// Handler: List Users
func listUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, false, "Method not allowed", nil)
		return
	}

	users, err := loadUsers()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca database user", nil)
		return
	}

	type UserInfo struct {
		Password string `json:"password"`
		Expired  string `json:"expired"`
		Status   string `json:"status"`
	}

	userList := []UserInfo{}
	today := time.Now().Format("2006-01-02")

	for _, line := range users {
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			pass := strings.TrimSpace(parts[0])
			exp := strings.TrimSpace(parts[1])
			status := "Active"
			if exp < today {
				status = "Expired"
			}
			userList = append(userList, UserInfo{
				Password: pass,
				Expired:  exp,
				Status:   status,
			})
		}
	}

	jsonResponse(w, http.StatusOK, true, "Daftar user", userList)
}

// Handler: System Info
func getSystemInfo(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan IP Publik (sederhana)
	cmd := exec.Command("curl", "-s", "ifconfig.me")
	ipPub, _ := cmd.Output()

	// Mendapatkan Hostname/IP Private
	cmd = exec.Command("hostname", "-I")
	ipPriv, _ := cmd.Output()

	// Mendapatkan Domain
	domain := "Tidak diatur"
	if domainBytes, err := ioutil.ReadFile(DomainFile); err == nil {
		domain = strings.TrimSpace(string(domainBytes))
	}

	info := map[string]string{
		"domain":     domain,
		"public_ip":  strings.TrimSpace(string(ipPub)),
		"private_ip": strings.Fields(string(ipPriv))[0],
		"port":       "5667",
		"service":    "zivpn",
	}

	jsonResponse(w, http.StatusOK, true, "System Info", info)
}

// --- Helper Functions ---

func loadConfig() (Config, error) {
	var config Config
	file, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config)
	return config, err
}

func saveConfig(config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(ConfigFile, data, 0644)
}

func loadUsers() ([]string, error) {
	file, err := ioutil.ReadFile(UserDB)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	lines := strings.Split(string(file), "\n")
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

func saveUsers(lines []string) error {
	data := strings.Join(lines, "\n") + "\n"
	return ioutil.WriteFile(UserDB, []byte(data), 0644)
}

func restartService() error {
	cmd := exec.Command("systemctl", "restart", "zivpn.service")
	return cmd.Run()
}
