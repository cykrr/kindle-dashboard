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
