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

	// rtcWakeLead wakes before the minute boundary, giving Kindle hardware time
	// to resume so the visible redraw can happen at roughly :00 instead of after.
	rtcWakeLead = 2 * time.Second

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

	// uiRefreshTimeout is how long the suspend loop waits for queued GTK work
	// to run before giving up and continuing. It should normally complete fast.
	uiRefreshTimeout = 5 * time.Second

	// einkRefreshSettle gives the e-ink controller time to finish the visible
	// refresh before we suspend again, avoiding half-painted/ghosted states.
	einkRefreshSettle = 2 * time.Second

	// quietHourStart/End define the window where we skip per-minute wakes and
	// instead sleep until quietWakeHour. During this window the screen is almost
	// certainly not being viewed, so there is no benefit to waking WiFi every
	// minute. The Kindle still wakes once for the redraw at the end of quiet hours.
	quietHourStart = 22 // 10 PM — start of quiet hours
	quietHourEnd   = 6  // 6 AM  — end of quiet hours (wake time)

	// pcConsecutiveFailLimit: once the PC macro endpoint has failed this many
	// consecutive background wakes, we stop polling it until the next successful
	// poll or until the user explicitly opens the launcher view (which calls
	// RefreshStatus directly via Touch/Execute).
	pcConsecutiveFailLimit = 3
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

// ultraSavingMode, when true, enables extra power-saving behaviour:
//   - During quiet hours: physically disables WiFi, skips HA/PC polling entirely
//   - During daytime: wakes every 2 minutes instead of every 1
var ultraSavingMode atomic.Bool

// quietHoursDisabled, when true, forces the normal per-minute daytime cycle
// even during the 22:00-06:00 window. Settings-view toggle for testing the
// regular wake path without waiting for or faking the clock.
var quietHoursDisabled atomic.Bool

var forceSuspendCh = make(chan struct{}, 1)

// markActivity records a touch/click as "now", deferring suspend.
func markActivity() {
	lastActivityNano.Store(time.Now().UnixNano())
}

// clearActivity resets the activity timer so the device can suspend immediately.
func clearActivity() {
	lastActivityNano.Store(0)
}

func timeSinceActivity() time.Duration {
	return time.Since(time.Unix(0, lastActivityNano.Load()))
}

// sleepOrInterrupt sleeps for d. It returns true if it was interrupted by a force suspend.
func sleepOrInterrupt(d time.Duration) bool {
	if d <= 0 {
		return false
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return false
	case <-forceSuspendCh:
		return true
	}
}

// setWakeAlarm clears any pending RTC alarm and schedules a new one d from now.
func setWakeAlarm(d time.Duration) error {
	if err := os.WriteFile(wakeAlarmPath, []byte("0"), 0644); err != nil {
		return fmt.Errorf("clear wakealarm: %w", err)
	}
	secs := wakeAlarmSeconds(d)
	if err := os.WriteFile(wakeAlarmPath, []byte(fmt.Sprintf("+%d", secs)), 0644); err != nil {
		return fmt.Errorf("set wakealarm: %w", err)
	}
	return nil
}

func wakeAlarmSeconds(d time.Duration) int {
	secs := int((d + time.Second - time.Nanosecond) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return secs
}

// suspendToRAM suspends the device to RAM. Returns once the device resumes.
func suspendToRAM() error {
	return os.WriteFile(powerStatePath, []byte("mem"), 0644)
}

// waitForNetwork blocks until outbound connectivity is available or maxWait
// elapses, whichever comes first. It returns false if interrupted by force suspend.
func waitForNetwork(maxWait time.Duration) bool {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
		if sleepOrInterrupt(1 * time.Second) {
			return false
		}
	}
	return true
}

func isEarlyWakeWall(resumedAt, scheduledWakeAt time.Time, margin time.Duration) bool {
	return resumedAt.Before(scheduledWakeAt.Add(-margin))
}

