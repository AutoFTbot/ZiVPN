#!/bin/bash

# Colors
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
CYAN="\033[1;36m"
RED="\033[1;31m"
RESET="\033[0m"
BOLD="\033[1m"
GRAY="\033[1;30m"

print_task() {
  echo -ne "${GRAY}•${RESET} $1..."
}

print_done() {
  echo -e "\r${GREEN}✓${RESET} $1      "
}

run_silent() {
  local msg="$1"
  local cmd="$2"
  
  print_task "$msg"
  bash -c "$cmd" &>/tmp/zivpn_uninstall.log
  if [ $? -eq 0 ]; then
    print_done "$msg"
  else
    print_done "$msg" 
  fi
}

clear
echo -e "${BOLD}ZiVPN UDP Uninstaller${RESET}"
echo -e "${GRAY}AutoFTbot Edition${RESET}"
echo ""

run_silent "Stopping services" "systemctl stop zivpn.service zivpn-api.service zivpn-bot.service zivpn_backfill.service &>/dev/null; systemctl disable zivpn.service zivpn-api.service zivpn-bot.service zivpn_backfill.service &>/dev/null; killall zivpn zivpn-api zivpn-bot &>/dev/null"

run_silent "Removing files" "rm -rf /etc/zivpn /usr/local/bin/zivpn /etc/systemd/system/zivpn.service /etc/systemd/system/zivpn-api.service /etc/systemd/system/zivpn-bot.service /etc/systemd/system/zivpn_backfill.service /etc/zivpn-iptables-fix-applied /usr/local/bin/menu-zivpn /etc/zivpn/bot-config.json /etc/zivpn/apikey"

iface=$(ip -4 route ls | grep default | grep -Po '(?<=dev )(\S+)' | head -1)
run_silent "Cleaning network rules" "iptables -t nat -D PREROUTING -i $iface -p udp --dport 6000:19999 -j DNAT --to-destination :5667 &>/dev/null"

run_silent "Reloading systemd" "systemctl daemon-reload && systemctl daemon-reexec"
run_silent "Cleaning cache" "echo 3 > /proc/sys/vm/drop_caches && sysctl -w vm.drop_caches=3 &>/dev/null && swapoff -a && swapon -a"

echo ""
echo -e "${BOLD}Uninstallation Complete${RESET}"
echo -e "${GRAY}ZiVPN has been completely removed from your system.${RESET}"
echo ""
