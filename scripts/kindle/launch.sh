#!/bin/sh
set -e
export DISPLAY=:0
DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
LOG_FILE="/tmp/dashboard-native.log"

# echo "=== Stopping Kindle GUI ==="
# trap "" TERM
# stop lab126_gui 2>/dev/null || /etc/init.d/framework stop 2>/dev/null || true
# usleep 1250000
# trap - TERM

eips -c 2>/dev/null || true

# Swap USB mass-storage gadget for USB ethernet, so plugging in a cable
# gives a usb0 network interface instead of triggering USB drive mode
# (which kills wlan0).
rmmod g_mass_storage 2>/dev/null || true
modprobe g_ether 2>/dev/null || true

killall -9 dashboard-native 2>/dev/null || true
sleep 2

echo "=== Launching native dashboard ==="
echo "Logging to $LOG_FILE"
{
  echo "=== $(date '+%Y-%m-%dT%H:%M:%S%z') launching dashboard-native ==="
  "$DASHBOARD_DIR/dashboard-native" -hw-landscape -suspend-cycle
  echo "=== $(date '+%Y-%m-%dT%H:%M:%S%z') dashboard-native exited: $? ==="
} >>"$LOG_FILE" 2>&1 &
DPID=$!
echo "PID: $DPID"

# Keep device awake
lipc-set-prop -i com.lab126.powerd wakeUp 1 2>/dev/null || true
lipc-set-prop -i com.lab126.powerd preventScreenSaver 1 2>/dev/null || true

wait $DPID
