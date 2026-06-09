#!/bin/sh
set -eu

DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
SETTINGS_FILE="$DASHBOARD_DIR/device-settings.env"
BRIGHTNESS_FILE="/sys/devices/soc0/bl/backlight/bl/brightness"
MAX_FILE="/sys/devices/soc0/bl/backlight/bl/max_brightness"
PID_FILE="$DASHBOARD_DIR/settings-server.pid"
PORT="${SETTINGS_PORT:-8177}"

read_int_file() {
  if [ -f "$1" ]; then
    tr -cd '0-9' <"$1"
  fi
}

apply_saved_brightness() {
  [ -f "$SETTINGS_FILE" ] || return 0
  . "$SETTINGS_FILE"
  case "${BRIGHTNESS:-}" in
    ''|*[!0-9]*) return 0 ;;
  esac
  max="$(read_int_file "$MAX_FILE")"
  [ -n "$max" ] || max=2399
  value="$BRIGHTNESS"
  [ "$value" -ge 0 ] || value=0
  [ "$value" -le "$max" ] || value="$max"
  printf '%s' "$value" >"$BRIGHTNESS_FILE" 2>/dev/null || true
}

cleanup() {
  rm -f "$PID_FILE"
}

trap cleanup EXIT INT TERM

echo $$ >"$PID_FILE"
apply_saved_brightness
exec nc -lk -s 127.0.0.1 -p "$PORT" -e "$DASHBOARD_DIR/settings-api.sh"
