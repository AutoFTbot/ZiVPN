#!/bin/bash

# ╔══════════════════════════════════════════════════════════════════╗
# ║   🔐 PERBAIKAN ATURAN IPTABLES PERSISTEN UNTUK ZIVPN UDP TUNNEL  ║
# ║   👤 Penulis: AutoFTbot                                          ║
# ║   🛠️ Memperbaiki hilangnya aturan iptables setelah restart        ║
# ╚══════════════════════════════════════════════════════════════════╝

# 🎨 Warna
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RED="\033[1;31m"
RESET="\033[0m"

echo -e "${CYAN}🔍 Mendeteksi antarmuka jaringan...${RESET}"
iface=$(ip -4 route ls | grep default | grep -Po '(?<=dev )(\S+)' | head -1)

if [[ -z "$iface" ]]; then
  echo -e "${RED}❌ Tidak dapat mendeteksi antarmuka jaringan. Dibatalkan.${RESET}"
  exit 1
fi

echo -e "${CYAN}🌐 Antarmuka terdeteksi: ${YELLOW}$iface${RESET}"

# 📌 Terapkan aturan iptables jika belum ada
echo -e "${CYAN}🧪 Memeriksa aturan iptables untuk ZIVPN...${RESET}"
if iptables -t nat -C PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667 2>/dev/null; then
  echo -e "${YELLOW}⚠️ Aturan sudah ada. Tidak akan diterapkan lagi.${RESET}"
else
  echo -e "${GREEN}✅ Menambahkan aturan iptables untuk ZIVPN...${RESET}"
  iptables -t nat -A PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667
fi

# 🔥 Buka port dengan UFW jika tersedia
if command -v ufw &>/dev/null; then
  echo -e "${CYAN}🔓 Mengonfigurasi UFW...${RESET}"
  ufw allow 6000:19999/udp &>/dev/null
  ufw allow 5667/udp &>/dev/null
fi

# 📦 Instal iptables-persistent jika belum ada
if ! dpkg -s iptables-persistent &>/dev/null; then
  echo -e "${CYAN}📦 Menginstal iptables-persistent untuk mempertahankan aturan...${RESET}"
  echo iptables-persistent iptables-persistent/autosave_v4 boolean true | debconf-set-selections
  echo iptables-persistent iptables-persistent/autosave_v6 boolean true | debconf-set-selections
  apt-get install -y iptables-persistent &>/dev/null
fi

# 💾 Simpan aturan untuk restart
echo -e "${CYAN}💾 Menyimpan aturan untuk restart...${RESET}"
iptables-save > /etc/iptables/rules.v4

echo -e "${GREEN}✅ Aturan berhasil diterapkan dan disimpan.${RESET}"
