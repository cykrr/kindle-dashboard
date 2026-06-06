#!/bin/sh
set -e
export DISPLAY=:0
export XDG_CONFIG_HOME="/mnt/us/system/browser/"
CHROME_LIBS="/usr/bin/chromium/lib:/usr/bin/chromium/usr/lib:/usr/lib/"

DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
PROFILE_DIR="/mnt/us/system/browser/kindle-dashboard-profile"
SETTINGS_FILE="$DASHBOARD_DIR/device-settings.env"
LAUNCH_ORIENTATION="${ORIENTATION:-}"
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

ORIENTATION="${LAUNCH_ORIENTATION:-${SAVED_ORIENTATION:-270}}"
case "$ORIENTATION" in
  0|90|180|270) ;;
  *) ORIENTATION="270" ;;
esac

{
  [ -n "$SAVED_BRIGHTNESS" ] && printf 'BRIGHTNESS=%s\n' "$SAVED_BRIGHTNESS"
  printf 'ORIENTATION=%s\n' "$ORIENTATION"
} >"$SETTINGS_FILE"

TARGET="file://$DASHBOARD_DIR/index.html?orientation=$ORIENTATION"

# Stop GUI for fullscreen
trap "" TERM
stop lab126_gui 2>/dev/null || /etc/init.d/framework stop 2>/dev/null || true
usleep 1250000
trap - TERM

# Clear e-ink display artifacts
eips -c 2>/dev/null || true
eips -c 2>/dev/null || true

killall -9 kindle_browser 2>/dev/null || true
if [ -f "$DASHBOARD_DIR/settings-server.pid" ]; then
  kill "$(cat "$DASHBOARD_DIR/settings-server.pid")" 2>/dev/null || true
  rm -f "$DASHBOARD_DIR/settings-server.pid"
fi
rm -rf "$PROFILE_DIR"
mkdir -p "$PROFILE_DIR"

# Start local device settings API
"$DASHBOARD_DIR/settings-server.sh" >"$DASHBOARD_DIR/settings-server.out" 2>&1 &

# Launch browser in fullscreen mode
LD_LIBRARY_PATH="$CHROME_LIBS" /usr/bin/chromium/bin/kindle_browser \
  "$TARGET" \
  --kiosk \
  --no-first-run \
  --no-zygote \
  --no-sandbox \
  --single-process \
  --user-data-dir="$PROFILE_DIR" \
  --disable-gpu \
  --content-shell-hide-toolbar \
  >"$DASHBOARD_DIR/browser.out" 2>&1 &

echo $! >"$DASHBOARD_DIR/browser.pid"

# Keep the device awake
lipc-set-prop -i com.lab126.powerd wakeUp 1 2>/dev/null || true
lipc-set-prop -i com.lab126.powerd preventScreenSaver 1 2>/dev/null || true
