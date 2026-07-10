package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// Key codes from linux/input-event-codes.h
const (
	keyPower = 116  // KEY_POWER
	keySleep = 142  // KEY_SLEEP
	keyWake  = 143  // KEY_WAKEUP
	evKey    = 0x01 // EV_KEY event type
)

// inputEvent is 16 bytes on 32-bit ARM (Kindle).
// struct input_event {
//   struct timeval { __kernel_time_t tv_sec; __kernel_suseconds_t tv_usec; } time;  // 8 bytes
//   __u16 type;    // 2 bytes
//   __u16 code;    // 2 bytes
//   __s32 value;   // 4 bytes
// };
type inputEvent struct {
	_     [8]byte // struct timeval (padding)
	Type  uint16
	Code  uint16
	Value int32
}

// WatchPowerButton monitors the Kindle's gpio-keys input device for power
// button events. When KEY_POWER is pressed, it:
//   - Marks activity (deferring our own suspend cycle)
//   - Jumps to ViewHome (the rest/clock screen)
//   - Restores brightness if it was dimmed
//
// The goal is to give the user the rest screen immediately, before the
// Kindle OS's power management layer can suspend the device.
func WatchPowerButton(ctx context.Context, d *Dashboard) {
	devices := scanInputDevices()
	if len(devices) == 0 {
		log.Printf("powerbutton: no gpio-keys input device found — disabling")
		return
	}

	log.Printf("powerbutton: monitoring %d device(s) for KEY_POWER", len(devices))

	// Start one goroutine per device so we never miss the button
	// even if one device path is stale.
	for _, path := range devices {
		go watchDevice(ctx, path, d)
	}

	<-ctx.Done()
}

// watchDevice reads input events from a single device file in a blocking loop.
func watchDevice(ctx context.Context, path string, d *Dashboard) {
	fd, err := openBlocking(path)
	if err != nil {
		log.Printf("powerbutton: open %s: %v", path, err)
		return
	}
	defer fd.Close()

	// The read below blocks indefinitely, so a ctx cancel alone can't unblock
	// it — and while parked it still holds the exclusive EVIOCGRAB. Close the
	// fd on cancel: that releases the grab AND makes Read return an error so
	// this goroutine exits (instead of leaking across an in-app restart, where
	// a second exclusive grabber would then fight this one for the button).
	go func() {
		<-ctx.Done()
		fd.Close()
	}()

	log.Printf("powerbutton: watching %s", path)

	// 16-byte buffer = one input_event on 32-bit ARM
	buf := make([]byte, 16)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := fd.Read(buf)
		if err != nil {
			log.Printf("powerbutton: read %s: %v — stopping", path, err)
			return
		}
		if n < 16 {
			continue
		}

		ev := (*inputEvent)(unsafe.Pointer(&buf[0]))
		if ev.Type != evKey {
			continue
		}
		if ev.Code != keyPower && ev.Code != keySleep && ev.Code != keyWake {
			continue
		}

		// KEY_PRESS (value=1), not KEY_RELEASE (0) or KEY_HOLD (2+)
		if ev.Value != 1 {
			continue
		}

		log.Printf("powerbutton: %s press detected on %s", keyName(ev.Code), path)

		// Jump to ViewHome (rest screen) on the UI thread.
		d.runOnUI(func() {
			log.Printf("powerbutton: jumping to ViewHome")
			d.showView(ViewHome)
		})

		// Check if we are asleep (screen dimmed) or awake.
		saved := readBrightness()
		if saved <= 1 {
			// We were asleep or dimming, so this is a wake press.
			maxB := readMaxBrightness()
			if maxB > 0 {
				restoreVal := maxB * 60 / 100 // 60%
				log.Printf("powerbutton: restoring brightness from %d to %d", saved, restoreVal)
				writeBrightness(restoreVal)
				d.UpdateBrightnessValue(restoreVal)
			}
			// Mark activity to give the suspend cycle a moment to pick up.
			markActivity()
		} else {
			// We were awake, so this is a sleep press.
			log.Printf("powerbutton: device is awake, forcing suspend")
			clearActivity()
			select {
			case forceSuspendCh <- struct{}{}:
			default:
			}
		}
	}
}

// openBlocking opens an input device for blocking reads and grabs it exclusively.
func openBlocking(path string) (*os.File, error) {
	fd, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	// EVIOCGRAB = _IOW('E', 0x90, int) -> 0x40044590
	_, _, e1 := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), 0x40044590, 1)
	if e1 != 0 {
		log.Printf("powerbutton: warning: failed to grab %s exclusively (err=%d)", path, e1)
	}
	return fd, nil
}

