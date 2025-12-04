#!/bin/bash

# Colors
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RED="\033[1;31m"
BLUE="\033[1;34m"
RESET="\033[0m"
BOLD="\033[1m"
GRAY="\033[1;30m"

print_task() {
  echo -ne "${GRAY}•${RESET} $1..."
}

print_done() {
  echo -e "\r${GREEN}✓${RESET} $1      "
}

print_fail() {
  echo -e "\r${RED}✗${RESET} $1      "
  exit 1
}

run_silent() {
  local msg="$1"
  local cmd="$2"
  
  print_task "$msg"
  bash -c "$cmd" &>/tmp/zivpn_install.log
  if [ $? -eq 0 ]; then
    print_done "$msg"
  else
    print_fail "$msg (Check /tmp/zivpn_install.log)"
  fi
}

clear
echo -e "${BOLD}ZiVPN UDP Installer${RESET}"
echo -e "${GRAY}AutoFTbot Edition${RESET}"
echo ""

if [[ "$(uname -s)" != "Linux" ]] || [[ "$(uname -m)" != "x86_64" ]]; then
  print_fail "System not supported (Linux AMD64 only)"
fi

if [ -f /usr/local/bin/zivpn ]; then
  print_fail "ZiVPN already installed"
fi

run_silent "Updating system" "sudo apt-get update"

if ! command -v go &> /dev/null; then
  run_silent "Installing dependencies" "sudo apt-get install -y golang git"
else
  print_done "Dependencies ready"
fi

echo ""
echo -ne "${BOLD}Domain Configuration${RESET}\n"
while true; do
  read -p "Enter Domain: " domain
  if [[ -n "$domain" ]]; then
    break
  fi
done
echo ""

echo -ne "${BOLD}API Key Configuration${RESET}\n"
generated_key=$(openssl rand -hex 16)
echo -e "Generated Key: ${CYAN}$generated_key${RESET}"
read -p "Enter API Key (Press Enter to use generated): " input_key
if [[ -z "$input_key" ]]; then
  api_key="$generated_key"
else
  api_key="$input_key"
fi
echo -e "Using Key: ${GREEN}$api_key${RESET}"
echo ""

systemctl stop zivpn.service &>/dev/null
run_silent "Downloading Core" "wget -q https://github.com/zahidbd2/udp-zivpn/releases/download/udp-zivpn_1.4.9/udp-zivpn-linux-amd64 -O /usr/local/bin/zivpn && chmod +x /usr/local/bin/zivpn"

mkdir -p /etc/zivpn
echo "$domain" > /etc/zivpn/domain
echo "$api_key" > /etc/zivpn/apikey
run_silent "Configuring" "wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/config.json -O /etc/zivpn/config.json"

run_silent "Generating SSL" "openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 -subj '/C=ID/ST=Jawa Barat/L=Bandung/O=AutoFTbot/OU=IT Department/CN=$domain' -keyout /etc/zivpn/zivpn.key -out /etc/zivpn/zivpn.crt"

sysctl -w net.core.rmem_max=16777216 &>/dev/null
sysctl -w net.core.wmem_max=16777216 &>/dev/null

cat <<EOF > /etc/systemd/system/zivpn.service
[Unit]
Description=ZIVPN UDP VPN Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/etc/zivpn
ExecStart=/usr/local/bin/zivpn server -c /etc/zivpn/config.json
Restart=always
RestartSec=3
Environment=ZIVPN_LOG_LEVEL=info
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

# 11. API Setup
mkdir -p /etc/zivpn/api
run_silent "Setting up API" "wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/zivpn-api.go -O /etc/zivpn/api/zivpn-api.go && wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/go.mod -O /etc/zivpn/api/go.mod"

cd /etc/zivpn/api
if go build -o zivpn-api zivpn-api.go &>/dev/null; then
  print_done "Compiling API"
else
  print_fail "Compiling API"
fi

cat <<EOF > /etc/systemd/system/zivpn-api.service
[Unit]
Description=ZiVPN Golang API Service
After=network.target zivpn.service

[Service]
Type=simple
User=root
WorkingDirectory=/etc/zivpn/api
ExecStart=/etc/zivpn/api/zivpn-api
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

echo ""
echo -ne "${BOLD}Telegram Bot Configuration${RESET}\n"
echo -ne "${GRAY}(Leave empty to skip)${RESET}\n"
read -p "Bot Token: " bot_token
read -p "Admin ID : " admin_id

if [[ -n "$bot_token" ]] && [[ -n "$admin_id" ]]; then
  echo "{\"bot_token\": \"$bot_token\", \"admin_id\": $admin_id}" > /etc/zivpn/bot-config.json
  
  run_silent "Downloading Bot" "wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/zivpn-bot.go -O /etc/zivpn/api/zivpn-bot.go"
  
  cd /etc/zivpn/api
  run_silent "Downloading Bot Deps" "go get github.com/go-telegram-bot-api/telegram-bot-api/v5"
  
  if go build -o zivpn-bot zivpn-bot.go &>/dev/null; then
    print_done "Compiling Bot"
    
    cat <<EOF > /etc/systemd/system/zivpn-bot.service
[Unit]
Description=ZiVPN Telegram Bot
After=network.target zivpn-api.service

[Service]
Type=simple
User=root
WorkingDirectory=/etc/zivpn/api
ExecStart=/etc/zivpn/api/zivpn-bot
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
    systemctl enable zivpn-bot.service &>/dev/null
    systemctl start zivpn-bot.service &>/dev/null
  else
    print_fail "Compiling Bot"
  fi
else
  print_task "Skipping Bot Setup"
  echo ""
fi

run_silent "Starting Services" "systemctl enable zivpn.service && systemctl start zivpn.service && systemctl enable zivpn-api.service && systemctl start zivpn-api.service"

iface=$(ip -4 route ls | grep default | grep -Po '(?<=dev )(\S+)' | head -1)
iptables -t nat -A PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667 &>/dev/null
ufw allow 6000:19999/udp &>/dev/null
ufw allow 5667/udp &>/dev/null
ufw allow 8080/tcp &>/dev/null

rm -f install.sh install.tmp install.log &>/dev/null

echo ""
echo -e "${BOLD}Installation Complete${RESET}"
echo -e "Domain  : ${CYAN}$domain${RESET}"
echo -e "API     : ${CYAN}Port 8080${RESET}"
echo -e "Token   : ${CYAN}$api_key${RESET}"
echo ""
