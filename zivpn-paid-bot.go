package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ==========================================
// Constants & Configuration
// ==========================================

const (
	BotConfigFile = "/etc/zivpn/bot-config.json"
	ApiPortFile   = "/etc/zivpn/api_port"
	ApiKeyFile    = "/etc/zivpn/apikey"
	DomainFile    = "/etc/zivpn/domain"
	PortFile	  = "/etc/zivpn/port"
)

var ApiUrl = "http://127.0.0.1:" + PortFile + "/api"

var ApiKey = "AutoFtBot-agskjgdvsbdreiWG1234512SDKrqw"

type BotConfig struct {
	BotToken      string `json:"bot_token"`
	AdminID        int64  `json:"admin_id"`
	Mode           string `json:"mode"`
	Domain         string `json:"domain"`
	PakasirSlug    string `json:"pakasir_slug"`
	PakasirApiKey  string `json:"pakasir_api_key"`
	DailyPrice     int    `json:"daily_price"`
}

type IpInfo struct {
	City string `json:"city"`
	Isp  string `json:"isp"`
}

type UserData struct {
	Password string `json:"password"`
	Expired  string `json:"expired"`
	Status   string `json:"status"`
}

// ==========================================
// Global State
// ==========================================

var userStates = make(map[int64]string)
var tempUserData = make(map[int64]map[string]string)
var lastMessageIDs = make(map[int64]int)
var mutex = &sync.Mutex{}

// ==========================================
// Main Entry Point
// ==========================================

func main() {
	if keyBytes, err := ioutil.ReadFile(ApiKeyFile); err == nil {
		ApiKey = strings.TrimSpace(string(keyBytes))
	}

	// Load API Port
	if portBytes, err := ioutil.ReadFile(ApiPortFile); err == nil {
		port := strings.TrimSpace(string(portBytes))
		ApiUrl = fmt.Sprintf("http://127.0.0.1:%s/api", port)
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatal("Gagal memuat konfigurasi bot:", err)
	}

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Start Payment Checker
	go startPaymentChecker(bot, &config)

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message, &config)
		} else if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery, &config)
		}
	}
}

// ==========================================
// Telegram Event Handlers
// ==========================================

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, config *BotConfig) {
	// In Paid Bot, everyone can access, but actions are restricted/paid
	// Admin still has full control

	if state, exists := userStates[msg.From.ID]; exists {
		handleState(bot, msg, state, config)
		return
	}

	// Handle Document Upload (Restore) - Admin Only
	if msg.Document != nil && msg.From.ID == config.AdminID {
		if state, exists := userStates[msg.From.ID]; exists && state == "waiting_restore_file" {
			processRestoreFile(bot, msg, config)
			return
		}
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			showMainMenu(bot, msg.Chat.ID, config)
		default:
			replyError(bot, msg.Chat.ID, "Perintah tidak dikenal.")
		}
	}
}

func handleCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, config *BotConfig) {
	chatID := query.Message.Chat.ID
	userID := query.From.ID

	switch {
	case query.Data == "menu_create":
		startCreateUser(bot, chatID, userID)
	case query.Data == "menu_info":
		systemInfo(bot, chatID, config)
	case query.Data == "cancel":
		cancelOperation(bot, chatID, userID, config)

	case query.Data == "menu_admin":
		if userID == config.AdminID {
			showBackupRestoreMenu(bot, chatID)
		}
	case query.Data == "menu_backup_action":
		if userID == config.AdminID {
			performBackup(bot, chatID)
		}
	case query.Data == "menu_restore_action":
		if userID == config.AdminID {
			startRestore(bot, chatID, userID)
		}
	}

	bot.Request(tgbotapi.NewCallback(query.ID, ""))
}

func handleState(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state string, config *BotConfig) {
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	switch state {
	case "create_password":
		if !validatePassword(bot, chatID, text) {
			return
		}
		mutex.Lock()
		tempUserData[userID]["password"] = text
		mutex.Unlock()
		userStates[userID] = "create_days"
		sendMessage(bot, chatID, fmt.Sprintf("⏳ Masukkan Durasi (hari)\nHarga: Rp %d / hari:", config.DailyPrice))

	case "create_days":
		days, ok := validateNumber(bot, chatID, text, 1, 365, "Durasi")
		if !ok {
			return
		}
		mutex.Lock()
		tempUserData[userID]["days"] = text
		mutex.Unlock()

		// Process Payment
		processPayment(bot, chatID, userID, days, config)
	}
}

// ==========================================
// Feature Implementation
// ==========================================

func startCreateUser(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	userStates[userID] = "create_password"
	mutex.Lock()
	tempUserData[userID] = make(map[string]string)
	tempUserData[userID]["chat_id"] = strconv.FormatInt(chatID, 10)
	mutex.Unlock()
	sendMessage(bot, chatID, "👤 Masukkan Password Baru:")
}

