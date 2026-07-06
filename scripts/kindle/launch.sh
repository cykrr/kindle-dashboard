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

# Keep device awake — preventScreenSaver stops the screensaver but powerd may
# still suspend on the physical power button press. The dashboard's own
# powerbutton.go intercepts the gpio-keys input event (KEY_POWER) and jumps
# to the rest screen (ViewHome) before powerd can act.
lipc-set-prop -i com.lab126.powerd wakeUp 1 2>/dev/null || true
lipc-set-prop -i com.lab126.powerd preventScreenSaver 1 2>/dev/null || true

# Stop the touchscreen (cyttsp5) from being a suspend wakeup source — it was
# firing spontaneous resumes mid-quiet-hour-sleep unrelated to the power
# button or RTC alarm. Only gpio-keys (power button) and the RTC should wake
# the device.
for d in /sys/bus/i2c/devices/*/; do
  if [ -f "${d}name" ] && grep -qi cyttsp "${d}name" 2>/dev/null; then
    echo disabled >"${d}power/wakeup" 2>/dev/null || true
  fi
done

wait $DPID
