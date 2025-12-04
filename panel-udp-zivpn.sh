#!/bin/bash

# ╔════════════════════════════════════════════════════════════════════════════╗
# ║                    🧩 ZIVPN - PANEL PENGGUNA UDP - v1.0                    ║
# ╚════════════════════════════════════════════════════════════════════════════╝

# 📁 File
CONFIG_FILE="/etc/zivpn/config.json"
USER_DB="/etc/zivpn/users.db"
CONF_FILE="/etc/zivpn.conf"
BACKUP_FILE="/etc/zivpn/config.json.bak"
DOMAIN_FILE="/etc/zivpn/domain"

# 🎨 Warna
RED="\033[1;31m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RESET="\033[0m"

# 🧽 Bersihkan layar
clear

# 🛠️ Dependensi
command -v jq >/dev/null 2>&1 || { echo -e "${RED}❌ jq tidak terinstal. Gunakan: apt install jq -y${RESET}"; exit 1; }

# 🧠 Buat file jika tidak ada
mkdir -p /etc/zivpn
[ ! -f "$CONFIG_FILE" ] && echo '{"listen":":5667","cert":"/etc/zivpn/zivpn.crt","key":"/etc/zivpn/zivpn.key","obfs":"zivpn","auth":{"mode":"passwords","config":["zivpn"]}}' > "$CONFIG_FILE"
[ ! -f "$USER_DB" ] && touch "$USER_DB"
[ ! -f "$CONF_FILE" ] && echo 'AUTOCLEAN=OFF' > "$CONF_FILE"

# 🔁 Muat konfigurasi
source "$CONF_FILE"

# 📦 Fungsi utama
add_user() {
  echo -e "${CYAN}⚠️  Masukkan '0' kapan saja untuk membatalkan.${RESET}"

  # Minta password dan validasi agar tidak kosong atau sudah ada
  while true; do
    read -p "🔐 Masukkan password baru: " pass

    if [[ "$pass" == "0" ]]; then
      echo -e "${YELLOW}⚠️  Pembuatan dibatalkan.${RESET}"
      return
    fi

    if [[ -z "$pass" ]]; then
      echo -e "${RED}❌ Password tidak boleh kosong.${RESET}"
      continue
    fi

    if jq -e --arg pw "$pass" '.auth.config | index($pw)' "$CONFIG_FILE" > /dev/null; then
      echo -e "${RED}❌ Password sudah ada.${RESET}"
      continue
    fi

    break
  done

  # Minta hari kedaluwarsa dan validasi angka positif
  while true; do
    read -p "📅 Hari kedaluwarsa: " days

    if [[ "$days" == "0" ]]; then
      echo -e "${YELLOW}⚠️  Pembuatan pengguna dibatalkan.${RESET}"
      return
    fi

    if [[ ! "$days" =~ ^[0-9]+$ ]] || [[ "$days" -le 0 ]]; then
      echo -e "${RED}❌ Masukkan angka positif yang valid.${RESET}"
      continue
    fi

    break
  done

  exp_date=$(date -d "+$days days" +%Y-%m-%d)

  # Buat backup sebelum memodifikasi
  cp "$CONFIG_FILE" "$BACKUP_FILE"

  # Tambahkan pengguna ke konfigurasi JSON
  jq --arg pw "$pass" '.auth.config += [$pw]' "$CONFIG_FILE" > temp && mv temp "$CONFIG_FILE"

  # Tambahkan pengguna ke database dengan format seragam
  echo "$pass | $exp_date" >> "$USER_DB"

  echo -e "${GREEN}✅ Pengguna ditambahkan dengan kedaluwarsa: $exp_date${RESET}"

  # Restart layanan untuk menerapkan perubahan
  systemctl restart zivpn.service

  # 🛑 Jeda untuk menampilkan hasil
  read -p "🔙 Tekan Enter untuk kembali ke menu..."
}

remove_user() {
  echo -e "${CYAN}🗂️ Daftar pengguna saat ini:${RESET}"
  list_users
  
  echo -e "\n🔢 Masukkan ID pengguna yang akan dihapus (0 untuk membatalkan)."
  
  while true; do
    read -p "➡️ Pilihan: " id
    
    if [[ "$id" == "0" ]]; then
      echo -e "${YELLOW}⚠️ Penghapusan dibatalkan.${RESET}"
      read -p "🔙 Tekan Enter untuk kembali ke menu..."
      return
    fi
    
    # Validasi angka dan dalam rentang
    if ! [[ "$id" =~ ^[0-9]+$ ]]; then
      echo -e "${RED}❌ Harap masukkan angka yang valid atau 0 untuk membatalkan.${RESET}"
      continue
    fi

    sel_pass=$(sed -n "${id}p" "$USER_DB" | cut -d'|' -f1 | xargs)

    if [[ -z "$sel_pass" ]]; then
      echo -e "${RED}❌ ID tidak valid. Coba lagi atau tekan 0 untuk membatalkan.${RESET}"
      continue
    fi

    break
  done

  cp "$CONFIG_FILE" "$BACKUP_FILE"

  if jq --arg pw "$sel_pass" '.auth.config -= [$pw]' "$CONFIG_FILE" > temp && mv temp "$CONFIG_FILE"; then
    sed -i "/^$sel_pass[[:space:]]*|/d" "$USER_DB"
    echo -e "${GREEN}🗑️ Pengguna berhasil dihapus.${RESET}"
    systemctl restart zivpn.service
  else
    echo -e "${RED}❌ Gagal menghapus pengguna. Tidak ada perubahan yang dilakukan.${RESET}"
  fi

  read -p "🔙 Tekan Enter untuk kembali ke menu..."
}

renew_user() {
  list_users

  while true; do
    read -p "🔢 ID pengguna yang akan diperpanjang (0 untuk membatalkan): " id
    id=$(echo "$id" | xargs)  # Hapus spasi

    if [[ "$id" == "0" ]]; then
      echo -e "${YELLOW}⚠️ Perpanjangan dibatalkan.${RESET}"
      read -p "🔙 Tekan Enter untuk kembali ke menu..."
      return
    fi

    if [[ ! "$id" =~ ^[0-9]+$ ]]; then
      echo -e "${RED}❌ Harap masukkan angka yang valid.${RESET}"
      continue
    fi

    sel_pass=$(sed -n "${id}p" "$USER_DB" | cut -d'|' -f1 | xargs)

    if [[ -z "$sel_pass" ]]; then
      echo -e "${RED}❌ ID tidak valid atau tidak ada. Coba lagi atau tekan 0 untuk membatalkan.${RESET}"
      continue
    fi

    break
  done

  while true; do
    read -p "📅 Hari tambahan: " days
    if [[ ! "$days" =~ ^[0-9]+$ ]] || [[ "$days" -le 0 ]]; then
      echo -e "${RED}❌ Masukkan angka positif yang valid.${RESET}"
    else
      break
    fi
  done

  old_exp=$(sed -n "/^$sel_pass[[:space:]]*|/p" "$USER_DB" | cut -d'|' -f2 | xargs)

  if [[ -z "$old_exp" ]]; then
    echo -e "${RED}❌ Tanggal kedaluwarsa tidak ditemukan untuk pengguna ini.${RESET}"
    read -p "🔙 Tekan Enter untuk kembali ke menu..."
    return
  fi

  new_exp=$(date -d "$old_exp +$days days" +%Y-%m-%d)

  sed -i "s/^$sel_pass[[:space:]]*|.*/$sel_pass | $new_exp/" "$USER_DB"

  echo -e "${GREEN}🔁 Pengguna diperpanjang hingga: $new_exp${RESET}"

  systemctl restart zivpn.service

  read -p "🔙 Tekan Enter untuk kembali ke menu..."
}

list_users() {
  echo -e "\n${CYAN}📋 DAFTAR PENGGUNA TERDAFTAR${RESET}"
  echo -e "${CYAN}╔════╦══════════════════════╦══════════════════╦══════════════════╗${RESET}"
  echo -e "${CYAN}║ ID ║     PASSWORD         ║     EXPIRED      ║     STATUS       ║${RESET}"
  echo -e "${CYAN}╠════╬══════════════════════╬══════════════════╬══════════════════╣${RESET}"

  i=1
  today=$(date +%Y-%m-%d)
  while IFS='|' read -r pass exp; do
    pass=$(echo "$pass" | xargs)
    exp=$(echo "$exp" | xargs)

    if [[ "$exp" < "$today" ]]; then
      status="🔴 KEDALUWARSA"
    else
      status="🟢 AKTIF"
    fi

    printf "${CYAN}║ %2s ║ ${YELLOW}%-20s${CYAN} ║ ${YELLOW}%-16s${CYAN} ║ ${YELLOW}%-14s${CYAN}     ║${RESET}\n" "$i" "$pass" "$exp" "$status"
    ((i++))
  done < "$USER_DB"

  echo -e "${CYAN}╚════╩══════════════════════╩══════════════════╩══════════════════╝${RESET}\n"
  # Hanya tampilkan jeda jika dipanggil dengan argumen true
  [[ "$1" == "true" ]] && read -p "🔙 Tekan Enter untuk kembali ke menu..."
}

clean_expired_users() {
  local today=$(date +%Y-%m-%d)
  local updated=0
  local expired=()

  cp "$CONFIG_FILE" "$BACKUP_FILE"

  while IFS='|' read -r pass exp; do
    pass=$(echo "$pass" | xargs)
    exp=$(echo "$exp" | xargs)
    if [[ "$exp" < "$today" ]]; then
      expired+=("$pass")
    fi
  done < "$USER_DB"

  if [[ ${#expired[@]} -eq 0 ]]; then
    echo -e "${GREEN}✅ Tidak ada pengguna kedaluwarsa untuk dihapus.${RESET}"
    return
  fi

  # Perbarui config.json dengan menghapus semua pengguna kedaluwarsa sekaligus
  local jq_filter='.'
  for pw in "${expired[@]}"; do
    jq_filter+=" | del(.auth.config[] | select(. == \"$pw\"))"
  done

  if ! jq "$jq_filter" "$CONFIG_FILE" > temp && mv temp "$CONFIG_FILE"; then
    echo -e "${RED}❌ Gagal memperbarui $CONFIG_FILE dengan jq.${RESET}"
    return 1
  fi

  # Hapus pengguna kedaluwarsa dari USER_DB dengan aman
  local temp_db=$(mktemp)
  grep -v -F -f <(printf '%s\n' "${expired[@]}") "$USER_DB" > "$temp_db" && mv "$temp_db" "$USER_DB"

  for u in "${expired[@]}"; do
    echo -e "${YELLOW}🧹 Pengguna kedaluwarsa dihapus: $u${RESET}"
  done

  systemctl restart zivpn.service
  echo -e "${GREEN}✅ Pembersihan selesai dan layanan direstart.${RESET}"
}

toggle_autoclean() {
  if [[ "$AUTOCLEAN" == "ON" ]]; then
    echo "AUTOCLEAN=OFF" > "$CONF_FILE"
    AUTOCLEAN=OFF
  else
    echo "AUTOCLEAN=ON" > "$CONF_FILE"
    AUTOCLEAN=ON
  fi
}

# ▶️ Layanan
start_service() {
  if systemctl start zivpn.service; then
    echo -e "${GREEN}▶️ Layanan dimulai.${RESET}"
  else
    echo -e "${RED}❌ Gagal memulai layanan.${RESET}"
  fi
  read -rp "🔙 Tekan Enter untuk kembali ke menu..."
}

stop_service() {
  if systemctl stop zivpn.service; then
    echo -e "${RED}⏹️ Layanan dihentikan.${RESET}"
  else
    echo -e "${RED}❌ Gagal menghentikan layanan.${RESET}"
  fi
  read -rp "🔙 Tekan Enter untuk kembali ke menu..."
}

restart_service() {
  if systemctl restart zivpn.service; then
    echo -e "${YELLOW}🔁 Layanan direstart.${RESET}"
  else
    echo -e "${RED}❌ Gagal merestart layanan.${RESET}"
  fi
  read -rp "🔙 Tekan Enter untuk kembali ke menu..."
}

# 📺 Menu utama
while true; do
  clear  # ✅ Bersihkan layar di setiap iterasi menu

[[ "$AUTOCLEAN" == "ON" ]] && clean_expired_users > /dev/null

# Dapatkan data nyata
IP_PRIVADA=$(hostname -I | awk '{print $1}')
IP_PUBLICA=$(curl -s ifconfig.me)
OS_MACHINE=$(grep -oP '^PRETTY_NAME="\K[^"]+' /etc/os-release)
ARCH_MACHINE=$(uname -m)

# Dapatkan Domain
if [[ -f "$DOMAIN_FILE" ]]; then
  DOMAIN=$(cat "$DOMAIN_FILE")
else
  DOMAIN="Tidak diatur"
fi

# Normalisasi arsitektur untuk menampilkan AMD atau ARM
if [[ "$ARCH_MACHINE" =~ "arm" || "$ARCH_MACHINE" =~ "aarch" ]]; then
  ARCH_DISPLAY="ARM"
else
  ARCH_DISPLAY="AMD"
fi
PORT="5667"
PORT_RANGE="6000-19999"

echo -e "\n${CYAN}╔═════════════════════════════════════════════════════════════════╗"
echo -e "║                🧩 ZIVPN - PANEL PENGGUNA UDP                    ║"
echo -e "╠═════════════════════════════════════════════════════════════════╣"
echo -e "║                         📊 INFORMASI                            ║"
echo -e "╠═════════════════════════════════════════════════════════════════╣"
echo -e "${CYAN}║ 🌐 Domain:       ${GREEN}${DOMAIN}${CYAN}                                 ║"
echo -e "${CYAN}║ 📶 IP Privat:    ${GREEN}${IP_PRIVADA}${CYAN}                                       ║"
echo -e "${CYAN}║ 🌐 IP Publik:    ${GREEN}${IP_PUBLICA}${CYAN}                                 ║"
echo -e "${CYAN}║ 🖥️ OS:           ${GREEN}${OS_MACHINE}${CYAN}                             ║"
echo -e "${CYAN}║ 🧠 Arsitektur:   ${GREEN}${ARCH_DISPLAY}${CYAN}                                            ║"
echo -e "${CYAN}║ 📍 Port:         ${GREEN}${PORT}${CYAN}                                           ║"
echo -e "${CYAN}║ 🔥 IPTABLES:     ${GREEN}${PORT_RANGE}${CYAN}                                     ║"
echo -e "╠═════════════════════════════════════════════════════════════════╣"
echo -e "║ [1] ➕  Buat pengguna baru (dengan kedaluwarsa)                 ║"
echo -e "║ [2] ❌  Hapus pengguna                                          ║"
echo -e "║ [3] 🗓  Perpanjang pengguna                                     ║"
echo -e "║ [4] 📋  Informasi pengguna                                      ║"
echo -e "║ [5] ▶️  Mulai layanan                                           ║"
echo -e "║ [6] 🔁  Restart layanan                                         ║"
echo -e "║ [7] ⏹️  Hentikan layanan                                        ║"
if [[ "$AUTOCLEAN" == "ON" ]]; then
  echo -e "║ [8] 🧹  Hapus pengguna kedaluwarsa            [${GREEN}ON${CYAN}]              ║"
else
  echo -e "║ [8] 🧹  Hapus pengguna kedaluwarsa            [${RED}OFF${CYAN}]             ║"
fi
echo -e "║ [9] 🚪  Keluar                                                  ║"
echo -e "╚═════════════════════════════════════════════════════════════════╝${RESET}"

read -p "📌 Pilih opsi: " opc
case $opc in
  1) add_user;;
  2) remove_user;;
  3) renew_user;;
  4) list_users true;;
  5) start_service;;
  6) restart_service;;
  7) stop_service;;
  8) toggle_autoclean;;
  9) exit;;
  *) echo -e "${RED}❌ Opsi tidak valid.${RESET}";;
esac
done