// scanInputDevices finds the gpio-keys input event device(s) on a Kindle.
// It tries multiple approaches since sysfs paths vary by firmware version:
//
//  1. Parse /proc/bus/input/devices for "gpio-keys" handlers
//  2. Scan /dev/input/event* and check sysfs name for gpio-keys
//  3. Fallback to known paths
func scanInputDevices() []string {
	// Strategy 1: Parse /proc/bus/input/devices — most reliable
	if devices := parseProcInput(); len(devices) > 0 {
		return devices
	}

	// Strategy 2: Scan /dev/input/event* and check sysfs device name
	if devices := scanDevInput(); len(devices) > 0 {
		return devices
	}

	// Strategy 3: Known GPIO paths on various Kindle models
	known := []string{
		"/dev/input/event0", // Most common (first input device = gpio-keys)
		"/dev/input/event1", // Sometimes gpio-keys is event1
	}
	var fallback []string
	for _, path := range known {
		if _, err := os.Stat(path); err == nil {
			fallback = append(fallback, path)
		}
	}
	return fallback
}

// parseProcInput reads /proc/bus/input/devices and extracts the event
// handler for "gpio-keys" or "Power Button" or "keypad".
func parseProcInput() []string {
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		return nil
	}

	content := string(data)
	blocks := strings.Split(content, "\n\n")

	var result []string
	for _, block := range blocks {
		if !strings.Contains(block, "gpio-keys") &&
			!strings.Contains(block, "Power Button") &&
			!strings.Contains(block, "keypad") &&
			!strings.Contains(block, "KEY_POWER") {
			continue
		}

		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "H:") {
				// Parse "Handlers=sysrq kbd event3" etc.
				// Look for eventN in the handlers
				fields := strings.Fields(line)
				for _, f := range fields {
					if strings.HasPrefix(f, "event") {
						path := filepath.Join("/dev/input", f)
						result = append(result, path)
					}
				}
			}
		}
	}
	return result
}

// scanDevInput scans /dev/input/event* and checks the sysfs device name.
// The naming under /sys/class/input/ is:
//   /sys/class/input/inputN/name  — human-readable input device name
//   /sys/class/input/eventN/      — symlink to the event device under inputN
// Typically /sys/class/input/eventN/device/name resolves to the same file
// as /sys/class/input/inputN/name, but the inputN path is more direct.
func scanDevInput() []string {
	entries, err := os.ReadDir("/dev/input")
	if err != nil {
		return nil
	}

	var result []string
	for _, e := range entries {
		devName := e.Name()
		if !strings.HasPrefix(devName, "event") {
			continue
		}
		path := filepath.Join("/dev/input", devName)

		// Extract 'inputN' from 'eventN' to read the input device name.
		// On Kindle: event0 → input0, event1 → input1
		inputName := "input" + strings.TrimPrefix(devName, "event")

		// Read device name from sysfs — multiple possible paths
		devNameStr := readSysfsString(filepath.Join("/sys/class/input", inputName, "name"))
		if devNameStr == "" {
			// Fallback: try via the event device symlink
			devNameStr = readSysfsString(filepath.Join("/sys/class/input", devName, "device/name"))
		}
		if devNameStr == "" {
			devNameStr = readSysfsString(filepath.Join("/sys/class/input", devName, "device/device/name"))
		}

		// Match: Kindle power button devices, gpio-keys, or anything with "power" or "key"
		// We specifically want snvs-powerkey and bd71827-power, not the touchscreen (cyttsp5_mt).
		if strings.Contains(devNameStr, "power") ||
			strings.Contains(devNameStr, "Power") ||
			strings.Contains(devNameStr, "gpio-keys") ||
			strings.Contains(devNameStr, "keypad") ||
			strings.Contains(devNameStr, "KEY") {
			result = append(result, path)
		}
	}
	return result
}

func readSysfsString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func keyName(code uint16) string {
	switch code {
	case keyPower:
		return "KEY_POWER"
	case keySleep:
		return "KEY_SLEEP"
	case keyWake:
		return "KEY_WAKEUP"
	default:
		return fmt.Sprintf("KEY_%d", code)
	}
}

// We use unsafe.Pointer for event struct access rather than encoding/binary
// for speed. The struct layout is verified to be 16 bytes on 32-bit ARM.
