#!/bin/sh
# stop.sh
# Kills the browser dashboard and restores Kindle GUI

DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
PID_FILE="$DASHBOARD_DIR/browser.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    echo "Killing browser process $PID..."
    kill -9 "$PID" 2>/dev/null || true
    rm -f "$PID_FILE"
else
    killall -9 kindle_browser 2>/dev/null || true
fi

echo "Restarting Kindle GUI..."
start lab126_gui 2>/dev/null || /etc/init.d/framework start 2>/dev/null || true

eips -c 2>/dev/null || true
echo "Dashboard stopped."
