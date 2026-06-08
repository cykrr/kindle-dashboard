#!/bin/sh
export DISPLAY=:0
echo "Starting..."
/mnt/us/documents/kindle-dashboard/dashboard-native 2>/tmp/dash.err
echo "Exit: $?"
cat /tmp/dash.err
