#!/bin/bash

# ╔══════════════════════════════════════════════════════════════════════╗
# ║       ❌  UNINSTALLER UDP ZIVPN                                      ║
# ║       🧽 Pembersihan sistem lengkap                                  ║
# ║       👤 Penulis: Zahid Islam / Diadaptasi oleh AutoFTbot            ║
# ╚══════════════════════════════════════════════════════════════════════╝

# 🎨 Warna
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RED="\033[1;31m"
MAGENTA="\033[1;35m"
RESET="\033[0m"

# Fungsi untuk mencetak bagian
print_section() {
  local title="$1"
  echo -e "${MAGENTA}╔════════════════════════════════════════════════════════════════════╗${RESET}"
  printf "${MAGENTA}║ %-66s ║\n" "$title"
  echo -e "${MAGENTA}╚════════════════════════════════════════════════════════════════════╝${RESET}"
}

clear
print_section "🧹 MEMULAI UNINSTALL ZiVPN"

# ╔════════════════════════════════════════════════════════════════════╗
print_section "🛑 MENGHENTIKAN LAYANAN"
systemctl stop zivpn.service &>/dev/null
systemctl stop zivpn-api.service &>/dev/null
systemctl stop zivpn_backfill.service &>/dev/null
systemctl disable zivpn.service &>/dev/null
systemctl disable zivpn-api.service &>/dev/null
systemctl disable zivpn_backfill.service &>/dev/null

# ╔════════════════════════════════════════════════════════════════════╗
print_section "🧽 MENGHAPUS BINARY DAN FILE KONFIGURASI"
rm -f /etc/systemd/system/zivpn.service
rm -f /etc/systemd/system/zivpn-api.service
rm -f /etc/systemd/system/zivpn_backfill.service
rm -rf /etc/zivpn
rm -f /usr/local/bin/zivpn
killall zivpn &>/dev/null
killall zivpn-api &>/dev/null

# ╔════════════════════════════════════════════════════════════════════╗
print_section "🔥 MENGHAPUS ATURAN IPTABLES"
iface=$(ip -4 route ls | grep default | grep -Po '(?<=dev )(\S+)' | head -1)
iptables -t nat -D PREROUTING -i "$iface" -p udp --dport 6000:19999 -j DNAT --to-destination :5667 &>/dev/null

# ╔════════════════════════════════════════════════════════════════════╗
print_section "🗑️ MENGHAPUS INDIKATOR DAN PERBAIKAN"
rm -f /etc/zivpn-iptables-fix-applied

# ╔════════════════════════════════════════════════════════════════════╗
print_section "📋 MEMERIKSA STATUS AKHIR"
if pgrep "zivpn" &>/dev/null; then
  echo -e "${RED}⚠️  Proses ZIVPN masih aktif.${RESET}"
else
  echo -e "${GREEN}✅ Proses ZIVPN berhasil dihentikan.${RESET}"
fi

if pgrep "zivpn-api" &>/dev/null; then
  echo -e "${RED}⚠️  Proses API masih aktif.${RESET}"
else
  echo -e "${GREEN}✅ Proses API berhasil dihentikan.${RESET}"
fi

if [ -e "/usr/local/bin/zivpn" ]; then
  echo -e "${YELLOW}⚠️  Binary masih ada. Coba lagi.${RESET}"
else
  echo -e "${GREEN}✅ Binary berhasil dihapus.${RESET}"
fi

# ╔════════════════════════════════════════════════════════════════════╗
print_section "🧼 MEMBERSIHKAN CACHE DAN SWAP"
echo 3 > /proc/sys/vm/drop_caches
sysctl -w vm.drop_caches=3 &>/dev/null
swapoff -a && swapon -a

# ╔════════════════════════════════════════════════════════════════════╗
print_section "🏁 SELESAI"
echo -e "${GREEN}✅ UDP ZiVPN dan API telah berhasil di-uninstall.${RESET}"
