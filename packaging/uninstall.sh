#!/bin/sh

set -eu

BINARY_DST="/usr/local/sbin/config-api"
SERVICE_DST="/etc/systemd/system/config-api.service"

if [ "$(id -u)" -ne 0 ]; then
  echo "please run as root: sudo ./uninstall.sh" >&2
  exit 1
fi

if systemctl list-unit-files | grep -q '^config-api\.service'; then
  systemctl disable --now config-api || true
fi

rm -f "$SERVICE_DST"
rm -f "$BINARY_DST"
systemctl daemon-reload

echo "config-api removed."
