#!/bin/bash

# ╔════════════════════════════════════════════════════════════════════╗
# ║       🚀 INSTALLER MODUL UDP ZIVPN                                     ║
# ║       👤 Penulis: Zahid Islam                                          ║
# ║       👤 Remasterisasi: AutoFTbot                                      ║
# ║       🛠️ Menginstal dan mengonfigurasi layanan UDP ZIVPN               ║
# ╚════════════════════════════════════════════════════════════════════╝

# Warna untuk presentasi
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RED="\033[1;31m"
MAGENTA="\033[1;35m"
RESET="\033[0m"

# Fungsi untuk mencetak bagian dengan bingkai
print_section() {
  local title="$1"
  echo -e "${MAGENTA}╔════════════════════════════════════════════════════════════════╗${RESET}"
  echo -e "${MAGENTA}║  $title${RESET}$(printf ' %.0s' {1..$(($(tput cols)-${#title}-4))})${MAGENTA}║${RESET}"
  echo -e "${MAGENTA}╚════════════════════════════════════════════════════════════════╝${RESET}"
}

# ╔════════════════════════════════════════════════════════════════╗
print_section "🔍 MEMERIKSA KOMPATIBILITAS SISTEM"

# Cek OS Linux
if [[ "$(uname -s)" != "Linux" ]]; then
  echo -e "${RED}❌ Error: Script ini hanya berjalan di Linux.${RESET}"
  exit 1
fi

# Cek Arsitektur AMD64
ARCH=$(uname -m)
if [[ "$ARCH" != "x86_64" ]]; then
  echo -e "${RED}❌ Error: Arsitektur '$ARCH' tidak didukung.${RESET}"
  echo -e "${YELLOW}⚠️  Installer ini khusus untuk AMD64 (x86_64).${RESET}"
  exit 1
fi

echo -e "${GREEN}✅ Sistem kompatibel: Linux AMD64${RESET}"

