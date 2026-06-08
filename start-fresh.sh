#!/bin/sh
export DISPLAY=:0
/mnt/us/documents/kindle-dashboard/dashboard-native 2>/tmp/dash.err &
echo "PID: $!"
sleep 6
ps | grep dashboard-native | grep -v grep | head -1
echo "---"
if [ -s /tmp/dash.err ]; then
    echo "Errors:"
    cat /tmp/dash.err
else
    echo "No errors"
fi
echo "---"
xwininfo -root -tree 2>/dev/null | grep "Kindle Dashboard" | head -1