func processPayment(bot *tgbotapi.BotAPI, chatID int64, userID int64, days int, config *BotConfig) {
	price := days * config.DailyPrice
	if price < 500 {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Total harga Rp %d. Minimal transaksi adalah Rp 500.\nSilakan tambah durasi.", price))
		return
	}
	orderID := fmt.Sprintf("ZIVPN-%d-%d", userID, time.Now().Unix())

	// Call Pakasir API
	payment, err := createPakasirTransaction(config, orderID, price)
	if err != nil {
		replyError(bot, chatID, "Gagal membuat pembayaran: "+err.Error())
		resetState(userID)
		return
	}

	// Store Order ID for verification
	mutex.Lock()
	tempUserData[userID]["order_id"] = orderID
	tempUserData[userID]["price"] = strconv.Itoa(price)
	mutex.Unlock()

	// Generate QR Image URL
	qrUrl := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=%s", payment.PaymentNumber)

	msgText := fmt.Sprintf("💳 **Tagihan Pembayaran**\n\nPassword: `%s`\nDurasi: %d Hari\nTotal: Rp %d\n\nSilakan scan QRIS di atas untuk membayar.\nSistem akan otomatis mengecek pembayaran setiap menit.\nExpired: %s",
		tempUserData[userID]["password"], days, price, payment.ExpiredAt)

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(qrUrl))
	photo.Caption = msgText
	photo.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "cancel"),
		),
	)
	photo.ReplyMarkup = keyboard

	deleteLastMessage(bot, chatID)
	sentMsg, err := bot.Send(photo)
	if err == nil {
		lastMessageIDs[chatID] = sentMsg.MessageID
	}

	// Clear state but keep tempUserData for verification
	delete(userStates, userID)
}

func startPaymentChecker(bot *tgbotapi.BotAPI, config *BotConfig) {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		mutex.Lock()
		for userID, data := range tempUserData {
			if orderID, ok := data["order_id"]; ok {
				price := data["price"]
				chatID, _ := strconv.ParseInt(data["chat_id"], 10, 64)
				
				status, err := checkPakasirStatus(config, orderID, price)
				if err == nil && (status == "completed" || status == "success") {
					// Payment Success
					password := data["password"]
					days, _ := strconv.Atoi(data["days"])
					
					createUser(bot, chatID, password, days, config)
					delete(tempUserData, userID)
					delete(userStates, userID)
				} else if err != nil {
					log.Printf("Error checking payment for %d: %v", userID, err)
				}
			}
		}
		mutex.Unlock()
	}
}

func createUser(bot *tgbotapi.BotAPI, chatID int64, password string, days int, config *BotConfig) {
	res, err := apiCall("POST", "/user/create", map[string]interface{}{
		"password": password,
		"days":     days,
	})

	if err != nil {
		replyError(bot, chatID, "Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		data := res["data"].(map[string]interface{})
		sendAccountInfo(bot, chatID, data, config)
	} else {
		replyError(bot, chatID, fmt.Sprintf("Gagal membuat akun: %s", res["message"]))
	}
}

// ==========================================
// Pakasir API
// ==========================================

type PakasirPayment struct {
	PaymentNumber string `json:"payment_number"`
	ExpiredAt     string `json:"expired_at"`
}

func createPakasirTransaction(config *BotConfig, orderID string, amount int) (*PakasirPayment, error) {
	url := fmt.Sprintf("https://app.pakasir.com/api/transactioncreate/qris")
	payload := map[string]interface{}{
		"project":  config.PakasirSlug,
		"order_id": orderID,
		"amount":   amount,
		"api_key":  config.PakasirApiKey,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if paymentData, ok := result["payment"].(map[string]interface{}); ok {
		return &PakasirPayment{
			PaymentNumber: paymentData["payment_number"].(string),
			ExpiredAt:     paymentData["expired_at"].(string),
		}, nil
	}
	return nil, fmt.Errorf("invalid response from Pakasir")
}

func checkPakasirStatus(config *BotConfig, orderID string, amountStr string) (string, error) {
	url := fmt.Sprintf("https://app.pakasir.com/api/transactiondetail?project=%s&amount=%s&order_id=%s&api_key=%s",
		config.PakasirSlug, amountStr, orderID, config.PakasirApiKey)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if transaction, ok := result["transaction"].(map[string]interface{}); ok {
		return transaction["status"].(string), nil
	}
	return "", fmt.Errorf("transaction not found")
}

// ==========================================
// UI & Helpers (Simplified for Paid Bot)
// ==========================================

func showMainMenu(bot *tgbotapi.BotAPI, chatID int64, config *BotConfig) {
	ipInfo, _ := getIpInfo()
	domain := config.Domain
	if domain == "" {
		domain = "(Not Configured)"
	}

	msgText := fmt.Sprintf("```\n━━━━━━━━━━━━━━━━━━━━━\n    STORE ZIVPN UDP\n━━━━━━━━━━━━━━━━━━━━━\n • Domain   : %s\n • City     : %s\n • ISP      : %s\n • Harga    : Rp %d / Hari\n━━━━━━━━━━━━━━━━━━━━━\n```\n👇 Silakan pilih menu dibawah ini:", domain, ipInfo.City, ipInfo.Isp, config.DailyPrice)

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "Markdown"
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛒 Beli Akun Premium", "menu_create"),
		),
	)

	// Add Admin Panel for Admin
	if chatID == config.AdminID {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 System Info", "menu_info"),
		))
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛠️ Admin Panel", "menu_admin"),
		))
	}

	msg.ReplyMarkup = keyboard
	sendAndTrack(bot, msg)
}

