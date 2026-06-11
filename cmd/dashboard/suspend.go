package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"
)

const (
	wakeAlarmPath  = "/sys/class/rtc/rtc0/wakealarm"
	powerStatePath = "/sys/power/state"

	// wakeGraceMin is the minimum settle time after resume before doing any
	// work, letting WiFi/driver reinit start.
	wakeGraceMin = 2 * time.Second

	// wakeGraceMax is the maximum time to wait for network connectivity
	// after resume before giving up and polling anyway.
	wakeGraceMax = 30 * time.Second

	// activityGracePeriod: don't suspend while the user is actively
	// touching the screen (or shortly after).
	activityGracePeriod = 15 * time.Second

	// earlyWakeMargin: if the device resumes this much earlier than its
	// scheduled wakealarm, treat it as a manual (power button) wake rather
	// than the scheduled RTC alarm.
	earlyWakeMargin = 5 * time.Second

	// buttonWakeGrace: after a manual wake, give the user this long to
	// operate the device before suspending again.
	buttonWakeGrace = 30 * time.Second

	// pcViewKick: while in the PC view during a button-wake grace period,
	// extend the grace by this much each check instead of expiring it.
	pcViewKick = 10 * time.Second
)

// lastActivityNano holds the UnixNano timestamp of the last touch/click,
// updated from UI callbacks in app.go.
var lastActivityNano atomic.Int64

// suppressBrightnessSync, while true, makes handleBrightnessState a no-op.
// Set during post-resume polls so waking doesn't change the frontlight.
var suppressBrightnessSync atomic.Bool

// buttonWakeDeadline, when nonzero, holds the UnixNano time until which a
// manual (power button) wake keeps the device awake regardless of view.
var buttonWakeDeadline atomic.Int64

// markActivity records a touch/click as "now", deferring suspend.
func markActivity() {
	lastActivityNano.Store(time.Now().UnixNano())
}

func timeSinceActivity() time.Duration {
	return time.Since(time.Unix(0, lastActivityNano.Load()))
}

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

// waitForNetwork blocks until outbound connectivity is available or maxWait
// elapses, whichever comes first.
func waitForNetwork(maxWait time.Duration) {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 2*time.Second)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// runSuspendCycle suspends to RAM between minute boundaries, waking via RTC
// alarm at (or just after) each wall-clock minute to refresh the clock and
// poll HA/PC status. If the wakealarm can't be set, it stays awake for that
// cycle instead of suspending — never suspend without a confirmed wake source.
func runSuspendCycle(d *Dashboard) {
	for {
		if idle := timeSinceActivity(); idle < activityGracePeriod {
			log.Printf("suspend: deferring, idle=%v < %v", idle, activityGracePeriod)
			time.Sleep(activityGracePeriod - idle)
			continue
		}

		view := d.CurrentView()
		if dl := buttonWakeDeadline.Load(); dl != 0 {
			now := time.Now()
			if view == ViewLauncher {
				log.Printf("suspend: button-wake grace, currentView=launcher - kicking +%v", pcViewKick)
				buttonWakeDeadline.Store(now.Add(pcViewKick).UnixNano())
				time.Sleep(pcViewKick)
				continue
			}
			if deadline := time.Unix(0, dl); now.Before(deadline) {
				log.Printf("suspend: button-wake grace, %v remaining", time.Until(deadline))
				time.Sleep(2 * time.Second)
				continue
			}
			// Grace expired - clear it and suspend below regardless of view.
			buttonWakeDeadline.Store(0)
		} else if view != ViewHome {
			// Don't suspend while the user is browsing other views -
			// just wait and re-check.
			log.Printf("suspend: deferring, currentView=%d != %d", view, ViewHome)
			time.Sleep(2 * time.Second)
			continue
		}

		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		wait := time.Until(next)
		if wait < 5*time.Second {
			wait += time.Minute
		}

		preSuspendBrightness := readBrightness()

		scheduledSleep := wait + 2*time.Second
		log.Printf("suspend: suspending for %v (wakealarm +%v)", wait, scheduledSleep)
		if err := setWakeAlarm(scheduledSleep); err != nil {
			log.Printf("suspend: %v — staying awake this cycle", err)
			time.Sleep(wait)
		} else {
			suspendStart := time.Now()
			if err := suspendToRAM(); err != nil {
				log.Printf("suspend: %v — staying awake this cycle", err)
				time.Sleep(wait)
			} else {
				// The backlight driver may reset/restore brightness on resume -
				// force it back to the pre-suspend value so the frontlight
				// doesn't flash on.
				writeBrightness(preSuspendBrightness)

				// If we resumed well before the scheduled wakealarm, this was
				// a manual (power button) wake - give the user a window to
				// operate the device.
				if elapsed := time.Since(suspendStart); elapsed < scheduledSleep-earlyWakeMargin {
					log.Printf("suspend: early wake (%v < %v) - button-wake grace %v", elapsed, scheduledSleep, buttonWakeGrace)
					buttonWakeDeadline.Store(time.Now().Add(buttonWakeGrace).UnixNano())
				}

				// Resume happens asynchronously (WiFi firmware reload, driver
				// reinit) - give the device a moment to settle, then poll for
				// connectivity before doing any work, or it can hang.
				time.Sleep(wakeGraceMin)
				log.Printf("suspend: resumed, waiting up to %v for network", wakeGraceMax-wakeGraceMin)
				waitForNetwork(wakeGraceMax - wakeGraceMin)
			}
		}

		// Suppress brightness sync for the whole refresh - clock/redraw and
		// status polling shouldn't flash the frontlight on.
		suppressBrightnessSync.Store(true)

		d.RefreshVisibleView(time.Now())

		if hassClient != nil {
			err := hassClient.fetchAll()
			if err != nil {
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

		suppressBrightnessSync.Store(false)
	}
}
