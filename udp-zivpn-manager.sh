#!/bin/bash

# ╔════════════════════════════════════════════════════════════════════╗
# ║        🛡️ MANAJER TUNNEL UDP ZIVPN – DITINGKATKAN                 ║
# ╚════════════════════════════════════════════════════════════════════╝

# 🎨 Warna
RED="\033[1;31m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
BLUE="\033[1;34m"
CYAN="\033[1;36m"
RESET="\033[0m"

# 🧭 Deteksi Arsitektur
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
  ARCH_TEXT="AMD64"
elif [[ "$ARCH" == "aarch64" ]]; then
  ARCH_TEXT="ARM64"
else
  ARCH_TEXT="Tidak Diketahui"
fi

# ╔══════════════════════════════════════════════════╗
# ║ 🔍 FUNGSI: Tampilkan port yang digunakan zivpn    ║
# ╚══════════════════════════════════════════════════╝
mostrar_puertos_zivpn() {
  # Dapatkan PID proses zivpn jika sedang berjalan
  PID=$(pgrep -f /usr/local/bin/zivpn)
  if [[ -z "$PID" ]]; then
    echo -e " Port: ${RED}Tidak dapat mendeteksi proses zivpn.${RESET}"
    return
  fi

  # Gunakan ss jika tersedia
  if command -v ss &>/dev/null; then
    PUERTOS=$(ss -tulnp | grep "$PID" | awk '{print $5}' | cut -d':' -f2 | sort -u | tr '\n' ',' | sed 's/,$//')
  else
    # fallback ke netstat
    PUERTOS=$(netstat -tulnp 2>/dev/null | grep "$PID" | awk '{print $4}' | rev | cut -d':' -f1 | rev | sort -u | tr '\n' ',' | sed 's/,$//')
  fi

  if [[ -z "$PUERTOS" ]]; then
    echo -e " Port: ${YELLOW}Tidak ada port terbuka yang terdeteksi.${RESET}"
  else
    echo -e " Port: ${GREEN}$PUERTOS${RESET}"
  fi
}

# ╔══════════════════════════════════════════════════╗
# ║ 🔍 FUNGSI: Tampilkan port tetap dan iptables      ║
# ╚══════════════════════════════════════════════════╝
mostrar_puerto_iptables() {
  local PUERTO="5667"
  local IPTABLES="6000-19999"
  echo -e " ${YELLOW}📛 Port:${RESET} ${GREEN}$PUERTO${RESET}   ${RED}🔥 Iptables:${RESET} ${CYAN}$IPTABLES${RESET}"
}

# ╔══════════════════════════════════════════════════╗
# ║ 🔍 FUNGSI: Tampilkan status layanan ZIVPN         ║
# ╚══════════════════════════════════════════════════╝
mostrar_estado_servicio() {
  if [ -f /usr/local/bin/zivpn ] && [ -f /etc/systemd/system/zivpn.service ]; then
    systemctl is-active --quiet zivpn.service
    if [ $? -eq 0 ]; then
      echo -e " 🟢 Layanan UDP ZIVPN terinstal dan aktif"
      mostrar_puerto_iptables
    else
      echo -e " 🟡 Layanan UDP ZIVPN terinstal tetapi ${YELLOW}tidak aktif${RESET}"
      mostrar_puerto_iptables
    fi
  else
    echo -e " 🔴 Layanan UDP ZIVPN ${RED}tidak terinstal${RESET}"
  fi
}

# ╔══════════════════════════════════════════════════╗
# ║ 🔍 FUNGSI: Tampilkan status fix iptables          ║
# ╚══════════════════════════════════════════════════╝
mostrar_estado_fix() {
  if [ -f /etc/zivpn-iptables-fix-applied ]; then
    echo -e "${GREEN}[ON]${RESET}"
  else
    echo -e "${RED}[OFF]${RESET}"
  fi
}

