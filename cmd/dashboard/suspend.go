package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const (
	wakeAlarmPath  = "/sys/class/rtc/rtc0/wakealarm"
	powerStatePath = "/sys/power/state"
)

// setWakeAlarm clears any pending RTC alarm and schedules a new one d from now.
func setWakeAlarm(d time.Duration) error {
	if err := os.WriteFile(wakeAlarmPath, []byte("0"), 0644); err != nil {
		return fmt.Errorf("clear wakealarm: %w", err)
	}
	secs := int(d.Round(time.Second) / time.Second)
	if secs < 1 {
		secs = 1
	}
	if err := os.WriteFile(wakeAlarmPath, []byte(fmt.Sprintf("+%d", secs)), 0644); err != nil {
		return fmt.Errorf("set wakealarm: %w", err)
	}
	return nil
}

// suspendToRAM suspends the device to RAM. Returns once the device resumes.
func suspendToRAM() error {
	return os.WriteFile(powerStatePath, []byte("mem"), 0644)
}

// runSuspendCycle suspends to RAM between minute boundaries, waking via RTC
// alarm at (or just after) each wall-clock minute to refresh the clock and
// poll HA/PC status. If the wakealarm can't be set, it stays awake for that
// cycle instead of suspending — never suspend without a confirmed wake source.
func runSuspendCycle(d *Dashboard) {
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		wait := time.Until(next)
		if wait < 5*time.Second {
			wait += time.Minute
		}

		if err := setWakeAlarm(wait + 2*time.Second); err != nil {
			log.Printf("suspend: %v — staying awake this cycle", err)
			time.Sleep(wait)
		} else if err := suspendToRAM(); err != nil {
			log.Printf("suspend: %v — staying awake this cycle", err)
			time.Sleep(wait)
		}

		d.UpdateClock(time.Now())
		d.Redraw()

		if hassClient != nil {
			if err := hassClient.fetchAll(); err != nil {
				log.Printf("hass: post-resume fetch: %v", err)
			} else {
				hassClient.setConnStatus("Connected")
			}
		}
		if pcMacroClient != nil {
			if err := pcMacroClient.RefreshStatus(); err != nil {
				log.Printf("pc macro: post-resume refresh: %v", err)
			}
		}
	}
}
