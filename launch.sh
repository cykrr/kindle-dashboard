#!/bin/sh
set -e
export DISPLAY=:0
export XDG_CONFIG_HOME="/mnt/us/system/browser/"
CHROME_LIBS="/usr/bin/chromium/lib:/usr/bin/chromium/usr/lib:/usr/lib/"

DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
PROFILE_DIR="/mnt/us/system/browser/kindle-dashboard-profile"
TARGET="file://$DASHBOARD_DIR/index.html"

# Stop GUI for fullscreen
trap "" TERM
stop lab126_gui 2>/dev/null || /etc/init.d/framework stop 2>/dev/null || true
usleep 1250000
trap - TERM

# Clear e-ink display artifacts
eips -c 2>/dev/null || true
eips -c 2>/dev/null || true

killall -9 kindle_browser 2>/dev/null || true
rm -rf "$PROFILE_DIR"
mkdir -p "$PROFILE_DIR"

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
