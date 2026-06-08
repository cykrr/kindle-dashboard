#!/bin/sh
export DISPLAY=:0
/mnt/us/documents/kindle-dashboard/dashboard-native > /dev/null 2>&1 &
echo "Started: $!"
