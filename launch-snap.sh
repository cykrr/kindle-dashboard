#!/bin/sh
export DISPLAY=:0

# Kill old instances
killall -9 dashboard-native 2>/dev/null
killall -9 kindle_browser 2>/dev/null
stop lab126_gui 2>/dev/null || true
usleep 1250000

eips -c 2>/dev/null

# Start dashboard
/mnt/us/documents/kindle-dashboard/dashboard-native &
sleep 6
eips -f

# Now screenshot
WID=$(xwininfo -root -tree 2>/dev/null | grep "Kindle Dashboard" | head -1 | awk '{print $1}')
echo "WID=$WID"
if [ -n "$WID" ]; then
    xwd -id "$WID" -out /tmp/dash-win.xwd 2>&1
    xwd -root -out /tmp/dash-root.xwd 2>&1
    echo "Captured window and root"
    ls -la /tmp/dash-*.xwd
else
    echo "Cannot find dashboard window"
    xwininfo -root -tree 2>/dev/null | head -20
fi