# ╔══════════════════════════════════════════════════╗
# ║ 🌀 Spinner                                        ║
# ╚══════════════════════════════════════════════════╝
spinner() {
  local pid=$!
  local delay=0.1
  local spinstr='|/-\'
  while ps -p $pid &>/dev/null; do
    local temp=${spinstr#?}
    printf " [%c]  " "$spinstr"
    spinstr=$temp${spinstr%"$temp"}
    sleep $delay
    printf "\b\b\b\b\b\b"
  done
}

# ╔══════════════════════════════════════════════════╗
# ║ 📋 Menu Utama                                     ║
# ╚══════════════════════════════════════════════════╝
mostrar_menu() {
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -e "           🛠️ ${GREEN}MANAJER TUNNEL UDP ZIVPN${RESET}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"

  # Dapatkan Domain
  if [[ -f "/etc/zivpn/domain" ]]; then
    DOMAIN=$(cat "/etc/zivpn/domain")
  else
    DOMAIN="Tidak diatur"
  fi

  # Tampilkan arsitektur
  echo -e " 🔍 Arsitektur terdeteksi: ${YELLOW}$ARCH_TEXT${RESET}"
  echo -e " 🌐 Domain: ${YELLOW}$DOMAIN${RESET}"

  # Tampilkan status layanan
  mostrar_estado_servicio

  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -ne " ${YELLOW}1.${RESET} 🚀 Instal Layanan UDP (${BLUE}AMD64${RESET})\n"
  echo -ne " ${YELLOW}2.${RESET} 📦 Instal Layanan UDP (${GREEN}ARM64${RESET})\n"
  echo -ne " ${YELLOW}3.${RESET} ❌ Uninstall Layanan UDP\n"
  echo -ne " ${YELLOW}4.${RESET} 🔁 Terapkan Fix Iptables Persisten $(mostrar_estado_fix)\n"
  echo -ne " ${YELLOW}5.${RESET} 🔙 Keluar\n"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -ne "📤 ${BLUE}Pilih opsi:${RESET} "
}

# ╔══════════════════════════════════════════════════╗
# ║ 🚀 FUNGSI INSTALASI, UNINSTALL                    ║
# ╚══════════════════════════════════════════════════╝

instalar_amd() {
  clear
  echo -e "${GREEN}🚀 Mengunduh installer untuk AMD64...${RESET}"
  wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/install-amd.sh -O install-amd.sh &
  spinner
  if [[ ! -f install-amd.sh ]]; then
    echo -e "${RED}❌ Error: Gagal mengunduh file.${RESET}"
    read -p "Tekan Enter untuk melanjutkan..."
    return
  fi
  echo -e "${GREEN}🔧 Menjalankan instalasi...${RESET}"
  bash install-amd.sh
  rm -f install-amd.sh
  echo -e "${GREEN}✅ Instalasi selesai.${RESET}"
  read -p "Tekan Enter untuk melanjutkan..."
}

instalar_arm() {
  clear
  echo -e "${GREEN}📦 Mengunduh installer untuk ARM64...${RESET}"
  wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/install-arm.sh -O install-arm.sh &
  spinner
  if [[ ! -f install-arm.sh ]]; then
    echo -e "${RED}❌ Error: Gagal mengunduh file.${RESET}"
    read -p "Tekan Enter untuk melanjutkan..."
    return
  fi
  echo -e "${GREEN}🔧 Menjalankan instalasi...${RESET}"
  bash install-arm.sh
  rm -f install-arm.sh
  echo -e "${GREEN}✅ Instalasi selesai.${RESET}"
  read -p "Tekan Enter untuk melanjutkan..."
}

desinstalar_udp() {
  clear
  echo -e "${RED}🧹 Mengunduh script uninstall...${RESET}"
  wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/uninstall.sh -O uninstall.sh &
  spinner
  if [[ ! -f uninstall.sh ]]; then
    echo -e "${RED}❌ Error: Gagal mengunduh file.${RESET}"
    read -p "Tekan Enter untuk melanjutkan..."
    return
  fi
  echo -e "${RED}⚙️ Menjalankan uninstall...${RESET}"
  bash uninstall.sh
  rm -f uninstall.sh
  echo -e "${GREEN}✅ Uninstall selesai.${RESET}"
  read -p "Tekan Enter untuk melanjutkan..."
}

# ╔══════════════════════════════════════════════════╗
# ║ 🛠️ FUNGSI: Terapkan fix iptables persisten        ║
# ╚══════════════════════════════════════════════════╝
fix_iptables_zivpn() {
  clear
  echo -e "${CYAN}🔧 Menerapkan fix iptables persisten untuk ZIVPN...${RESET}"
  wget -q https://raw.githubusercontent.com/AutoFTbot/ZiVPN/main/zivpn-iptables-fix.sh -O zivpn-iptables-fix.sh
  if [[ ! -f zivpn-iptables-fix.sh ]]; then
    echo -e "${RED}❌ Error: Gagal mengunduh fix.${RESET}"
    read -p "Tekan Enter untuk melanjutkan..."
    return
  fi
  bash zivpn-iptables-fix.sh
  local res=$?
  rm -f zivpn-iptables-fix.sh
  if [[ $res -eq 0 ]]; then
    # Buat file indikator untuk ON
    touch /etc/zivpn-iptables-fix-applied 2>/dev/null || echo -e "${YELLOW}⚠️ Gagal membuat file indikator status.${RESET}"
    echo -e "${GREEN}✅ Fix berhasil diterapkan.${RESET}"
  else
    echo -e "${RED}❌ Terjadi kesalahan saat menerapkan fix.${RESET}"
  fi
  read -p "Tekan Enter untuk melanjutkan..."
}

# 🔁 Loop menu utama
while true; do
  clear
  mostrar_menu
  read -r opcion
  case $opcion in
    1) instalar_amd ;;
    2) instalar_arm ;;
    3) desinstalar_udp ;;
    4) fix_iptables_zivpn ;;
    5) echo -e "${YELLOW}👋 Sampai jumpa!${RESET}"; exit 0 ;;
    *) echo -e "${RED}❌ Opsi tidak valid. Coba lagi.${RESET}"; sleep 2 ;;
  esac
done
