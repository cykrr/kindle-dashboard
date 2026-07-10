#!/bin/sh
# Kindle Dashboard KUAL helper - start/stop/status
# The actual dashboard binary lives at /mnt/us/kindle-dashboard/dashboard-native
# and the long-running lifecycle is owned by launch.sh.
#
# This script is the KUAL-facing shim that:
#   - Starts/stops the dashboard without touching USB gadget modules
#     (USB networking is managed by the SSH Manager extension)
#   - Shows status on the eInk screen via eips

DASHBOARD_DIR="/mnt/us/kindle-dashboard"
# The dashboard owns LOG_FILE in-process (size-capped, rotated). The shell only
# captures stdout/stderr — panics and any pre-logger output — into a small boot
# log that is truncated on every start so it can never grow unbounded.
LOG_FILE="/tmp/dashboard-native.log"
BOOT_LOG="/tmp/dashboard-boot.log"

screen_msg() {
    eips -c 2>/dev/null
    eips 1 1 "$1"
    eips 1 2 "$2"
    eips 1 3 "$3"
    eips 1 4 "$4"
}

is_running() {
    pidof dashboard-native >/dev/null 2>&1
}

dashboard_start() {
    if is_running; then
        screen_msg "Dashboard already running"
        return
    fi

    if [ ! -x "$DASHBOARD_DIR/dashboard-native" ]; then
        screen_msg "Dashboard failed" "binary missing at:" "$DASHBOARD_DIR/dashboard-native"
        return
    fi

    # Stop the Kindle GUI (framework) so the dashboard can claim the display.
    # Launch with DISPLAY=:0 and common flags.
    export DISPLAY=:0

    # Keep device awake
    lipc-set-prop -i com.lab126.powerd wakeUp 1 2>/dev/null || true
    lipc-set-prop -i com.lab126.powerd preventScreenSaver 1 2>/dev/null || true

    # Stop the touchscreen from being a suspend wakeup source
    for d in /sys/bus/i2c/devices/*/; do
      if [ -f "${d}name" ] && grep -qi cyttsp "${d}name" 2>/dev/null; then
        echo disabled >"${d}power/wakeup" 2>/dev/null || true
      fi
    done

    screen_msg "Starting Dashboard..."

    # Detach fully from KUAL's process group: setsid() so the daemon survives
    # KUAL exiting, and the subshell's exec replaces the shell so there's no
    # lingering pid.
    (
        exec setsid "$DASHBOARD_DIR/dashboard-native" -hw-landscape -suspend-cycle \
            > "$BOOT_LOG" 2>&1
    ) &
    DPID=$!

    sleep 2
    sync 2>/dev/null

    if is_running; then
        screen_msg "Dashboard started" "PID: $(pidof dashboard-native)"
    else
        screen_msg "Dashboard failed" "Check log:" "$LOG_FILE"
    fi
}

dashboard_stop() {
    if ! is_running; then
        screen_msg "Dashboard not running"
        return
    fi

    killall -9 dashboard-native 2>/dev/null || true
    sleep 1

    # Restore Kindle GUI
    start lab126_gui 2>/dev/null || /etc/init.d/framework start 2>/dev/null || true

    # Re-enable screensaver
    lipc-set-prop -i com.lab126.powerd preventScreenSaver 0 2>/dev/null || true

    screen_msg "Dashboard stopped"
}

dashboard_status() {
    if is_running; then
        local pid=$(pidof dashboard-native | tr ' ' ',')
        local uptime=$(ps -o etimes= -p "${pid%%,*}" 2>/dev/null | tr -d ' ')
        screen_msg "Dashboard: RUNNING" "PID(s): $pid" "Uptime: ${uptime:-?}s"
    else
        screen_msg "Dashboard: STOPPED"
    fi
}

case "${1:-status}" in
    start)   dashboard_start ;;
    stop)    dashboard_stop ;;
    restart) dashboard_stop; sleep 1; dashboard_start ;;
    status)  dashboard_status ;;
    *)
        screen_msg "Usage:" "{start|stop|status|restart}"
        exit 1
        ;;
esac
