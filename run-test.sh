#!/bin/sh
export DISPLAY=:0
/mnt/us/documents/kindle-dashboard/dashboard-native > /tmp/dash.log 2>&1 &
sleep 5
ps | grep dashboard-native | grep -v grep
