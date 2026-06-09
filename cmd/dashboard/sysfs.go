package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	brightnessFile = "/sys/devices/soc0/bl/backlight/bl/brightness"
	maxBrightFile  = "/sys/devices/soc0/bl/backlight/bl/max_brightness"
	batteryCap     = "/sys/class/power_supply/bd71827_bat/capacity"
	batteryHealth  = "/sys/class/power_supply/bd71827_bat/health"
	batteryStatus  = "/sys/class/power_supply/bd71827_bat/status"
)

func readBrightness() int    { return readIntFile(brightnessFile) }
func readMaxBrightness() int { return readIntFile(maxBrightFile) }
func writeBrightness(val int) {
	os.WriteFile(brightnessFile, []byte(fmt.Sprintf("%d\n", val)), 0644)
}
func readBatteryCapacity() string { return readFile(batteryCap) }
func readBatteryHealth() string   { return readFile(batteryHealth) }
func readBatteryStatus() string   { return readFile(batteryStatus) }

func WatchBatteryCapacity(ctx context.Context, callback func(string)) {
	// Initial read
	callback(readBatteryCapacity())

	fd, err := unix.Open(batteryCap, unix.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("failed to open battery capacity file for watching: %v\n", err)
		return
	}
	defer unix.Close(fd)

	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		fmt.Printf("failed to create epoll: %v\n", err)
		return
	}
	defer unix.Close(epfd)

	event := unix.EpollEvent{
		Events: unix.EPOLLPRI | unix.EPOLLERR,
		Fd:     int32(fd),
	}
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		fmt.Printf("failed to add fd to epoll: %v\n", err)
		return
	}

	events := make([]unix.EpollEvent, 1)
	buf := make([]byte, 16)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := unix.EpollWait(epfd, events, 1000)
			if err != nil {
				if err == unix.EINTR {
					continue
				}
				fmt.Printf("epoll wait error: %v\n", err)
				return
			}

			if n > 0 {
				// Seek to beginning and read to clear the POLLPRI condition
				nRead, err := unix.Pread(fd, buf, 0)
				if err != nil {
					fmt.Printf("failed to read battery capacity: %v\n", err)
					continue
				}
				val := strings.TrimSpace(string(buf[:nRead]))
				callback(val)
			}
		}
	}
}

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
	return strings.TrimSpace(string(b))
}
