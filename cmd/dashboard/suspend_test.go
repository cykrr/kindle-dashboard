package main

import (
	"testing"
	"time"
)

func TestWakeAlarmSecondsCeils(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want int
	}{
		{0, 1},
		{time.Nanosecond, 1},
		{999*time.Millisecond + 999*time.Microsecond, 1},
		{time.Second, 1},
		{time.Second + time.Nanosecond, 2},
		{30*time.Second + 429*time.Millisecond, 31},
	}
	for _, tc := range cases {
		if got := wakeAlarmSeconds(tc.d); got != tc.want {
			t.Fatalf("wakeAlarmSeconds(%v) = %d; want %d", tc.d, got, tc.want)
		}
	}
}

func TestWakeAlarmDelayForBoundaryUsesLead(t *testing.T) {
	now := time.Date(2026, 6, 11, 11, 18, 40, 135947004, time.UTC)
	boundary := time.Date(2026, 6, 11, 11, 19, 0, 0, time.UTC)
	got := wakeAlarmDelayForBoundary(now, boundary)
	if got != 18*time.Second {
		t.Fatalf("wakeAlarmDelayForBoundary = %v; want 18s", got)
	}
}

func TestNextRedrawBoundarySkipsWhenTooClose(t *testing.T) {
	now := time.Date(2026, 6, 11, 11, 18, 57, 0, time.UTC)
	got := nextRedrawBoundary(now)
	want := time.Date(2026, 6, 11, 11, 20, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("nextRedrawBoundary = %s; want %s", got, want)
	}
}

func TestRememberBrightness(t *testing.T) {
	if got := rememberBrightness(120, 50); got != 120 {
		t.Fatalf("rememberBrightness should prefer current nonzero brightness, got %d", got)
	}
	if got := rememberBrightness(0, 50); got != 50 {
		t.Fatalf("rememberBrightness should preserve saved brightness while dimmed, got %d", got)
	}
}

func TestIsEarlyWakeWall(t *testing.T) {
	scheduled := time.Date(2026, 6, 11, 11, 9, 2, 0, time.UTC)

	if isEarlyWakeWall(scheduled, scheduled, earlyWakeMargin) {
		t.Fatal("wake at scheduled wall-clock time was classified as early")
	}
	if isEarlyWakeWall(scheduled.Add(-earlyWakeMargin/2), scheduled, earlyWakeMargin) {
		t.Fatal("wake inside early-wake margin was classified as early")
	}
	if !isEarlyWakeWall(scheduled.Add(-earlyWakeMargin-time.Second), scheduled, earlyWakeMargin) {
		t.Fatal("wake before early-wake margin was not classified as early")
	}
}

// ── Quiet-hour tests ────────────────────────────────────────────────────────

func TestIsQuietHour(t *testing.T) {
	loc := time.UTC
	cases := []struct {
		h    int
		want bool
	}{
		{0, true},   // midnight
		{3, true},   // 3 AM
		{5, true},   // 5 AM (last quiet hour)
		{6, false},  // 6 AM — daytime starts
		{12, false}, // noon
		{21, false}, // 9 PM — not yet quiet
		{22, true},  // 10 PM — quiet starts
		{23, true},  // 11 PM
	}
	for _, tc := range cases {
		now := time.Date(2026, 6, 15, tc.h, 30, 0, 0, loc)
		if got := isQuietHour(now); got != tc.want {
			t.Errorf("isQuietHour(hour=%d) = %v; want %v", tc.h, got, tc.want)
		}
	}
}

func TestNextQuietWakeTime_BeforeMidnight(t *testing.T) {
	loc := time.UTC
	// 11 PM on the 14th → should wake on 6 AM the 15th (minus lead)
	now := time.Date(2026, 6, 14, 23, 0, 0, 0, loc)
	got := nextQuietWakeTime(now)
	want := time.Date(2026, 6, 15, quietHourEnd, 0, 0, 0, loc).Add(-rtcWakeLead)
	if !got.Equal(want) {
		t.Fatalf("nextQuietWakeTime(23:00 on 14th) = %s; want %s", got, want)
	}
}

func TestNextQuietWakeTime_AfterMidnight(t *testing.T) {
	loc := time.UTC
	// 2 AM on the 15th → same day's 6 AM
	now := time.Date(2026, 6, 15, 2, 0, 0, 0, loc)
	got := nextQuietWakeTime(now)
	want := time.Date(2026, 6, 15, quietHourEnd, 0, 0, 0, loc).Add(-rtcWakeLead)
	if !got.Equal(want) {
		t.Fatalf("nextQuietWakeTime(02:00 on 15th) = %s; want %s", got, want)
	}
}

func TestNextQuietWakeTime_AfterMorning(t *testing.T) {
	loc := time.UTC
	// 10 PM on the 15th (post-quietHourEnd but in quiet hours) → 6 AM the 16th
	now := time.Date(2026, 6, 15, 22, 0, 0, 0, loc)
	got := nextQuietWakeTime(now)
	want := time.Date(2026, 6, 16, quietHourEnd, 0, 0, 0, loc).Add(-rtcWakeLead)
	if !got.Equal(want) {
		t.Fatalf("nextQuietWakeTime(22:00 on 15th) = %s; want %s", got, want)
	}
}

func TestNextQuietWakeTime_IsInFuture(t *testing.T) {
	// For any quiet-hour time, the returned wake should always be in the future.
	loc := time.UTC
	quietTimes := []int{22, 23, 0, 1, 2, 3, 4, 5}
	for _, h := range quietTimes {
		now := time.Date(2026, 6, 15, h, 45, 0, 0, loc)
		wake := nextQuietWakeTime(now)
		if !wake.After(now) {
			t.Errorf("nextQuietWakeTime(hour=%d) = %s is not after now=%s", h, wake, now)
		}
	}
}

func TestNextQuietWakeTime_LeadApplied(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 6, 15, 3, 0, 0, 0, loc)
	got := nextQuietWakeTime(now)
	// Should be exactly quietHourEnd:00:00 minus rtcWakeLead
	exactBoundary := time.Date(2026, 6, 15, quietHourEnd, 0, 0, 0, loc)
	wantLead := exactBoundary.Sub(got)
	if wantLead != rtcWakeLead {
		t.Fatalf("rtcWakeLead not applied: gap between wake and boundary = %v; want %v", wantLead, rtcWakeLead)
	}
}