# Fungsi untuk menampilkan spinner dan menangani error
run_with_spinner() {
  local msg="$1"
  local cmd="$2"

  echo -ne "${CYAN}${msg}...${RESET}"
  bash -c "$cmd" &>/tmp/zivpn_spinner.log &
  local pid=$!

  local delay=0.1
  local spinstr='|/-\'
  while kill -0 $pid 2>/dev/null; do
    local temp=${spinstr#?}
    printf " [%c]  " "$spinstr"
    local spinstr=$temp${spinstr%"$temp"}
    sleep $delay
    printf "\b\b\b\b\b\b"
  done
  wait $pid
  local exit_code=$?

  if [ $exit_code -eq 0 ]; then
    echo -e " ${GREEN}✔️${RESET}"
  else
    echo -e " ${RED}❌ Error${RESET}"
    echo -e "${RED}🛑 Terjadi kesalahan saat menjalankan:${RESET} ${YELLOW}$msg${RESET}"
    echo -e "${RED}📄 Detail kesalahan:${RESET}"
    cat /tmp/zivpn_spinner.log
    exit 1
  fi
  rm -f /tmp/zivpn_spinner.log
}

# ╔════════════════════════════════════════════════════════════════╗
print_section "🔍 MEMERIKSA INSTALASI ZIVPN UDP SEBELUMNYA"
if [ -f /usr/local/bin/zivpn ] || [ -f /etc/systemd/system/zivpn.service ]; then
  echo -e "${YELLOW}⚠️  ZIVPN UDP tampaknya sudah terinstal di sistem ini.${RESET}"
  echo -e "${YELLOW}Demi keamanan, instalasi akan dihentikan untuk menghindari penimpaan.${RESET}"
  exit 1
fi

# ╔════════════════════════════════════════════════════════════════╗
print_section "📦 MEMPERBARUI SISTEM & INSTAL GOLANG"
run_with_spinner "🔄 Memperbarui paket sistem" "sudo apt-get update"

if ! command -v go &> /dev/null; then
    run_with_spinner "🐹 Menginstal Golang" "sudo apt-get install -y golang git"
else
    echo -e "${GREEN}✅ Golang sudah terinstal.${RESET}"
fi

# ╔════════════════════════════════════════════════════════════════╗
print_section "🌐 KONFIGURASI DOMAIN"
echo -e "${YELLOW}⚠️  Domain diperlukan untuk konfigurasi sertifikat.${RESET}"
while true; do
  read -p "📌 Masukkan Domain/Host (contoh: vpn.domain.com): " domain
  if [[ -z "$domain" ]]; then
    echo -e "${RED}❌ Domain tidak boleh kosong.${RESET}"
  else
    echo -e "${GREEN}✅ Domain diset ke: $domain${RESET}"
    break
  fi
done

# ╔════════════════════════════════════════════════════════════════╗
print_section "⬇️ MENGUNDUH ZIVPN UDP"
echo -e "${CYAN}📥 Mengunduh binary ZIVPN...${RESET}"
systemctl stop zivpn.service &>/dev/null
wget -q https://github.com/zahidbd2/udp-zivpn/releases/download/udp-zivpn_1.4.9/udp-zivpn-linux-amd64 -O /usr/local/bin/zivpn
chmod +x /usr/local/bin/zivpn

echo -e "${CYAN}📁 Menyiapkan konfigurasi...${RESET}"
mkdir -p /etc/zivpn
echo "$domain" > /etc/zivpn/domain
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/config.json -O /etc/zivpn/config.json

# ╔════════════════════════════════════════════════════════════════╗
print_section "🔐 MEMBUAT SERTIFIKAT SSL"
echo -e "${CYAN}🔐 Membuat sertifikat SSL untuk ${YELLOW}$domain${CYAN}...${RESET}"
run_with_spinner "🔐 Generating SSL" "openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 -subj '/C=ID/ST=Jawa Barat/L=Bandung/O=AutoFTbot/OU=IT Department/CN=$domain' -keyout /etc/zivpn/zivpn.key -out /etc/zivpn/zivpn.crt"

# ╔════════════════════════════════════════════════════════════════╗
print_section "⚙️ MENGOPTIMALKAN PARAMETER SISTEM"
sysctl -w net.core.rmem_max=16777216 &>/dev/null
sysctl -w net.core.wmem_max=16777216 &>/dev/null

# ╔════════════════════════════════════════════════════════════════╗
print_section "🧩 MEMBUAT LAYANAN SYSTEMD (VPN)"
if [ -f /etc/systemd/system/zivpn.service ]; then
    echo -e "${YELLOW}⚠️ Layanan ZIVPN sudah ada. Pembuatan akan dilewati.${RESET}"
else
    echo -e "${CYAN}🔧 Mengonfigurasi layanan systemd...${RESET}"
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
fi

# ╔════════════════════════════════════════════════════════════════╗
print_section "🐹 MENYIAPKAN API GOLANG"
echo -e "${CYAN}📥 Mengunduh source code API...${RESET}"
mkdir -p /etc/zivpn/api
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/zivpn-api.go -O /etc/zivpn/api/zivpn-api.go
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/go.mod -O /etc/zivpn/api/go.mod

echo -e "${CYAN}🔨 Mengompilasi API...${RESET}"
cd /etc/zivpn/api
if go build -o zivpn-api zivpn-api.go; then
    echo -e "${GREEN}✅ API berhasil dikompilasi.${RESET}"
else
    echo -e "${RED}❌ Gagal mengompilasi API. Pastikan Golang terinstal dengan benar.${RESET}"
fi

echo -e "${CYAN}🔧 Membuat layanan systemd untuk API...${RESET}"
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

# ╔════════════════════════════════════════════════════════════════╗
print_section "🚀 MEMULAI DAN MENGAKTIFKAN LAYANAN"
systemctl enable zivpn.service
systemctl start zivpn.service
systemctl enable zivpn-api.service
systemctl start zivpn-api.service

# ╔════════════════════════════════════════════════════════════════╗
print_section "🌐 MENGONFIGURASI IPTABLES DAN FIREWALL"
iface=$(ip -4 route ls | grep default | grep -Po '(?<=dev )(\S+)' | head -1)
if ! iptables -t nat -C PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667 &>/dev/null; then
    iptables -t nat -A PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667
else
    echo -e "${YELLOW}⚠️ Aturan iptables sudah ada. Penambahan dilewati.${RESET}"
fi

ufw allow 6000:19999/udp
ufw allow 5667/udp
ufw allow 8080/tcp

# ╔════════════════════════════════════════════════════════════════╗
print_section "✅ SELESAI"
rm -f install-amd.sh install-amd.tmp install-amd.log &>/dev/null
echo -e "${GREEN}✅ ZIVPN UDP & API berhasil diinstal.${RESET}"
echo -e "${GREEN}🔰 Domain terkonfigurasi: ${YELLOW}$domain${RESET}"
echo -e "${GREEN}🐹 API Golang berjalan di port 8080.${RESET}"
echo -e "${GREEN}📄 Silakan cek dokumentasi Postman di repository AutoFTbot/ZiVPN.${RESET}"
