#!/usr/bin/env bash
# Install or update LEDLEDLED on a Raspberry Pi. Run on the Pi:
#
#   sudo deploy/install.sh [path/to/ledledled]
#
# `make install` builds the binary and runs this for you. The binary defaults
# to ../bin/ledledled (the repo's build output) next to this script.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  exec sudo "$0" "$@"
fi
here="$(cd "$(dirname "$0")" && pwd)"
bin="${1:-$here/../bin/ledledled}"
if [[ ! -x "$bin" ]]; then
  echo "No binary at $bin - run 'make install' from the repo instead." >&2
  exit 1
fi

install -d /opt/ledledled
install -m 755 "$bin" /opt/ledledled/ledledled
install -m 644 "$here/ledledled.service" /etc/systemd/system/ledledled.service

systemctl daemon-reload
systemctl enable ledledled.service
systemctl restart ledledled.service

echo "LEDLEDLED installed. Status:"
systemctl --no-pager --lines=5 status ledledled.service || true
