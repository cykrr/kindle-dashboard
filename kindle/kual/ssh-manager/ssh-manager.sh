#!/bin/sh
# SSH Manager - start/stop/status for dropbear SSH + USB networking on Kindle

DROPBEAR="/mnt/us/koreader/dropbear"
DB_CLIENT="/mnt/us/koreader/dbclient"
KOREADER_DIR="/mnt/us/koreader"
KOREADER_SSH_DIR="$KOREADER_DIR/settings/SSH"
SSH_DIR="/mnt/us/extensions/ssh-manager"
# Log on /mnt/us (vfat) so it's readable from the PC over USB (I:\tmp\log.txt).
LOG_FILE="/mnt/us/tmp/log.txt"
PORT=2222
USB_IP="192.168.15.200"
USB_SUBNET="255.255.255.0"

# Logging toggle: set LOG_ENABLED=true in config.cfg to enable
LOG_ENABLED=false
[ -f "$SSH_DIR/config.cfg" ] && . "$SSH_DIR/config.cfg"

# Ensure the log directory exists (first run after a fresh install).
mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null

# Ensure KOReader's SSH settings directory exists
mkdir -p "$KOREADER_SSH_DIR" 2>/dev/null

log() {
    [ "$LOG_ENABLED" = "true" ] && echo "$(date): $*" >> "$LOG_FILE"
}

# logf: always record (bypasses the LOG_ENABLED toggle) and flush to disk, so
# the entry survives on the vfat and is readable from the PC over USB even on
# a "successful" start.
logf() {
    echo "$(date): $*" >> "$LOG_FILE"
    sync 2>/dev/null
}

screen_msg() {
    eips -c 2>/dev/null
    eips 1 1 "$1"
    eips 1 2 "$2"
    eips 1 3 "$3"
    eips 1 4 "$4"
}

get_wifi_ip() {
    ifconfig wlan0 2>/dev/null | grep 'inet addr' | awk '{print $2}' | cut -d: -f2
}

get_usb_ip() {
    ifconfig usb0 2>/dev/null | grep 'inet addr' | awk '{print $2}' | cut -d: -f2
}

port_listening() {
    command -v netstat >/dev/null 2>&1 || return 0  # can't check -> assume ok
    netstat -ln 2>/dev/null | grep -qE "[:.]$PORT[[:space:]]"
}

# The Kindle firmware firewall DROPs inbound connections on wlan0 by default
# (which is why USB net works but WiFi SSH times out). Open our port on start.
open_firewall() {
    command -v iptables >/dev/null 2>&1 || { logf "firewall: no iptables"; return 0; }
    # Remove any existing copy first so we don't stack duplicate rules
    # (busybox iptables may lack -C, so delete-then-insert instead of check).
    iptables -D INPUT -p tcp --dport "$PORT" -j ACCEPT 2>/dev/null
    if iptables -I INPUT 1 -p tcp --dport "$PORT" -j ACCEPT 2>/dev/null; then
        logf "firewall: opened tcp $PORT"
    else
        logf "firewall: FAILED to open tcp $PORT"
    fi
}

close_firewall() {
    command -v iptables >/dev/null 2>&1 || return 0
    while iptables -D INPUT -p tcp --dport "$PORT" -j ACCEPT 2>/dev/null; do :; done
    logf "firewall: closed tcp $PORT"
}

is_running() {
    # Authoritative: a dropbear process exists AND our port is actually bound.
    # Guards against a ghost dropbear left on the wrong port by an old run.
    pidof dropbear >/dev/null 2>&1 || return 1
    port_listening
}

usbnet_is_up() {
    ifconfig usb0 2>/dev/null | grep -q 'inet addr'
}

cleanup_gadget() {
    # Tear down any existing configfs gadget
    if [ -d /sys/kernel/config/usb_gadget/g1 ]; then
        echo '' > /sys/kernel/config/usb_gadget/g1/UDC 2>/dev/null
        rm -f /sys/kernel/config/usb_gadget/g1/os_desc/c.1 2>/dev/null
        rm -f /sys/kernel/config/usb_gadget/g1/configs/c.1/rndis.usb0 2>/dev/null
        rmdir /sys/kernel/config/usb_gadget/g1/configs/c.1/strings/0x409 2>/dev/null
        rmdir /sys/kernel/config/usb_gadget/g1/configs/c.1 2>/dev/null
        rmdir /sys/kernel/config/usb_gadget/g1/functions/rndis.usb0 2>/dev/null
        rmdir /sys/kernel/config/usb_gadget/g1/strings/0x409 2>/dev/null
        rmdir /sys/kernel/config/usb_gadget/g1 2>/dev/null
    fi
}

