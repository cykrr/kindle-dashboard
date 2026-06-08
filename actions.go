package main

import (
	"log"
	"net/http"
	"time"
)

// executeAction runs the named action and returns an error if it fails.
func executeAction(action string) error {
	switch action {
	case "mute_mic":
		return runPowerShell(false, `Set-AudioDevice -Index 8 -ToggleMute`)
	case "play_pause":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]179)`)
	case "prev_track":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]177)`)
	case "next_track":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]176)`)
	case "sleep":
		return runPowerShell(true, `Add-Type -Assembly System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend', $false, $false)`)
	case "gaming_mode":
		return runPowerShell(false, `& "C:\Users\krr\bin\toggle-mode.ps1" -mode power`)
	case "screenshot":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]44)`)
	case "launch_chrome":
		return runPowerShell(false, `Start-Process "chrome.exe"`)
	case "launch_mail":
		return runPowerShell(false, `Start-Process "mailto:"`)
	case "monitor_toggle":
		return runPowerShell(false, `& "C:\Users\krr\bin\toggle-monitor.ps1"`)
	case "restart":
		return runPowerShell(true, `Restart-Computer -Force`)
	case "shutdown":
		return runPowerShell(true, `Stop-Computer -Force`)
	default:
		return unknownActionError(action)
	}
}

type actionError struct {
	msg string
}

func (e *actionError) Error() string { return e.msg }

func unknownActionError(a string) error {
	return &actionError{msg: "unknown action: " + a}
}

func isUnknownAction(err error) bool {
	_, ok := err.(*actionError)
	return ok
}

// handleExecute handles POST /execute requests.
func handleExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.URL.Query().Get("key") != cfg.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	action := r.URL.Query().Get("action")
	log.Printf("Executing action: %s", action)

	err := executeAction(action)
	if err != nil {
		if isUnknownAction(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Action %s failed: %v", action, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Wait for Windows to process the action before reading status
	time.Sleep(400 * time.Millisecond)
	broker.Publish(buildStatus())
	w.WriteHeader(http.StatusOK)
}
