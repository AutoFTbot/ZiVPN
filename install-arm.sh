#!/bin/bash

# ╔════════════════════════════════════════════════════════════╗
# ║   🚀 INSTALLER MODUL UDP ZIVPN - ARM                      ║
# ║   👤 Penulis: Zahid Islam                                  ║
# ║   👤 Remasterisasi: AutoFTbot                              ║
# ║   💡 Versi untuk sistem ARM64                              ║
# ╚════════════════════════════════════════════════════════════╝

# 🎨 Warna
GREEN="\e[32m"
YELLOW="\e[33m"
CYAN="\e[36m"
RED="\e[31m"
RESET="\e[0m"

# ╔════════════════════════════════════════════════════════════╗
# ║  📦 MEMPERBARUI SISTEM                                     ║
# ╚════════════════════════════════════════════════════════════╝
echo -e "${CYAN}🔄 Memperbarui paket sistem...${RESET}"
sudo apt-get update && sudo apt-get upgrade -y

# ╔════════════════════════════════════════════════════════════╗
# ║  🌐 KONFIGURASI DOMAIN                                     ║
# ╚════════════════════════════════════════════════════════════╝
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

# ╔════════════════════════════════════════════════════════════╗
# ║  ⬇️ MENGUNDUH ZIVPN UNTUK ARM64                            ║
# ╚════════════════════════════════════════════════════════════╝
echo -e "${CYAN}📥 Mengunduh binary ARM64 ZIVPN...${RESET}"
systemctl stop zivpn.service &>/dev/null
wget -q https://github.com/zahidbd2/udp-zivpn/releases/download/udp-zivpn_1.4.9/udp-zivpn-linux-arm64 -O /usr/local/bin/zivpn
chmod +x /usr/local/bin/zivpn

echo -e "${CYAN}📁 Menyiapkan direktori konfigurasi...${RESET}"
mkdir -p /etc/zivpn
echo "$domain" > /etc/zivpn/domain
wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/config.json -O /etc/zivpn/config.json

# ╔════════════════════════════════════════════════════════════╗
# ║  🔐 MEMBUAT SERTIFIKAT SSL                                 ║
# ╚════════════════════════════════════════════════════════════╝
echo -e "${CYAN}🔐 Membuat sertifikat SSL untuk ${YELLOW}$domain${CYAN}...${RESET}"
openssl req -new -newkey rsa:4096 -days 365 -nodes -x509 \
-subj "/C=ID/ST=Jawa Barat/L=Bandung/O=AutoFTbot/OU=IT Department/CN=$domain" \
-keyout "/etc/zivpn/zivpn.key" -out "/etc/zivpn/zivpn.crt"

# ╔════════════════════════════════════════════════════════════╗
# ║  ⚙️ MENGOPTIMALKAN PARAMETER SISTEM                        ║
# ╚════════════════════════════════════════════════════════════╝
sysctl -w net.core.rmem_max=16777216 &>/dev/null
sysctl -w net.core.wmem_max=16777216 &>/dev/null

# ╔════════════════════════════════════════════════════════════╗
# ║  🧩 MEMBUAT LAYANAN SYSTEMD                                ║
# ╚════════════════════════════════════════════════════════════╝
echo -e "${CYAN}🔧 Mengonfigurasi layanan systemd...${RESET}"
cat <<EOF > /etc/systemd/system/zivpn.service
[Unit]
Description=ZIVPN UDP VPN Server (ARM)
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

# ╔════════════════════════════════════════════════════════════╗
# ║  🚀 MEMULAI DAN MENGAKTIFKAN LAYANAN                       ║
# ╚════════════════════════════════════════════════════════════╝
systemctl enable zivpn.service
systemctl start zivpn.service

# ╔════════════════════════════════════════════════════════════╗
# ║  🌐 MENGONFIGURASI IPTABLES DAN FIREWALL                   ║
# ╚════════════════════════════════════════════════════════════╝
iface=$(ip -4 route ls | grep default | grep -Po '(?<=dev )(\S+)' | head -1)
iptables -t nat -A PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667
ufw allow 6000:19999/udp
ufw allow 5667/udp

# ╔════════════════════════════════════════════════════════════╗
# ║  ✅ SELESAI                                                ║
# ╚════════════════════════════════════════════════════════════╝
rm -f zi2.* &>/dev/null
echo -e "${GREEN}✅ ZIVPN (ARM) berhasil diinstal.${RESET}"
echo -e "${GREEN}🔰 Domain terkonfigurasi: ${YELLOW}$domain${RESET}"
echo -e "${GREEN}🔰 Pembuatan user dilakukan tanpa UI, langsung menggunakan API Golang.${RESET}"
echo -e "${GREEN}📄 Silakan cek dokumentasi Postman di repository AutoFTbot/ZiVPN.${RESET}"