# --- USB Networking (configfs RNDIS with Windows OS descriptors) ---

usbnet_start() {
    log "starting usb networking"
    if usbnet_is_up; then
        screen_msg "USB net already up" "IP: $(get_usb_ip)"
        return
    fi

    # Clean up any existing gadget or legacy modules
    cleanup_gadget
    rmmod g_ether usb_f_rndis u_ether 2>/dev/null

    # Mount configfs if needed
    mount | grep -q configfs || mount -t configfs none /sys/kernel/config

    # Create RNDIS gadget with Microsoft OS descriptors for Windows
    mkdir /sys/kernel/config/usb_gadget/g1
    cd /sys/kernel/config/usb_gadget/g1

    echo 0x1949 > idVendor
    echo 0x0002 > idProduct   # 0x0002 (not 0x0001): forces Windows to re-enumerate fresh, skipping any cached Code 28 from earlier broken attempts
    echo 0x0100 > bcdDevice
    echo 0x0200 > bcdUSB

    mkdir -p strings/0x409
    echo '1234567890' > strings/0x409/serialnumber
    echo 'Amazon' > strings/0x409/manufacturer
    echo 'Kindle' > strings/0x409/product

    # Microsoft OS Descriptors (required for Windows RNDIS auto-detection)
    mkdir -p os_desc
    echo 0xcd > os_desc/b_vendor_code
    echo 'MSFT100' > os_desc/qw_sign

    # RNDIS function
    mkdir -p functions/rndis.usb0
    echo '00:DC:CA:11:22:44' > functions/rndis.usb0/dev_addr
    echo '00:DC:CA:11:22:33' > functions/rndis.usb0/host_addr

    # Windows binds RNDIS by MS-OS compatible ID, NOT by USB class codes.
    # Without these three, Windows never matches and shows Code 28 (no driver).
    echo 'RNDIS'   > functions/rndis.usb0/os_desc/interface.rndis/compatible_id
    echo '5162001' > functions/rndis.usb0/os_desc/interface.rndis/sub_compatible_id

    # Configuration
    mkdir -p configs/c.1/strings/0x409
    echo 'RNDIS' > configs/c.1/strings/0x409/configuration
    echo 120 > configs/c.1/MaxPower

    # Link function to config
    ln -s functions/rndis.usb0 configs/c.1

    # Link config to os_desc so Windows sees the extended descriptors
    ln -s configs/c.1 os_desc/c.1

    # Enable the MS-OS descriptor set (device advertises it at enumeration).
    # Must be on, or the compatible_id above is never sent -> Code 28.
    echo 1 > os_desc/use

    # Bind UDC
    echo ci_hdrc.0 > UDC

    sleep 1
    ifconfig usb0 "$USB_IP" netmask "$USB_SUBNET" up
    sleep 1

    if usbnet_is_up; then
        local ip=$(get_usb_ip)
        screen_msg "USB net started" "IP: $ip" "Unplug/replug USB" "on Windows side"
        log "started on $ip"
    else
        screen_msg "USB net failed" "Try re-plug USB cable"
        log "FAILED to start"
    fi
}

usbnet_stop() {
    log "stopping usb networking"
    ifconfig usb0 down 2>/dev/null
    cleanup_gadget
    screen_msg "USB net stopped"
    log "stopped"
}

usbnet_status() {
    if usbnet_is_up; then
        local ip=$(get_usb_ip)
        screen_msg "USB net: UP" "IP: $ip" "PC: 192.168.15.201"
    else
        screen_msg "USB net: DOWN"
    fi
}

# --- SSH ---

