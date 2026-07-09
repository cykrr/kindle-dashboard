#!/bin/sh
# SSH Manager - start/stop/status for dropbear SSH + USB networking on Kindle

DROPBEAR="/mnt/us/koreader/dropbear"
DB_CLIENT="/mnt/us/koreader/dbclient"
SSH_DIR="/mnt/us/extensions/ssh-manager"
LOG_FILE="/tmp/ssh-manager.log"
PORT=2222
USB_IP="192.168.15.200"
USB_SUBNET="255.255.255.0"

# Logging toggle: set LOG_ENABLED=true in config.cfg to enable
LOG_ENABLED=false
[ -f "$SSH_DIR/config.cfg" ] && . "$SSH_DIR/config.cfg"

log() {
    [ "$LOG_ENABLED" = "true" ] && echo "$(date): $*" >> "$LOG_FILE"
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

is_running() {
    pidof dropbear >/dev/null 2>&1
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
    else
        screen_msg "SSH: STOPPED"
        echo "SSH not running" > /tmp/ssh-status.txt
    fi
}

ssh_start() {
    log "starting dropbear"
    if is_running; then
        screen_msg "SSH already running"
        return
    fi

    if [ ! -f "$SSH_DIR/dropbear_rsa_host_key" ]; then
        log "generating host keys"
        screen_msg "Generating SSH keys..." "" "Please wait..."
        $DROPBEAR -R -r "$SSH_DIR/dropbear_rsa_host_key" 2>/dev/null
        sleep 1
    fi

    if [ ! -f "$SSH_DIR/dropbear_rsa_host_key" ]; then
        touch "$SSH_DIR/dropbear_rsa_host_key"
        chmod 600 "$SSH_DIR/dropbear_rsa_host_key"
    fi

    $DROPBEAR -p $PORT -r "$SSH_DIR/dropbear_rsa_host_key" -P /var/run/dropbear.pid 2>> "$LOG_FILE"

    sleep 1
    if is_running; then
        local wip=$(get_wifi_ip)
        local uip=$(get_usb_ip)
        screen_msg "SSH started!" "WiFi: ${wip:-n/a}:$PORT" "USB: ${uip:-n/a}:$PORT"
        log "started - WiFi: $wip:$PORT, USB: $uip:$PORT"
    else
        screen_msg "SSH failed to start"
        log "FAILED to start"
    fi
}

ssh_stop() {
    log "stopping dropbear"
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
    togglog)   ssh_toggle_log ;;
    usbstart)  usbnet_start ;;
    usbstop)   usbnet_stop ;;
    usbstatus) usbnet_status ;;
    *)
        screen_msg "Usage:" "{start|stop|status|restart|ip}" "{usbstart|usbstop|usbstatus}" "{togglog}"
        exit 1
        ;;
esac
