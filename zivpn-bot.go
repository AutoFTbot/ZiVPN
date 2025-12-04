package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BotConfigFile = "/etc/zivpn/bot-config.json"
	ApiUrl        = "http://127.0.0.1:8080/api"
	ApiKeyFile    = "/etc/zivpn/apikey"
)

var ApiKey = "AutoFtBot-agskjgdvsbdreiWG1234512SDKrqw"

type BotConfig struct {
	BotToken string `json:"bot_token"`
	AdminID  int64  `json:"admin_id"`
}

var userStates = make(map[int64]string)
var tempUserData = make(map[int64]map[string]string)

func main() {
	if keyBytes, err := ioutil.ReadFile(ApiKeyFile); err == nil {
		ApiKey = strings.TrimSpace(string(keyBytes))
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

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message, config.AdminID)
		} else if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery, config.AdminID)
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, adminID int64) {
	if msg.From.ID != adminID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⛔ Akses Ditolak. Anda bukan admin.")
		bot.Send(reply)
		return
	}

	state, exists := userStates[msg.From.ID]
	if exists {
		handleState(bot, msg, state)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			showMainMenu(bot, msg.Chat.ID)
		default:
			msg := tgbotapi.NewMessage(msg.Chat.ID, "Perintah tidak dikenal.")
			bot.Send(msg)
		}
	}
}

func handleCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, adminID int64) {
	if query.From.ID != adminID {
		bot.Request(tgbotapi.NewCallback(query.ID, "Akses Ditolak"))
		return
	}

	switch query.Data {
	case "menu_create":
		userStates[query.From.ID] = "create_username"
		tempUserData[query.From.ID] = make(map[string]string)
		sendMessage(bot, query.Message.Chat.ID, "👤 Masukkan Username:")
	case "menu_delete":
		userStates[query.From.ID] = "delete_username"
		sendMessage(bot, query.Message.Chat.ID, "🗑️ Masukkan Username yang akan dihapus:")
	case "menu_renew":
		userStates[query.From.ID] = "renew_username"
		tempUserData[query.From.ID] = make(map[string]string)
		sendMessage(bot, query.Message.Chat.ID, "🔄 Masukkan Username yang akan diperpanjang:")
	case "menu_list":
		listUsers(bot, query.Message.Chat.ID)
	case "menu_info":
		systemInfo(bot, query.Message.Chat.ID)
	case "cancel":
		delete(userStates, query.From.ID)
		delete(tempUserData, query.From.ID)
		showMainMenu(bot, query.Message.Chat.ID)
	}

	bot.Request(tgbotapi.NewCallback(query.ID, ""))
}

func handleState(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, state string) {
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)

	switch state {
	case "create_username":
		tempUserData[userID]["username"] = text
		userStates[userID] = "create_days"
		sendMessage(bot, msg.Chat.ID, "⏳ Masukkan Durasi (hari):")

	case "create_days":
		days, err := strconv.Atoi(text)
		if err != nil {
			sendMessage(bot, msg.Chat.ID, "❌ Durasi harus angka. Coba lagi:")
			return
		}
		createUser(bot, msg.Chat.ID, tempUserData[userID]["username"], days)
		resetState(userID)

	case "delete_username":
		deleteUser(bot, msg.Chat.ID, text)
		resetState(userID)

	case "renew_username":
		tempUserData[userID]["username"] = text
		userStates[userID] = "renew_days"
		sendMessage(bot, msg.Chat.ID, "⏳ Masukkan Tambahan Durasi (hari):")

	case "renew_days":
		days, err := strconv.Atoi(text)
		if err != nil {
			sendMessage(bot, msg.Chat.ID, "❌ Durasi harus angka. Coba lagi:")
			return
		}
		renewUser(bot, msg.Chat.ID, tempUserData[userID]["username"], days)
		resetState(userID)
	}
}

func showMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🚀 *ZiVPN UDP Manager*\nSilakan pilih menu:")
	msg.ParseMode = "Markdown"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Create User", "menu_create"),
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete User", "menu_delete"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Renew User", "menu_renew"),
			tgbotapi.NewInlineKeyboardButtonData("📋 List Users", "menu_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 System Info", "menu_info"),
		),
	)
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, inState := userStates[chatID]; inState {
		cancelKb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "cancel")),
		)
		msg.ReplyMarkup = cancelKb
	}
	bot.Send(msg)
}