func sendAccountInfo(bot *tgbotapi.BotAPI, chatID int64, data map[string]interface{}, config *BotConfig) {
	ipInfo, _ := getIpInfo()
	domain := config.Domain
	if domain == "" {
		domain = "(Not Configured)"
	}

	msg := fmt.Sprintf("```\n━━━━━━━━━━━━━━━━━━━━━\n  PREMIUM ACCOUNT\n━━━━━━━━━━━━━━━━━━━━━\nPassword   : %s\nCITY       : %s\nISP        : %s\nDomain     : %s\nExpired On : %s\n━━━━━━━━━━━━━━━━━━━━━\n```\nTerima kasih telah berlangganan!",
		data["password"], ipInfo.City, ipInfo.Isp, domain, data["expired"],
	)

	reply := tgbotapi.NewMessage(chatID, msg)
	reply.ParseMode = "Markdown"
	deleteLastMessage(bot, chatID)
	bot.Send(reply)
	showMainMenu(bot, chatID, config)
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, inState := userStates[chatID]; inState {
		cancelKb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "cancel")),
		)
		msg.ReplyMarkup = cancelKb
	}
	sendAndTrack(bot, msg)
}

func replyError(bot *tgbotapi.BotAPI, chatID int64, text string) {
	sendMessage(bot, chatID, "❌ "+text)
}

func cancelOperation(bot *tgbotapi.BotAPI, chatID int64, userID int64, config *BotConfig) {
	resetState(userID)
	showMainMenu(bot, chatID, config)
}

func sendAndTrack(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) {
	deleteLastMessage(bot, msg.ChatID)
	sentMsg, err := bot.Send(msg)
	if err == nil {
		lastMessageIDs[msg.ChatID] = sentMsg.MessageID
	}
}

func deleteLastMessage(bot *tgbotapi.BotAPI, chatID int64) {
	if msgID, ok := lastMessageIDs[chatID]; ok {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, msgID)
		bot.Request(deleteMsg)
		delete(lastMessageIDs, chatID)
	}
}

func resetState(userID int64) {
	delete(userStates, userID)
	// Don't delete tempUserData immediately if pending payment, but here we do for cancel
}

func validatePassword(bot *tgbotapi.BotAPI, chatID int64, text string) bool {
	if len(text) < 3 || len(text) > 20 {
		sendMessage(bot, chatID, "❌ Password harus 3-20 karakter. Coba lagi:")
		return false
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(text) {
		sendMessage(bot, chatID, "❌ Password hanya boleh huruf, angka, - dan _. Coba lagi:")
		return false
	}
	return true
}

func validateNumber(bot *tgbotapi.BotAPI, chatID int64, text string, min, max int, fieldName string) (int, bool) {
	val, err := strconv.Atoi(text)
	if err != nil || val < min || val > max {
		sendMessage(bot, chatID, fmt.Sprintf("❌ %s harus angka positif (%d-%d). Coba lagi:", fieldName, min, max))
		return 0, false
	}
	return val, true
}

func systemInfo(bot *tgbotapi.BotAPI, chatID int64, config *BotConfig) {
	res, err := apiCall("GET", "/info", nil)
	if err != nil {
		replyError(bot, chatID, "Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		data := res["data"].(map[string]interface{})
		ipInfo, _ := getIpInfo()

		msg := fmt.Sprintf("```\n━━━━━━━━━━━━━━━━━━━━━\n    INFO ZIVPN UDP\n━━━━━━━━━━━━━━━━━━━━━\nDomain         : %s\nIP Public      : %s\nPort           : %s\nService        : %s\nCITY           : %s\nISP            : %s\n━━━━━━━━━━━━━━━━━━━━━\n```",
			config.Domain, data["public_ip"], data["port"], data["service"], ipInfo.City, ipInfo.Isp)

		reply := tgbotapi.NewMessage(chatID, msg)
		reply.ParseMode = "Markdown"
		deleteLastMessage(bot, chatID)
		bot.Send(reply)
		showMainMenu(bot, chatID, config)
	} else {
		replyError(bot, chatID, "Gagal mengambil info.")
	}
}

func showBackupRestoreMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🛠️ *Admin Panel*\nSilakan pilih menu:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬇️ Backup Data", "menu_backup_action"),
			tgbotapi.NewInlineKeyboardButtonData("⬆️ Restore Data", "menu_restore_action"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Kembali", "cancel"),
		),
	)
	sendAndTrack(bot, msg)
}

