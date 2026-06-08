package main

import (
	"fmt"
	"os"
)

const (
	brightnessFile = "/sys/devices/soc0/bl/backlight/bl/brightness"
	maxBrightFile  = "/sys/devices/soc0/bl/backlight/bl/max_brightness"
	batteryCap     = "/sys/class/power_supply/bd71827_bat/capacity"
	batteryHealth  = "/sys/class/power_supply/bd71827_bat/health"
	batteryStatus  = "/sys/class/power_supply/bd71827_bat/status"
)

func readBrightness() int   { return readIntFile(brightnessFile) }
func readMaxBrightness() int { return readIntFile(maxBrightFile) }
func writeBrightness(val int) {
	os.WriteFile(brightnessFile, []byte(fmt.Sprintf("%d\n", val)), 0644)
}
func readBatteryCapacity() string { return readFile(batteryCap) }
func readBatteryHealth() string   { return readFile(batteryHealth) }
func readBatteryStatus() string   { return readFile(batteryStatus) }

func readIntFile(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var v int
	fmt.Sscanf(string(b), "%d", &v)
	return v
}
func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "N/A"
	}
	return string(b)
}
