#!/bin/sh
set -eu

DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
SETTINGS_FILE="$DASHBOARD_DIR/device-settings.env"
BRIGHTNESS_FILE="/sys/devices/soc0/bl/backlight/bl/brightness"
MAX_FILE="/sys/devices/soc0/bl/backlight/bl/max_brightness"

trim_cr() {
  printf '%s' "$1" | tr -d '\r'
}

read_int_file() {
  if [ -f "$1" ]; then
    tr -cd '0-9' <"$1"
  fi
}

load_settings() {
  SAVED_BRIGHTNESS=""
  SAVED_ORIENTATION=""
  if [ -f "$SETTINGS_FILE" ]; then
    # shellcheck disable=SC1090
    . "$SETTINGS_FILE"
    case "${BRIGHTNESS:-}" in
      ''|*[!0-9]*) ;;
      *) SAVED_BRIGHTNESS="$BRIGHTNESS" ;;
    esac
    case "${ORIENTATION:-}" in
      0|90|180|270) SAVED_ORIENTATION="$ORIENTATION" ;;
    esac
  fi
}

write_settings() {
  brightness="$1"
  orientation="$2"
  tmp="$SETTINGS_FILE.tmp"
  {
    [ -n "$brightness" ] && printf 'BRIGHTNESS=%s\n' "$brightness"
    [ -n "$orientation" ] && printf 'ORIENTATION=%s\n' "$orientation"
  } >"$tmp"
  mv "$tmp" "$SETTINGS_FILE"
}

current_brightness() {
  read_int_file "$BRIGHTNESS_FILE"
}

max_brightness() {
  value="$(read_int_file "$MAX_FILE")"
  [ -n "$value" ] || value=2399
  printf '%s' "$value"
}

current_orientation() {
  load_settings
  printf '%s' "${SAVED_ORIENTATION:-270}"
}

battery_level() {
  read_int_file "/sys/class/power_supply/bd71827_bat/capacity"
}

battery_status() {
  if [ -f "/sys/class/power_supply/bd71827_bat/status" ]; then
    tr -d '\r\n' <"/sys/class/power_supply/bd71827_bat/status"
  fi
}

clamp() {
  value="$1"
  max="$2"
  [ "$value" -ge 0 ] || value=0
  [ "$value" -le "$max" ] || value="$max"
  printf '%s' "$value"
}

reply() {
  status="$1"
  body="$2"
  length=$(printf '%s' "$body" | wc -c | tr -d ' ')
  printf 'HTTP/1.1 %s\r\n' "$status"
  printf 'Access-Control-Allow-Origin: *\r\n'
  printf 'Access-Control-Allow-Methods: GET, POST, OPTIONS\r\n'
  printf 'Access-Control-Allow-Headers: Content-Type\r\n'
  printf 'Content-Type: text/plain; charset=utf-8\r\n'
  printf 'Content-Length: %s\r\n' "$length"
  printf 'Connection: close\r\n'
  printf '\r\n'
  printf '%s' "$body"
}

read -r request_line || exit 0
request_line="$(trim_cr "$request_line")"
method=$(printf '%s' "$request_line" | awk '{print $1}')
target=$(printf '%s' "$request_line" | awk '{print $2}')

while IFS= read -r header_line; do
  header_line="$(trim_cr "$header_line")"
  [ -n "$header_line" ] || break
done

path="${target%%\?*}"
query=""
[ "$target" = "$path" ] || query="${target#*\?}"

case "$method:$path" in
  OPTIONS:*)
    reply '204 No Content' ''
    ;;
  GET:/brightness)
    reply '200 OK' "$(current_brightness)"
    ;;
  GET:/brightness-max)
    reply '200 OK' "$(max_brightness)"
    ;;
  GET:/orientation)
    reply '200 OK' "$(current_orientation)"
    ;;
  GET:/battery-level)
    reply '200 OK' "$(battery_level)"
    ;;
  GET:/battery-status)
    reply '200 OK' "$(battery_status)"
    ;;
  POST:/brightness)
    value=$(printf '%s' "$query" | sed -n 's/.*value=\([0-9][0-9]*\).*/\1/p' | head -n 1)
    if [ -z "$value" ]; then
      reply '400 Bad Request' 'missing value'
      exit 0
    fi
    max="$(max_brightness)"
    value="$(clamp "$value" "$max")"
    if printf '%s' "$value" >"$BRIGHTNESS_FILE" 2>/dev/null; then
      load_settings
      write_settings "$value" "$SAVED_ORIENTATION"
      reply '200 OK' "$(current_brightness)"
    else
      reply '500 Internal Server Error' 'failed to set brightness'
    fi
    ;;
  POST:/orientation)
    value=$(printf '%s' "$query" | sed -n 's/.*value=\([0-9][0-9]*\).*/\1/p' | head -n 1)
    case "$value" in
      0|90|180|270) ;;
      *)
        reply '400 Bad Request' 'invalid orientation'
        exit 0
        ;;
    esac
    load_settings
    write_settings "$SAVED_BRIGHTNESS" "$value"
    reply '200 OK' "$value"
    ;;
  *)
    reply '404 Not Found' 'not found'
    ;;
esac