func performBackup(bot *tgbotapi.BotAPI, chatID int64) {
	sendMessage(bot, chatID, "⏳ Sedang membuat backup...")

	// Files to backup
	files := []string{
		"/etc/zivpn/config.json",
		"/etc/zivpn/users.json",
		"/etc/zivpn/domain",
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			continue
		}
		defer f.Close()

		w, err := zipWriter.Create(filepath.Base(file))
		if err != nil {
			continue
		}

		if _, err := io.Copy(w, f); err != nil {
			continue
		}
	}

	zipWriter.Close()

	fileName := fmt.Sprintf("zivpn-backup-%s.zip", time.Now().Format("20060102-150405"))
	
	// Create a temporary file for the upload
	tmpFile := "/tmp/" + fileName
	if err := ioutil.WriteFile(tmpFile, buf.Bytes(), 0644); err != nil {
		replyError(bot, chatID, "Gagal membuat file backup.")
		return
	}
	defer os.Remove(tmpFile)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(tmpFile))
	doc.Caption = "✅ Backup Data ZiVPN"
	
	deleteLastMessage(bot, chatID)
	bot.Send(doc)
}

func startRestore(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	userStates[userID] = "waiting_restore_file"
	sendMessage(bot, chatID, "⬆️ *Restore Data*\n\nSilakan kirim file ZIP backup Anda sekarang.\n\n⚠️ PERINGATAN: Data saat ini akan ditimpa!")
}

func processRestoreFile(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, config *BotConfig) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	
	resetState(userID)
	sendMessage(bot, chatID, "⏳ Sedang memproses file...")

	// Download file
	fileID := msg.Document.FileID
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		replyError(bot, chatID, "Gagal mengunduh file.")
		return
	}

	fileUrl := file.Link(config.BotToken)
	resp, err := http.Get(fileUrl)
	if err != nil {
		replyError(bot, chatID, "Gagal mengunduh file content.")
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		replyError(bot, chatID, "Gagal membaca file.")
		return
	}

	// Unzip
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		replyError(bot, chatID, "File bukan format ZIP yang valid.")
		return
	}

	for _, f := range zipReader.File {
		// Security check: only allow specific files
		validFiles := map[string]bool{
			"config.json": true,
			"users.json": true,
			"bot-config.json": true,
			"domain": true,
			"apikey": true,
		}
		
		if !validFiles[f.Name] {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		defer rc.Close()

		dstPath := filepath.Join("/etc/zivpn", f.Name)
		dst, err := os.Create(dstPath)
		if err != nil {
			continue
		}
		defer dst.Close()

		io.Copy(dst, rc)
	}

	// Restart Services
	exec.Command("systemctl", "restart", "zivpn").Run()
	exec.Command("systemctl", "restart", "zivpn-api").Run()
	
	msgSuccess := tgbotapi.NewMessage(chatID, "✅ Restore Berhasil!\nService ZiVPN, API, dan Bot telah direstart.")
	bot.Send(msgSuccess)

	// Restart Bot with delay to allow message sending
	go func() {
		time.Sleep(2 * time.Second)
		exec.Command("systemctl", "restart", "zivpn-bot").Run()
	}()

	showMainMenu(bot, chatID, config)
}

func loadConfig() (BotConfig, error) {
	var config BotConfig
	file, err := ioutil.ReadFile(BotConfigFile)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config)

	if config.Domain == "" {
		if domainBytes, err := ioutil.ReadFile(DomainFile); err == nil {
			config.Domain = strings.TrimSpace(string(domainBytes))
		}
	}

	return config, err
}

func apiCall(method, endpoint string, payload interface{}) (map[string]interface{}, error) {
	var reqBody []byte
	var err error

	if payload != nil {
		reqBody, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, ApiUrl+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", ApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	return result, nil
}

func getIpInfo() (IpInfo, error) {
	resp, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return IpInfo{}, err
	}
	defer resp.Body.Close()

	var info IpInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return IpInfo{}, err
	}
	return info, nil
}
