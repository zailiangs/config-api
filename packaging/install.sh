#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
BINARY_SRC="$SCRIPT_DIR/config-api"
SERVICE_SRC="$SCRIPT_DIR/config-api.service"
BINARY_DST="/usr/local/sbin/config-api"
SERVICE_DST="/etc/systemd/system/config-api.service"

if [ "$(id -u)" -ne 0 ]; then
  echo "please run as root: sudo ./install.sh" >&2
  exit 1
fi

if [ ! -f "$BINARY_SRC" ]; then
  echo "missing binary: $BINARY_SRC" >&2
  exit 1
fi

if [ ! -f "$SERVICE_SRC" ]; then
  echo "missing systemd unit: $SERVICE_SRC" >&2
  exit 1
fi

install -m 0755 "$BINARY_SRC" "$BINARY_DST"
install -m 0644 "$SERVICE_SRC" "$SERVICE_DST"
systemctl daemon-reload
systemctl enable --now config-api

echo "config-api installed."
echo "health check: curl http://127.0.0.1:18881/healthz"
echo "status: systemctl status config-api"
