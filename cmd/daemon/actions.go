package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// executeAction runs the named action and returns an error if it fails.
func executeAction(action string) error {
	// Fallback to catalog launch if the action matches a shortcut ID
	safeID := makeID(action)
	if pathForID(safeID) != "" {
		return executeLaunch(safeID)
	}
	return unknownActionError(action)
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

// notifyWindows sends a Windows balloon-tip notification.
// Uses PowerShell's NotifyIcon so it works from any context (hidden, GUI, etc.).
func notifyWindows(title, message, iconType string) {
	notifyWindowsClickCopy(title, message, iconType, "")
}

// notifyWindowsClickCopy sends a Windows balloon-tip notification. If clipText
// is non-empty, clicking the balloon copies clipText to the clipboard.
func notifyWindowsClickCopy(title, message, iconType, clipText string) {
	switch iconType {
	case "error":
		iconType = "Error"
	case "warning":
		iconType = "Warning"
	default:
		iconType = "Info"
	}
	if clipText == "" {
		_ = runPowerShell(false, fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.Icon]::ExtractAssociatedIcon('C:\Program Files\KindleDashboard\macro-daemon.exe')
$n.BalloonTipTitle = '%s'
$n.BalloonTipText = '%s'
$n.BalloonTipIcon = '%s'
$n.Visible = $true
$n.ShowBalloonTip(5000)
Start-Sleep 6
$n.Dispose()
`, psEscape(title), psEscape(message), iconType))
		return
	}
	_ = runPowerShell(false, fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.Icon]::ExtractAssociatedIcon('C:\Program Files\KindleDashboard\macro-daemon.exe')
$n.BalloonTipTitle = '%s'
$n.BalloonTipText = '%s'
$n.BalloonTipIcon = '%s'
$n.Visible = $true
$clip = '%s'
$n.add_BalloonTipClicked({ Set-Clipboard -Value $clip }.GetNewClosure())
$n.ShowBalloonTip(5000)
$deadline = (Get-Date).AddSeconds(6)
while ((Get-Date) -lt $deadline) {
    [System.Windows.Forms.Application]::DoEvents()
    Start-Sleep -Milliseconds 100
}
$n.Dispose()
`, psEscape(title), psEscape(message), iconType, psEscape(clipText)))
}

// psEscape escapes a string for safe embedding in a PowerShell single-quoted string.
// Single quotes are doubled; dollar signs and backticks are literal inside single quotes.
func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
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

	target := r.URL.Query().Get("target")
	var err error
	// fast actions don't change the polled StatusResponse and must respond with
	// low latency (e.g. a real-time volume slider), so they skip the settle
	// sleep + status broadcast below.
	fast := false
	switch action {
	case "launch": // generic launcher: target=<catalog id>
		err = executeLaunch(target)
		fast = true
	case "volume_set": // target=0..100
		err = audioSetVolume(target)
		fast = true
	case "mute_toggle":
		err = audioToggleMute()
		fast = true
	case "audio_output": // target=<device id>
		err = audioSetDefault(target)
		fast = true
	default:
		err = executeAction(action)
	}
	if err != nil {
		if isUnknownAction(err) {
			notifyWindows("Kindle Macro", "Unknown action: "+action, "warning")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Action %s failed: %v", action, err)
		notifyWindowsClickCopy("Kindle Macro: Action Failed", action+": "+err.Error(), "error", action+": "+err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !fast {
		// Wait for Windows to process the action before reading status
		time.Sleep(400 * time.Millisecond)
		broker.Publish(buildStatus())
	}
	w.WriteHeader(http.StatusOK)
}