func resetState(userID int64) {
	delete(userStates, userID)
	delete(tempUserData, userID)
}

// --- API Calls ---

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

func createUser(bot *tgbotapi.BotAPI, chatID int64, username string, days int) {
	res, err := apiCall("POST", "/user/create", map[string]interface{}{
		"password": username,
		"days":     days,
	})

	if err != nil {
		sendMessage(bot, chatID, "❌ Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		data := res["data"].(map[string]interface{})
		msg := fmt.Sprintf("✅ *User Created*\n\n👤 Username: `%s`\n📅 Expired: `%s`\n🌐 Domain: `%s`",
			data["password"], data["expired"], data["domain"])
		
		reply := tgbotapi.NewMessage(chatID, msg)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
		showMainMenu(bot, chatID)
	} else {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Gagal: %s", res["message"]))
		showMainMenu(bot, chatID)
	}
}

func deleteUser(bot *tgbotapi.BotAPI, chatID int64, username string) {
	res, err := apiCall("POST", "/user/delete", map[string]interface{}{
		"password": username,
	})

	if err != nil {
		sendMessage(bot, chatID, "❌ Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		sendMessage(bot, chatID, "✅ User berhasil dihapus.")
		showMainMenu(bot, chatID)
	} else {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Gagal: %s", res["message"]))
		showMainMenu(bot, chatID)
	}
}

func renewUser(bot *tgbotapi.BotAPI, chatID int64, username string, days int) {
	res, err := apiCall("POST", "/user/renew", map[string]interface{}{
		"password": username,
		"days":     days,
	})

	if err != nil {
		sendMessage(bot, chatID, "❌ Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		data := res["data"].(map[string]interface{})
		msg := fmt.Sprintf("✅ *User Renewed*\n\n👤 Username: `%s`\n📅 New Expired: `%s`",
			data["password"], data["expired"])
		
		reply := tgbotapi.NewMessage(chatID, msg)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
		showMainMenu(bot, chatID)
	} else {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Gagal: %s", res["message"]))
		showMainMenu(bot, chatID)
	}
}

func listUsers(bot *tgbotapi.BotAPI, chatID int64) {
	res, err := apiCall("GET", "/users", nil)
	if err != nil {
		sendMessage(bot, chatID, "❌ Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		users := res["data"].([]interface{})
		if len(users) == 0 {
			sendMessage(bot, chatID, "📂 Tidak ada user.")
			return
		}

		msg := "📋 *List Users*\n"
		for _, u := range users {
			user := u.(map[string]interface{})
			status := "🟢"
			if user["status"] == "Expired" {
				status = "🔴"
			}
			msg += fmt.Sprintf("\n%s `%s` (%s)", status, user["password"], user["expired"])
		}
		
		reply := tgbotapi.NewMessage(chatID, msg)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	} else {
		sendMessage(bot, chatID, "❌ Gagal mengambil data.")
	}
}

func systemInfo(bot *tgbotapi.BotAPI, chatID int64) {
	res, err := apiCall("GET", "/info", nil)
	if err != nil {
		sendMessage(bot, chatID, "❌ Error API: "+err.Error())
		return
	}

	if res["success"] == true {
		data := res["data"].(map[string]interface{})
		msg := fmt.Sprintf("📊 *System Info*\n\n🌐 Domain: `%s`\n🖥️ IP Public: `%s`\n🔒 IP Private: `%s`\n🔌 Port: `%s`\n⚙️ Service: `%s`",
			data["domain"], data["public_ip"], data["private_ip"], data["port"], data["service"])
		
		reply := tgbotapi.NewMessage(chatID, msg)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	} else {
		sendMessage(bot, chatID, "❌ Gagal mengambil info.")
	}
}

func loadConfig() (BotConfig, error) {
	var config BotConfig
	file, err := ioutil.ReadFile(BotConfigFile)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config)
	return config, err
}