func rememberBrightness(current, saved int) int {
	if current > 0 {
		return current
	}
	return saved
}

func nextRedrawBoundary(now time.Time) time.Time {
	next := now.Truncate(time.Minute).Add(time.Minute)
	if next.Sub(now) < 5*time.Second {
		next = next.Add(time.Minute)
	}
	return next
}

// nextRedrawBoundary2Min returns the next even 2-minute boundary (:00, :02, :04...).
// Used in Ultra Saving Mode to halve the number of wake cycles per hour.
func nextRedrawBoundary2Min(now time.Time) time.Time {
	nextMin := ((now.Minute() / 2) + 1) * 2
	if nextMin >= 60 {
		nextMin = 0
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMin, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(2 * time.Minute)
	}
	if next.Sub(now) < 5*time.Second {
		next = next.Add(2 * time.Minute)
	}
	return next
}

func wakeAlarmDelayForBoundary(now, boundary time.Time) time.Duration {
	wakeAt := boundary.Add(-rtcWakeLead)
	if !wakeAt.After(now) {
		wakeAt = now.Add(time.Second)
	}
	return time.Duration(wakeAlarmSeconds(wakeAt.Sub(now))) * time.Second
}

// isQuietHour reports whether now falls inside the overnight quiet window
// [quietHourStart, midnight) ∪ [midnight, quietHourEnd).
func isQuietHour(now time.Time) bool {
	if quietHoursDisabled.Load() {
		return false
	}
	h := now.Hour()
	return h >= quietHourStart || h < quietHourEnd
}

// nextQuietWakeTime returns the wall-clock time of the next quiet-hour wakeup:
// the quietHourEnd boundary of the coming morning, minus rtcWakeLead.
func nextQuietWakeTime(now time.Time) time.Time {
	wakeDay := now
	if now.Hour() >= quietHourEnd {
		// Already past today's quietHourEnd — the next one is tomorrow.
		wakeDay = now.AddDate(0, 0, 1)
	}
	return time.Date(wakeDay.Year(), wakeDay.Month(), wakeDay.Day(),
		quietHourEnd, 0, 0, 0, now.Location()).Add(-rtcWakeLead)
}

