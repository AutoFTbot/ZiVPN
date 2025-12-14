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

const (
	ConfigFile = "/etc/zivpn/config.json"
	UserDB     = "/etc/zivpn/users.json"
	DomainFile = "/etc/zivpn/domain"
	ApiKeyFile = "/etc/zivpn/apikey"
	Port       = ":8080"
)

var AuthToken = "AutoFtBot-agskjgdvsbdreiWG1234512SDKrqw"

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

type UserRequest struct {
	Password string `json:"password"`
	Days     int    `json:"days"`
	IpLimit  int    `json:"ip_limit"`
}

type UserStore struct {
	Password string `json:"password"`
	Expired  string `json:"expired"`
	IpLimit  int    `json:"ip_limit"`
	Status   string `json:"status"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var mutex = &sync.Mutex{}

func main() {
	if keyBytes, err := ioutil.ReadFile(ApiKeyFile); err == nil {
		AuthToken = strings.TrimSpace(string(keyBytes))
	}

	http.HandleFunc("/api/user/create", authMiddleware(createUser))
	http.HandleFunc("/api/user/delete", authMiddleware(deleteUser))
	http.HandleFunc("/api/user/renew", authMiddleware(renewUser))
	http.HandleFunc("/api/users", authMiddleware(listUsers))
	http.HandleFunc("/api/info", authMiddleware(getSystemInfo))

	go monitorUserLimits()

	fmt.Printf("ZiVPN API berjalan di port %s\n", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}

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

func jsonResponse(w http.ResponseWriter, status int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: success,
		Message: message,
		Data:    data,
	})
}

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

	config.Auth.Config = append(config.Auth.Config, req.Password)
	if err := saveConfig(config); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan config", nil)
		return
	}

	expDate := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour).Format("2006-01-02")
	limit := req.IpLimit
	if limit <= 0 {
		limit = 0 // unlimited
	}

	users, err := loadUsers()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca database user", nil)
		return
	}

	newUser := UserStore{
		Password: req.Password,
		Expired:  expDate,
		IpLimit:  limit,
		Status:   "active",
	}
	users = append(users, newUser)

	if err := saveUsers(users); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan database user", nil)
		return
	}

	if err := restartService(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal merestart service", nil)
		return
	}

	domain := "Tidak diatur"
	if domainBytes, err := ioutil.ReadFile(DomainFile); err == nil {
		domain = strings.TrimSpace(string(domainBytes))
	}

	jsonResponse(w, http.StatusOK, true, "User berhasil dibuat", map[string]string{
		"password": req.Password,
		"expired":  expDate,
		"ip_limit": fmt.Sprintf("%d", limit),
		"domain":   domain,
	})
}

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

	users, err := loadUsers()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal membaca database user", nil)
		return
	}

	newUsers := []UserStore{}
	for _, u := range users {
		if u.Password == req.Password {
			continue
		}
		newUsers = append(newUsers, u)
	}

	if err := saveUsers(newUsers); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan database user", nil)
		return
	}

	if err := restartService(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal merestart service", nil)
		return
	}

	jsonResponse(w, http.StatusOK, true, "User berhasil dihapus", nil)
}

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
	newUsers := []UserStore{}
	var newExpDate string

	for _, u := range users {
		if u.Password == req.Password {
			found = true
			currentExp, err := time.Parse("2006-01-02", u.Expired)
			if err != nil {
				currentExp = time.Now()
			}
			
			if currentExp.Before(time.Now()) {
				currentExp = time.Now()
			}

			newExp := currentExp.Add(time.Duration(req.Days) * 24 * time.Hour)
			newExpDate = newExp.Format("2006-01-02")
			
			u.Expired = newExpDate
			
			if u.Status == "locked" {
				u.Status = "active"
				go enableUser(req.Password)
			}

			newUsers = append(newUsers, u)
		} else {
			newUsers = append(newUsers, u)
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

	if err := restartService(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Gagal merestart service", nil)
		return
	}

	jsonResponse(w, http.StatusOK, true, "User berhasil diperpanjang", map[string]string{
		"password": req.Password,
		"expired":  newExpDate,
	})
}

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
		IpLimit  int    `json:"ip_limit"`
	}

	userList := []UserInfo{}
	today := time.Now().Format("2006-01-02")

	for _, u := range users {
		status := "Active"
		if u.Status == "locked" {
			status = "Locked"
		} else if u.Expired < today {
			status = "Expired"
		}
		
		userList = append(userList, UserInfo{
			Password: u.Password,
			Expired:  u.Expired,
			Status:   status,
			IpLimit:  u.IpLimit,
		})
	}

	jsonResponse(w, http.StatusOK, true, "Daftar user", userList)
}

func getSystemInfo(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("curl", "-s", "ifconfig.me")
	ipPub, _ := cmd.Output()

	cmd = exec.Command("hostname", "-I")
	ipPriv, _ := cmd.Output()

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

func monitorUserLimits() {
	for {
		time.Sleep(1 * time.Minute)

		users, err := loadUsers()
		if err != nil {
			log.Println("Error loading users for monitor:", err)
			continue
		}

		userLimits := make(map[string]int)
		for _, u := range users {
			if u.IpLimit > 0 {
				userLimits[u.Password] = u.IpLimit
			}
		}

		if len(userLimits) == 0 {
			continue
		}
		cmd := exec.Command("journalctl", "-u", "zivpn.service", "--since", "1 minute ago", "--no-pager")
		out, err := cmd.Output()
		if err != nil {
			log.Println("Error reading logs:", err)
			continue
		}
		logContent := string(out)
		
		activeIps := make(map[string]map[string]bool)

		lines := strings.Split(logContent, "\n")
		for _, l := range lines {
			if strings.Contains(l, "user:") && strings.Contains(l, "source:") {
				fields := strings.Fields(l)
				var username, ip string
				for i, f := range fields {
					if strings.Contains(f, "user:") && i+1 < len(fields) {
						username = strings.Trim(fields[i+1], ",")
					}
					if strings.Contains(f, "source:") && i+1 < len(fields) {
						ip = strings.Trim(fields[i+1], ",")
					}
				}
				
				if username != "" && ip != "" {
					if _, exists := userLimits[username]; exists {
						if activeIps[username] == nil {
							activeIps[username] = make(map[string]bool)
						}
						activeIps[username][ip] = true
					}
				}
			}
		}

		// Check limits
		for user, ips := range activeIps {
			limit := userLimits[user]
			if len(ips) > limit {
				log.Printf("User %s exceeded IP limit (Active: %d, Limit: %d). Locking account.\n", user, len(ips), limit)
				lockUser(user)
			}
		}
	}
}

func lockUser(username string) {
	mutex.Lock()
	defer mutex.Unlock()

	config, err := loadConfig()
	if err == nil {
		newConfigAuth := []string{}
		changed := false
		for _, p := range config.Auth.Config {
			if p == username {
				changed = true
			} else {
				newConfigAuth = append(newConfigAuth, p)
			}
		}
		if changed {
			config.Auth.Config = newConfigAuth
			saveConfig(config)
			restartService()
		}
	}

	users, err := loadUsers()
	if err == nil {
		newUsers := []UserStore{}
		for _, u := range users {
			if u.Password == username {
				u.Status = "locked"
			}
			newUsers = append(newUsers, u)
		}
		saveUsers(newUsers)
	}
}

func enableUser(username string) {
	mutex.Lock()
	defer mutex.Unlock()

	config, err := loadConfig()
	if err != nil {
		return
	}

	exists := false
	for _, p := range config.Auth.Config {
		if p == username {
			exists = true
			break
		}
	}

	if !exists {
		config.Auth.Config = append(config.Auth.Config, username)
		saveConfig(config)
		restartService()
	}
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

func loadUsers() ([]UserStore, error) {
	var users []UserStore
	file, err := ioutil.ReadFile(UserDB)
	if err != nil {
		if os.IsNotExist(err) {
			return users, nil
		}
		return nil, err
	}
	err = json.Unmarshal(file, &users)
	return users, err
}

func saveUsers(users []UserStore) error {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(UserDB, data, 0644)
}

func restartService() error {
	cmd := exec.Command("systemctl", "restart", "zivpn.service")
	return cmd.Run()
}