ssh_status() {
    log "status check"
    if is_running; then
        local wip=$(get_wifi_ip)
        local uip=$(get_usb_ip)
        local pids=$(pidof dropbear | tr ' ' ',')
        screen_msg "SSH: RUNNING" "PID(s): $pids" "WiFi: ${wip:-n/a}:$PORT" "USB: ${uip:-n/a}:$PORT"
        echo "SSH running - WiFi: ${wip:-none}:$PORT, USB: ${uip:-none}:$PORT" > /tmp/ssh-status.txt
    elif pidof dropbear >/dev/null 2>&1; then
        local pids=$(pidof dropbear | tr ' ' ',')
        screen_msg "SSH: process up" "but NOT listening on $PORT" "PID(s): $pids" "Try Stop then Start"
        echo "dropbear pid $pids but not listening on $PORT" > /tmp/ssh-status.txt
    else
        screen_msg "SSH: STOPPED"
        echo "SSH not running" > /tmp/ssh-status.txt
    fi
}

ssh_showlog() {
    if [ ! -f "$LOG_FILE" ]; then
        screen_msg "No log yet" "$LOG_FILE" "Failures are logged here" "even if logging is off"
        return
    fi
    # Dump the last lines of the log straight to the eink screen.
    eips -c 2>/dev/null
    i=0
    tail -n 10 "$LOG_FILE" | while IFS= read -r ln; do
        eips 0 $i "$ln"
        i=$((i + 1))
    done
}

ssh_start() {
    log "starting dropbear"
    if is_running; then
        screen_msg "SSH already running"
        return
    fi

    # Pre-flight: dropbear binary must exist and be executable.
    if [ ! -x "$DROPBEAR" ]; then
        screen_msg "SSH failed to start" "dropbear missing at:" "$DROPBEAR"
        logf "FAILED: dropbear not found/executable at $DROPBEAR"
        return
    fi

    # Clear a ghost dropbear that exists but isn't listening on our port (e.g.
    # left on port 22 by an earlier broken start) so we can bind cleanly.
    if pidof dropbear >/dev/null 2>&1 && ! port_listening; then
        log "killing stale dropbear (not listening on $PORT)"
        killall dropbear 2>/dev/null
        sleep 1
    fi

    # --- CRITICAL: KOReader's patched dropbear reads settings/SSH/authorized_keys
    # as a RELATIVE path from cwd. We MUST cd to KOREADER_DIR for it to find the
    # authorized_keys file (and host keys stored there).
    # Ref: https://github.com/koreader/koreader/blob/master/plugins/SSH.koplugin/main.lua
    cd "$KOREADER_DIR" || {
        screen_msg "SSH failed to start" "Cannot cd to:" "$KOREADER_DIR"
        logf "FAILED: cannot cd to $KOREADER_DIR"
        return
    }

    # Ensure host key directory exists — dropbear -R will generate keys here
    # because cwd is now $KOREADER_DIR and the patched dropbear knows the path.
    mkdir -p "$KOREADER_SSH_DIR" 2>/dev/null

    # A previous run may have left an empty (0-byte) host key that dropbear
    # cannot parse — remove it so -R regenerates cleanly.
    if [ -f "$KOREADER_SSH_DIR/dropbear_rsa_host_key" ] && [ ! -s "$KOREADER_SSH_DIR/dropbear_rsa_host_key" ]; then
        log "removing empty host key"
        rm -f "$KOREADER_SSH_DIR/dropbear_rsa_host_key"
    fi

    [ -s "$KOREADER_SSH_DIR/dropbear_rsa_host_key" ] || screen_msg "Generating SSH key..." "" "Please wait..."

    # Start dropbear from KOREADER_DIR so the patched binary can find its
    # settings/SSH/ subdirectory for both authorized_keys and host keys.
    #
    #   -R  generate the host key at startup if missing (saved to cwd/settings/SSH/)
    #   -E  log to stderr instead of syslog
    #   -s  enforce key-only auth (disable password logins)
    #
    # We do NOT use -F (foreground) because we background with setsid/nohup + &.
    # We do NOT pass -r; the patched dropbear knows the relative key path.
    logf "start: cd $KOREADER_DIR && $DROPBEAR -R -E -s -p $PORT (detached)"
    (
        cd "$KOREADER_DIR" || exit 1
        if command -v setsid >/dev/null 2>&1; then
            exec setsid $DROPBEAR -R -E -s -p $PORT -P /tmp/dropbear.pid >> "$LOG_FILE" 2>&1
        else
            exec nohup $DROPBEAR -R -E -s -p $PORT -P /tmp/dropbear.pid >> "$LOG_FILE" 2>&1
        fi
    ) &

    sleep 2
    sync 2>/dev/null
    if is_running; then
        local wip=$(get_wifi_ip)
        local uip=$(get_usb_ip)
        screen_msg "SSH started!" "WiFi: ${wip:-n/a}:$PORT" "USB: ${uip:-n/a}:$PORT"
        logf "OK started, listening on $PORT (WiFi ${wip:-n/a}, USB ${uip:-n/a}, pid $(pidof dropbear | tr ' ' ','))"
        open_firewall
        # Reachability diagnostics: bind address, interfaces, firewall. If you
        # can't connect despite this, the answer is usually one of these.
        logf "listen: $(netstat -ln 2>/dev/null | grep -E "[:.]$PORT[[:space:]]" | tr '\n' '|')"
        logf "wlan0: $(ifconfig wlan0 2>/dev/null | grep -E 'inet addr|UP' | tr '\n' '|')"
        logf "route: $(ip route 2>/dev/null | tr '\n' '|')"
        if command -v iptables >/dev/null 2>&1; then
            logf "iptables INPUT: $(iptables -S INPUT 2>/dev/null | tr '\n' '|')"
        else
            logf "iptables: not present"
        fi
        sync 2>/dev/null
    else
        # Reason: dropbear's own output is now in the logfile — surface its tail.
        reason=$(tail -n 5 "$LOG_FILE" 2>/dev/null | tr '\n' ' ')
        if [ ! -s "$KOREADER_SSH_DIR/dropbear_rsa_host_key" ]; then
            reason="host key missing/empty; ${reason:-see log}"
        elif pidof dropbear >/dev/null 2>&1 && ! port_listening; then
            reason="dropbear up but not listening on $PORT; ${reason:-see log}"
        fi
        [ -z "$reason" ] && reason="see $LOG_FILE"
        screen_msg "SSH failed to start" "$reason" "Log: $LOG_FILE"
        logf "FAILED to start: reason=$reason"
    fi
}