// runSuspendCycle suspends to RAM between minute boundaries, waking via RTC
// alarm at (or just after) each wall-clock minute to refresh the clock and
// poll HA/PC status. If the wakealarm can't be set, it stays awake for that
// cycle instead of suspending — never suspend without a confirmed wake source.
//
// Power-saving improvements over the original:
//
//  1. Quiet hours (22:00–06:00): instead of waking every minute, the device
//     sleeps through until 06:00. The clock display is stale but invisible —
//     nobody is looking at it. WiFi radio is only raised once per night.
//
//  2. PC poll back-off: after pcConsecutiveFailLimit consecutive failures the
//     PC macro endpoint is assumed unreachable (PC asleep) and is skipped on
//     subsequent background wakes. It is re-tried the next time the user opens
//     the launcher view (Touch() / Execute() call RefreshStatus directly).
//
//  3. HA poll on morning wake: the single quiet-hour exit wake still fetches
//     fresh HA (calendar, lights) state so the user sees correct data at 06:00.
func runSuspendCycle(d *Dashboard) {
	savedBrightness := readBrightness()
	var pcFailCount int // consecutive background PC poll failures

	for {
		if idle := timeSinceActivity(); idle < activityGracePeriod {
			log.Printf("suspend: deferring, idle=%v < %v", idle, activityGracePeriod)
			if sleepOrInterrupt(activityGracePeriod - idle) {
				continue
			}
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
		}

		if d.CurrentView() != ViewHome {
			log.Printf("suspend: jumping to ViewHome before sleep")
			d.runOnUIWait(func() {
				d.showView(ViewHome)
			}, 300*time.Millisecond)
		}

		now := time.Now()
		savedBrightness = rememberBrightness(readBrightness(), savedBrightness)

		// ── Quiet-hour path ──────────────────────────────────────────────────
		// During 22:00–06:00 the screen is almost certainly not being looked at.
		// Skip the per-minute wakes and sleep until 06:00 instead. We still do a
		// full HA fetch on that single morning wake to freshen the agenda/lights.
		//
		// When Ultra Saving Mode is ON: WiFi is physically disabled before sleep
		// and all HA/PC polling is skipped. WiFi is re-enabled on the morning wake.
		if isQuietHour(now) {
			wakeTarget := nextQuietWakeTime(now)
			sleepDur := time.Until(wakeTarget)
			if sleepDur <= 0 {
				sleepDur = time.Second
			}
			scheduledWakeAt := time.Now().Add(sleepDur).Round(0)
			log.Printf("suspend: quiet hours — sleeping until %s (in %v)", wakeTarget.Format("15:04:05"), sleepDur)

			// Ultra Saving: physically turn off WiFi before sleep
			ultraSaving := ultraSavingMode.Load() || readBatteryStatus() == "Discharging"
			if ultraSaving {
				log.Printf("suspend: ultra saving / unplugged — disabling WiFi radio")
				wifiOff()
			}

			if err := setWakeAlarm(sleepDur); err != nil {
				log.Printf("suspend: quiet-hour wakealarm failed: %v — staying awake until morning", err)
				if ultraSaving {
					wifiOn()
				}
				if sleepOrInterrupt(sleepDur) {
					continue
				}
			} else {
				writeBrightness(0)
				if err := suspendToRAM(); err != nil {
					writeBrightness(savedBrightness)
					log.Printf("suspend: quiet-hour suspend failed: %v — staying awake", err)
					if ultraSaving {
						wifiOn()
					}
					if sleepOrInterrupt(sleepDur) {
						continue
					}
				} else {
					resumedAt := time.Now().Round(0)
					if isEarlyWakeWall(resumedAt, scheduledWakeAt, earlyWakeMargin) {
						// Power-button wake during quiet hours — give the user a window.
						if ultraSaving {
							wifiOn()
						}
						log.Printf("suspend: quiet-hour early wake, restoring brightness, jumping to ViewHome, and starting button-wake grace %v", buttonWakeGrace)
						writeBrightness(savedBrightness)
						d.UpdateBrightnessValue(savedBrightness)
						// Jump to the rest screen on power button press
						if ok := d.runOnUIWait(func() {
							d.showView(ViewHome)
						}, 500*time.Millisecond); !ok {
							log.Printf("suspend: timed out jumping to ViewHome on quiet-hour early wake")
						}
						buttonWakeDeadline.Store(time.Now().Add(buttonWakeGrace).UnixNano())
						continue
					}
				}
			}

			// Quiet-hour morning wake: restore WiFi (ultra saving), then frontlight
			// and a single full refresh so the user sees correct state at 06:00.
			if ultraSaving {
				log.Printf("suspend: ultra saving — re-enabling WiFi")
				wifiOn()
			}
			writeBrightness(savedBrightness)
			d.UpdateBrightnessValue(savedBrightness)
			suppressBrightnessSync.Store(true)
			if ok := d.RefreshVisibleViewAndWait(time.Now(), uiRefreshTimeout); !ok {
				log.Printf("suspend: timed out waiting for quiet-hour UI refresh")
			}

			// Ultra Saving: skip HA/PC polling and network wait entirely
			if !ultraSaving {
				time.Sleep(wakeGraceMin)
				waitForNetwork(wakeGraceMax - wakeGraceMin)
				if hassClient != nil {
					if err := hassClient.fetchAll(); err != nil {
						log.Printf("hass: quiet-hour fetch: %v", err)
					} else {
						hassClient.setConnStatus("Connected")
					}
				}
				// Reset PC fail counter on morning wake — PC might be on now.
				pcFailCount = 0
				if pcMacroClient != nil {
					if err := pcMacroClient.RefreshStatus(); err != nil {
						log.Printf("pc macro: quiet-hour refresh: %v", err)
						pcFailCount++
					}
				}
			} else {
				log.Printf("suspend: ultra saving — skipping HA/PC polls")
			}
			if ok := d.RefreshVisibleViewAndWait(time.Now(), uiRefreshTimeout); !ok {
				log.Printf("suspend: timed out waiting for quiet-hour final UI refresh")
			}
			log.Printf("suspend: settling display for %v before next sleep", einkRefreshSettle)
			if sleepOrInterrupt(einkRefreshSettle) {
				continue
			}
			suppressBrightnessSync.Store(false)
			continue
		}

		// ── Normal (daytime) path ────────────────────────────────────────────
		redrawAt := nextRedrawBoundary(now)
		if ultraSavingMode.Load() {
			// Ultra Saving: wake every 2 minutes instead of every 1
			redrawAt = nextRedrawBoundary2Min(now)
		}
		wait := time.Until(redrawAt)

		wakeAlarmDelay := wakeAlarmDelayForBoundary(now, redrawAt)
		scheduledWakeAt := time.Now().Add(wakeAlarmDelay).Round(0)
		resumedFromSuspend := false
		earlyWake := false
		log.Printf("suspend: suspending for %v until redraw=%s (wakealarm +%v, scheduled_wake=%s, lead=%v)", wait, redrawAt.Format(time.RFC3339Nano), wakeAlarmDelay, scheduledWakeAt.Format(time.RFC3339Nano), rtcWakeLead)
		if err := setWakeAlarm(wakeAlarmDelay); err != nil {
			log.Printf("suspend: %v — staying awake this cycle", err)
			if sleepOrInterrupt(wait) {
				continue
			}
		} else {
			log.Printf("suspend: dimming frontlight for power-save wake (saved_brightness=%d)", savedBrightness)
			writeBrightness(0)
			if err := suspendToRAM(); err != nil {
				writeBrightness(savedBrightness)
				log.Printf("suspend: %v — staying awake this cycle", err)
				if sleepOrInterrupt(wait) {
					continue
				}
			} else {
				resumedFromSuspend = true

				// If we resumed well before the scheduled wakealarm, this was
				// a manual (power button) wake - give the user a window to
				// operate the device. Use wall-clock times here: on Kindle, Go's
				// monotonic clock appears not to advance during suspend, so
				// time.Since(suspendStart) makes scheduled RTC wakes look early.
				resumedAt := time.Now().Round(0)
				if isEarlyWakeWall(resumedAt, scheduledWakeAt, earlyWakeMargin) {
					earlyWake = true
					log.Printf("suspend: early wake (resumed=%s scheduled_wake=%s margin=%v) - restoring brightness %d, jumping to ViewHome, and starting button-wake grace %v", resumedAt.Format(time.RFC3339Nano), scheduledWakeAt.Format(time.RFC3339Nano), earlyWakeMargin, savedBrightness, buttonWakeGrace)
					writeBrightness(savedBrightness)
					d.UpdateBrightnessValue(savedBrightness)
					// Jump to the rest screen on power button press
					if ok := d.runOnUIWait(func() {
						d.showView(ViewHome)
					}, 500*time.Millisecond); !ok {
						log.Printf("suspend: timed out jumping to ViewHome on early wake")
					}
					buttonWakeDeadline.Store(time.Now().Add(buttonWakeGrace).UnixNano())
				} else {
					log.Printf("suspend: rtc wake - keeping frontlight off")
				}
			}
		}

		// Suppress brightness sync for the whole refresh - clock/redraw and
		// status polling shouldn't flash the frontlight on.
		suppressBrightnessSync.Store(true)

		// RTC wakes happen before the minute boundary. Stay awake, frontlight off,
		// until :00 so the visible redraw lands on the correct minute. Manual
		// early wakes refresh immediately so the user is not staring at stale UI.
		if !earlyWake {
			if untilRedraw := time.Until(redrawAt); untilRedraw > 0 {
				log.Printf("suspend: waiting %v until redraw boundary %s", untilRedraw, redrawAt.Format(time.RFC3339Nano))
				if sleepOrInterrupt(untilRedraw) {
					continue
				}
			}
		}
		if ok := d.RefreshVisibleViewAndWait(time.Now(), uiRefreshTimeout); !ok {
			log.Printf("suspend: timed out waiting for initial UI refresh")
		}

		// Background (RTC) wakes only redraw the clock — no HASS/PC polling.
		// Network work happens only on manual (button) wakes.
		skipNetwork := !earlyWake
		if status := readBatteryStatus(); status == "Discharging" {
			if skipNetwork {
				// Unplugged background wake: also drop the WiFi radio to save battery.
				if wifiState() == "On" {
					log.Printf("suspend: unplugged and background wake - disabling WiFi")
					wifiOff()
				}
			} else {
				// Manual wake while unplugged: bring WiFi back for the poll.
				if wifiState() == "Off" {
					log.Printf("suspend: early wake - re-enabling WiFi")
					wifiOn()
				}
			}
		}

		if resumedFromSuspend && !skipNetwork {
			// Resume happens asynchronously (WiFi firmware reload, driver
			// reinit) - give the device a moment to settle before network work,
			// or it can hang.
			if sleepOrInterrupt(wakeGraceMin) {
				continue
			}
			log.Printf("suspend: resumed, waiting up to %v for network", wakeGraceMax-wakeGraceMin)
			if !waitForNetwork(wakeGraceMax - wakeGraceMin) {
				continue
			}
		}

		if !skipNetwork {
			if hassClient != nil {
				hassClient.setConnStatus("Fetching...")
				err := hassClient.FetchAllWithRetry(3, 1*time.Second)
				if err != nil {
					log.Printf("hass: post-resume fetch totally failed: %v", err)
					hassClient.setConnStatus("Error")
				} else {
					hassClient.setConnStatus("Connected")
				}
			}

			// PC poll back-off: if the PC has been unreachable for several
			// consecutive background wakes, skip polling until the user explicitly
			// opens the launcher (which calls RefreshStatus directly via Touch/
			// Execute). This avoids burning a WiFi-on cycle for a sleeping PC.
			if pcMacroClient != nil {
				if pcFailCount >= pcConsecutiveFailLimit {
					log.Printf("suspend: skipping PC poll (fail_count=%d >= %d)", pcFailCount, pcConsecutiveFailLimit)
				} else {
					if err := pcMacroClient.RefreshStatus(); err != nil {
						pcFailCount++
						log.Printf("pc macro: post-resume refresh: %v (fail_count=%d)", err, pcFailCount)
						if pcFailCount >= pcConsecutiveFailLimit {
							log.Printf("suspend: PC unreachable for %d consecutive wakes — suppressing background polls", pcFailCount)
							d.SetPCConnectionStatus("Unreachable")
						}
					} else {
						if pcFailCount > 0 {
							log.Printf("suspend: PC reconnected after %d failures", pcFailCount)
						}
						pcFailCount = 0
					}
				}
			}
		}

		// HA/PC refreshes enqueue GTK work. Drain one final refresh and let the
		// e-ink controller settle before the next suspend, otherwise the Kindle can
		// sleep while a track/title/agenda update is visibly mid-refresh.
		if ok := d.RefreshVisibleViewAndWait(time.Now(), uiRefreshTimeout); !ok {
			log.Printf("suspend: timed out waiting for final UI refresh")
		}
		log.Printf("suspend: settling display for %v before next sleep", einkRefreshSettle)
		if sleepOrInterrupt(einkRefreshSettle) {
			continue
		}

		suppressBrightnessSync.Store(false)
	}
}
