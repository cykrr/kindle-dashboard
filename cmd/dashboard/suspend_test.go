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