ssh_stop() {
    log "stopping dropbear"
    close_firewall
    local pids=$(pidof dropbear)
    if [ -n "$pids" ]; then
        killall dropbear 2>/dev/null
        sleep 1
        screen_msg "SSH stopped"
        log "stopped"
    else
        screen_msg "SSH not running"
    fi
}

ssh_ip() {
    local wip=$(get_wifi_ip)
    local uip=$(get_usb_ip)
    screen_msg "WiFi: ${wip:-none}:$PORT" "USB: ${uip:-none}:$PORT" "SSH: $(is_running && echo RUNNING || echo STOPPED)"
    echo "WiFi: $wip:$PORT" > /tmp/ssh-status.txt
    echo "USB: $uip:$PORT" >> /tmp/ssh-status.txt
}

ssh_toggle_log() {
    if grep -q "LOG_ENABLED=true" "$SSH_DIR/config.cfg" 2>/dev/null; then
        echo "LOG_ENABLED=false" > "$SSH_DIR/config.cfg"
        screen_msg "Logging: OFF"
    else
        echo "LOG_ENABLED=true" > "$SSH_DIR/config.cfg"
        screen_msg "Logging: ON"
    fi
}

case "${1:-status}" in
    start)     ssh_start ;;
    stop)      ssh_stop ;;
    restart)   ssh_stop; sleep 1; ssh_start ;;
    status)    ssh_status ;;
    ip)        ssh_ip ;;
    showlog)   ssh_showlog ;;
    togglog)   ssh_toggle_log ;;
    usbstart)  usbnet_start ;;
    usbstop)   usbnet_stop ;;
    usbstatus) usbnet_status ;;
    *)
        screen_msg "Usage:" "{start|stop|status|restart|ip|showlog}" "{usbstart|usbstop|usbstatus}" "{togglog}"
        exit 1
        ;;
esac
