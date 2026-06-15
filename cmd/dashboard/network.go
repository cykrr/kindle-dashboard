package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// ── WiFi ────────────────────────────────────────────────────────────────────

// wifiState returns "On" or "Off" based on wlan0 interface state.
func wifiState() string {
	out, err := exec.Command("ip", "link", "show", "wlan0").Output()
	if err != nil {
		return "Off"
	}
	if strings.Contains(string(out), "state UP") || strings.Contains(string(out), "LOWER_UP") {
		return "On"
	}
	return "Off"
}

// wifiOn enables WiFi: bring interface up, start wifid daemon.
func wifiOn() error {
	log.Printf("network: wifi on")
	cmds := [][]string{
		{"ip", "link", "set", "wlan0", "up"},
		{"initctl", "start", "wifid"},
	}
	var errs []string
	for _, cmd := range cmds {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("wifi on: %s", strings.Join(errs, "; "))
	}
	return nil
}

// wifiOff disables WiFi: stop wifid, kill wpa_supplicant, bring interface down.
func wifiOff() error {
	log.Printf("network: wifi off")
	cmds := [][]string{
		{"initctl", "stop", "wifid"},
		{"killall", "wpa_supplicant"},
		{"ip", "link", "set", "wlan0", "down"},
	}
	var errs []string
	for _, cmd := range cmds {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("wifi off: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ── USB Ethernet (g_ether) ──────────────────────────────────────────────────

// usbEthState returns "On" or "Off" based on usb0 interface state.
func usbEthState() string {
	out, err := exec.Command("ip", "link", "show", "usb0").Output()
	if err != nil {
		return "Off"
	}
	if strings.Contains(string(out), "state UP") || strings.Contains(string(out), "LOWER_UP") {
		return "On"
	}
	return "Off"
}

// usbEthOn loads the g_ether module (replacing g_mass_storage if loaded).
func usbEthOn() error {
	log.Printf("network: usb ethernet on")
	cmds := [][]string{
		{"rmmod", "g_mass_storage"},
		{"modprobe", "g_ether"},
	}
	var errs []string
	for _, cmd := range cmds {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("usb eth on: %s", strings.Join(errs, "; "))
	}
	// Once g_ether is loaded, add a default route via USB peer if reachable.
	if out, err := exec.Command("ping", "-c1", "-W1", "192.168.11.2").Output(); err == nil && len(out) > 0 {
		exec.Command("ip", "route", "add", "default", "via", "192.168.11.2", "dev", "usb0").Run()
	}
	return nil
}

// usbEthOff unloads the g_ether module.
func usbEthOff() error {
	log.Printf("network: usb ethernet off")
	if out, err := exec.Command("rmmod", "g_ether").CombinedOutput(); err != nil {
		return fmt.Errorf("usb eth off: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ── Bluetooth ───────────────────────────────────────────────────────────────

// btState returns "On" or "Off" based on btfd service status.
func btState() string {
	out, err := exec.Command("initctl", "status", "btfd").Output()
	if err != nil {
		return "Off"
	}
	if strings.Contains(string(out), "start/running") {
		return "On"
	}
	return "Off"
}

// btOn enables Bluetooth services.
func btOn() error {
	log.Printf("network: bluetooth on")
	cmds := [][]string{
		{"initctl", "start", "btfd"},
		{"initctl", "start", "btd"},
	}
	var errs []string
	for _, cmd := range cmds {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("bt on: %s", strings.Join(errs, "; "))
	}
	return nil
}

// btOff disables Bluetooth services.
func btOff() error {
	log.Printf("network: bluetooth off")
	cmds := [][]string{
		{"initctl", "stop", "btfd"},
		{"initctl", "stop", "btd"},
	}
	var errs []string
	for _, cmd := range cmds {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("bt off: %s", strings.Join(errs, "; "))
	}
	return nil
}
