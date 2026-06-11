package main

import (
	"log"
	"os/exec"
)

// stopKindleFramework kills the Kindle's lab126 launcher/UI so our window
// has the screen to itself.
func stopKindleFramework() {
	if err := exec.Command("sh", "-c", "stop lab126_gui 2>/dev/null || /etc/init.d/framework stop 2>/dev/null || true").Run(); err != nil {
		log.Printf("stopKindleFramework: %v", err)
	}
}

// RestoreKindleFramework restarts the Kindle's lab126 launcher/UI. Call on exit.
func RestoreKindleFramework() {
	if err := exec.Command("sh", "-c", "start lab126_gui 2>/dev/null || /etc/init.d/framework start 2>/dev/null || true").Run(); err != nil {
		log.Printf("RestoreKindleFramework: %v", err)
	}
}
