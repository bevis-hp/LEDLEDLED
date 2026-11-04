#!/usr/bin/env bash
# Configure the Pi's two Wi-Fi radios as access points with NetworkManager
# (Raspberry Pi OS Bookworm or later).
#
#   Onboard radio (brcmfmac)  -> "UI" AP that phones/laptops join
#                                10.42.0.1/24, DHCP for clients
#   USB Wi-Fi stick           -> "pixel" AP that the AirPixels join
#                                10.43.0.1/24, AirPixels use static IPs
#
# Connections are pinned to each radio's MAC address because wlan0/wlan1
# names can swap between boots when a USB stick is present.
#
# Override any setting with environment variables, e.g.
#   sudo UI_PASS=supersecret PIX_PASS=pixelpass deploy/setup-network.sh
#
# Run with --remove to delete the two connections again.
set -euo pipefail

UI_SSID="${UI_SSID:-LEDLEDLED}"
UI_PASS="${UI_PASS:-ledledled}"
UI_ADDR="${UI_ADDR:-10.42.0.1/24}"
# The Pi 4's onboard radio can do 5 GHz, which keeps phone traffic off the
# pixel network's channel. Use UI_BAND=bg UI_CHANNEL=1 if a device can't see it.
UI_BAND="${UI_BAND:-a}"
UI_CHANNEL="${UI_CHANNEL:-36}"

PIX_SSID="${PIX_SSID:-LEDLEDLED-PIX}"
PIX_PASS="${PIX_PASS:-ledpixels}"
PIX_ADDR="${PIX_ADDR:-10.43.0.1/24}"
# AirPixel Mini II is 2.4 GHz only.
PIX_CHANNEL="${PIX_CHANNEL:-6}"

# Regulatory country, required for AP mode (and for 5 GHz).
COUNTRY="${COUNTRY:-GB}"

# Captive portal: answer every DNS name on the networks with the Pi, so
# phones pop up the controller when they join. CAPTIVE=0 only maps led.lan.
CAPTIVE="${CAPTIVE:-1}"

if [[ $EUID -ne 0 ]]; then
  exec sudo --preserve-env=UI_SSID,UI_PASS,UI_ADDR,UI_BAND,UI_CHANNEL,PIX_SSID,PIX_PASS,PIX_ADDR,PIX_CHANNEL,COUNTRY,CAPTIVE "$0" "$@"
fi

if [[ "${1:-}" == "--remove" ]]; then
  nmcli connection delete ledledled-ui 2>/dev/null || true
  nmcli connection delete ledledled-pix 2>/dev/null || true
  rm -f /etc/NetworkManager/dnsmasq-shared.d/ledledled.conf
  rm -f /etc/systemd/system/ledledled.service.d/portal.conf
  rm -f /etc/modprobe.d/ledledled-rtw88.conf
  systemctl daemon-reload
  echo "Removed LEDLEDLED access points."
  exit 0
fi

