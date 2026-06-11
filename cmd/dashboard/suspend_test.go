package main

import (
	"testing"
	"time"
)

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
