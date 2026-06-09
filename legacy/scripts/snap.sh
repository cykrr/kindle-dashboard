#!/bin/sh
export DISPLAY=:0
WID=$(xwininfo -root -tree 2>/dev/null | grep "Kindle Dashboard" | head -1 | awk '{print $1}')
echo "WID=$WID"
if [ -n "$WID" ]; then
    xwd -id "$WID" -out /tmp/dash-win.xwd 2>&1
    ls -la /tmp/dash-win.xwd
else
    echo "Window not found"
fi
