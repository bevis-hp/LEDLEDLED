#!/usr/bin/env bash
# Install or update LEDLEDLED on a Raspberry Pi.
#
# Run on the Pi from a directory containing:
#   ledledled            (the linux/arm64 binary from `make pi`)
#   ledledled.service
# `make install-pi` copies these over and runs this script for you.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  exec sudo "$0" "$@"
fi
cd "$(dirname "$0")"

install -d /opt/ledledled
install -m 755 ledledled /opt/ledledled/ledledled
install -m 644 ledledled.service /etc/systemd/system/ledledled.service

systemctl daemon-reload
systemctl enable ledledled.service
systemctl restart ledledled.service

echo "LEDLEDLED installed. Status:"
systemctl --no-pager --lines=5 status ledledled.service || true
