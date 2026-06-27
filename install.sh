#!/usr/bin/env bash
#
# install.sh — build spark, install the binary to ~/.local/bin and register a
# system-wide systemd service that starts at boot. Linux + systemd only.
#
# Run as your normal user (it calls sudo where needed), or via sudo.
#
set -euo pipefail

APP="spark"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_FILE="/etc/systemd/system/$APP.service"

# --- preflight ---------------------------------------------------------------
if [[ "$(uname -s)" != "Linux" ]]; then
  echo "error: this installer only supports Linux (systemd)." >&2
  exit 1
fi
if ! command -v systemctl >/dev/null 2>&1; then
  echo "error: systemctl not found; this installer requires systemd." >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "error: go toolchain not found on PATH." >&2
  exit 1
fi

# Run privileged steps with sudo unless we are already root.
if [[ "$EUID" -eq 0 ]]; then
  SUDO=""
  TARGET_USER="${SUDO_USER:-root}"
else
  SUDO="sudo"
  TARGET_USER="$USER"
fi
TARGET_GROUP="$(id -gn "$TARGET_USER")"
TARGET_HOME="$(getent passwd "$TARGET_USER" | cut -d: -f6)"
if [[ -z "$TARGET_HOME" ]]; then
  echo "error: cannot resolve home directory for user $TARGET_USER." >&2
  exit 1
fi

BIN_DIR="$TARGET_HOME/.local/bin"
BIN_PATH="$BIN_DIR/$APP"
SPARK_DIR="$TARGET_HOME/.spark"
LOG_FILE="$SPARK_DIR/$APP.log"

# --- build (as the target user) ----------------------------------------------
echo ">> building $APP for user $TARGET_USER"
mkdir -p "$BIN_DIR" "$SPARK_DIR"
( cd "$SCRIPT_DIR" && go build -o "$BIN_PATH" . )
echo ">> installed binary: $BIN_PATH"

if [[ ! -f "$SPARK_DIR/config.json" ]]; then
  echo ">> note: $SPARK_DIR/config.json not found — copy config.example.json and edit it."
fi

# --- service -----------------------------------------------------------------
echo ">> writing service: $SERVICE_FILE"
TMP_SVC="$(mktemp)"
cat > "$TMP_SVC" <<EOF
[Unit]
Description=spark GitHub webhook runner
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$TARGET_USER
Group=$TARGET_GROUP
ExecStart=$BIN_PATH
WorkingDirectory=$SPARK_DIR
Restart=on-failure
RestartSec=3
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE

[Install]
WantedBy=multi-user.target
EOF
$SUDO cp "$TMP_SVC" "$SERVICE_FILE"
$SUDO chmod 644 "$SERVICE_FILE"
rm -f "$TMP_SVC"

# --- enable & start ----------------------------------------------------------
echo ">> reloading systemd"
$SUDO systemctl daemon-reload
$SUDO systemctl enable --now "$APP.service"

echo ">> done. service starts automatically at boot."
echo "   log file: $LOG_FILE"
echo "   status:   sudo systemctl status $APP"
echo "   logs:     tail -f $LOG_FILE   (or: sudo journalctl -u $APP -f)"