command -v nmcli >/dev/null || { echo "NetworkManager (nmcli) not found. Needs Pi OS Bookworm or later." >&2; exit 1; }
if (( ${#UI_PASS} < 8 || ${#PIX_PASS} < 8 )); then
  echo "Wi-Fi passwords must be at least 8 characters." >&2
  exit 1
fi

# Find the onboard (brcmfmac) and USB radios.
onboard="" usb="" usb_driver=""
for dev in /sys/class/net/wlan*; do
  [[ -e "$dev" ]] || continue
  name=$(basename "$dev")
  driver=$(basename "$(readlink -f "$dev/device/driver")")
  if [[ "$driver" == "brcmfmac" ]]; then
    onboard=$name
  else
    usb=$name usb_driver=$driver
  fi
done
[[ -n "$onboard" ]] || { echo "Onboard Wi-Fi (brcmfmac) not found." >&2; exit 1; }
[[ -n "$usb" ]] || { echo "USB Wi-Fi stick not found. Plug it in and retry." >&2; exit 1; }

onboard_mac=$(cat "/sys/class/net/$onboard/address")
usb_mac=$(cat "/sys/class/net/$usb/address")
echo "Onboard radio: $onboard ($onboard_mac) -> UI AP '$UI_SSID' on $UI_ADDR"
echo "USB radio:     $usb ($usb_mac) -> pixel AP '$PIX_SSID' on $PIX_ADDR"

# Check the USB stick supports AP mode.
phy=$(basename "$(readlink -f "/sys/class/net/$usb/phy80211")")
if command -v iw >/dev/null && ! iw phy "$phy" info | sed -n '/Supported interface modes/,/:$/p' | grep -q '\* AP$'; then
  echo "WARNING: $usb does not report AP mode support; the pixel AP may fail." >&2
fi

# Realtek rtw88 sticks (e.g. TP-Link Archer T3U / T3U Plus, RTL8822BU) get
# switched into USB 3 mode by the driver, which radiates noise in the very
# 2.4 GHz band the pixel AP uses. USB 2 is far more bandwidth than DMX needs.
if [[ "$usb_driver" == rtw88_* ]]; then
  echo "options rtw88_usb switch_usb_mode=N" > /etc/modprobe.d/ledledled-rtw88.conf
  echo "$usb_driver: disabled USB 3 mode (takes effect after replugging the stick or rebooting)."
fi

iw reg set "$COUNTRY" 2>/dev/null || true
if command -v raspi-config >/dev/null; then
  raspi-config nonint do_wifi_country "$COUNTRY" || true
fi
rfkill unblock wifi || true

make_ap() { # name mac ssid pass addr band channel powersave
  nmcli connection delete "$1" 2>/dev/null || true
  nmcli connection add type wifi con-name "$1" ifname '*' autoconnect yes \
    802-11-wireless.mac-address "$2" \
    ssid "$3" mode ap \
    wifi.band "$6" wifi.channel "$7" wifi.powersave "$8" \
    wifi-sec.key-mgmt wpa-psk wifi-sec.psk "$4" \
    wifi-sec.proto rsn wifi-sec.pairwise ccmp wifi-sec.group ccmp \
    ipv4.method shared ipv4.addresses "$5" \
    ipv6.method disabled
}

# powersave 2 = disabled; keeps pixel latency low and steady.
make_ap ledledled-ui "$onboard_mac" "$UI_SSID" "$UI_PASS" "$UI_ADDR" "$UI_BAND" "$UI_CHANNEL" 2
make_ap ledledled-pix "$usb_mac" "$PIX_SSID" "$PIX_PASS" "$PIX_ADDR" bg "$PIX_CHANNEL" 2

# DNS for the AP clients (NetworkManager's dnsmasq). led.lan always points
# at the Pi; with the captive portal on, every other name does too.
UI_IP="${UI_ADDR%/*}"
mkdir -p /etc/NetworkManager/dnsmasq-shared.d /etc/systemd/system/ledledled.service.d
if [[ "$CAPTIVE" == "1" ]]; then
  echo "address=/#/${UI_IP}" > /etc/NetworkManager/dnsmasq-shared.d/ledledled.conf
  PORTAL="http://${UI_IP}/"
else
  echo "address=/led.lan/${UI_IP}" > /etc/NetworkManager/dnsmasq-shared.d/ledledled.conf
  PORTAL=""
fi
# Tell the app where the portal lives (or that it's off).
printf '[Service]\nEnvironment=LEDLEDLED_PORTAL=%s\n' "$PORTAL" > /etc/systemd/system/ledledled.service.d/portal.conf
systemctl daemon-reload
systemctl try-restart ledledled.service || true

nmcli connection up ledledled-ui
nmcli connection up ledledled-pix

cat <<EOF

Done.
  Phones/laptops: join Wi-Fi "$UI_SSID" (password: $UI_PASS)
                  $( [[ "$CAPTIVE" == "1" ]] && echo "the controller should pop up automatically; otherwise " )open http://${UI_ADDR%/*}  (or http://led.lan)
  AirPixels:      join Wi-Fi "$PIX_SSID" (password: $PIX_PASS)
                  with static IPs in ${PIX_ADDR%.*}.x, e.g. ${PIX_ADDR%.*}.11, .12, .13
                  (subnet mask 255.255.255.0)

NOTE: if you were connected to this Pi over Wi-Fi, that link is now gone;
reconnect via the "$UI_SSID" network or Ethernet.
EOF
