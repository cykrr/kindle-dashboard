package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func setPowerMode(mode string) error {
	return runPowerShell(false, `& "C:\Users\krr\bin\toggle-mode.ps1" -mode `+mode)
}

func togglePowerMode() error {
	mode := getActivePowerScheme()
	if mode == "power" {
		return setPowerMode("save")
	}
	return setPowerMode("power")
}

// executeAction runs the named action and returns an error if it fails.
func executeAction(action string) error {
	switch action {
	case "mute_mic":
		return runPowerShell(false, `$sig='[DllImport("user32.dll")]public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);'; $k=Add-Type -MemberDefinition $sig -Name Win32KeyEvent -Namespace Win32Functions -PassThru; $k::keybd_event(0x08,0,0,[UIntPtr]::Zero); $k::keybd_event(0x0D,0,0,[UIntPtr]::Zero); Start-Sleep -Milliseconds 50; $k::keybd_event(0x08,0,2,[UIntPtr]::Zero); $k::keybd_event(0x0D,0,2,[UIntPtr]::Zero)`)
	case "play_pause":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]179)`)
	case "prev_track":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]177)`)
	case "next_track":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]176)`)
	case "sleep":
		return runPowerShell(true, `Add-Type -Assembly System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend', $false, $false)`)
	case "gaming_mode":
		return togglePowerMode()
	case "power_mode":
		return setPowerMode("power")
	case "save_mode":
		return setPowerMode("save")
	case "screenshot":
		return runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]44)`)
	case "launch_chrome":
		return runPowerShell(false, `
$exe = 'chrome.exe'
$name = 'chrome'
$hwnd = (Get-Process $name -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 }).MainWindowHandle
if ($hwnd) {
    $sig = @'
[DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
[DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
[DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
'@
    $win32 = Add-Type -MemberDefinition $sig -Name Win32Toggle -Namespace Win32Functions -PassThru
    if ($win32::IsIconic($hwnd)) {
        $win32::ShowWindowAsync($hwnd, 9)
        $win32::SetForegroundWindow($hwnd)
    } else {
        $win32::ShowWindowAsync($hwnd, 6)
    }
} else {
    Start-Process $exe
}
`)
	case "launch_mail":
		return runPowerShell(false, `
$exe = 'C:\Users\krr\scoop\apps\thunderbird\current\thunderbird.exe'
$name = 'thunderbird'
$hwnd = (Get-Process $name -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 }).MainWindowHandle
if ($hwnd) {
    $sig = @'
[DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
[DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
[DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
'@
    $win32 = Add-Type -MemberDefinition $sig -Name Win32Toggle -Namespace Win32Functions -PassThru
    if ($win32::IsIconic($hwnd)) {
        $win32::ShowWindowAsync($hwnd, 9)
        $win32::SetForegroundWindow($hwnd)
    } else {
        $win32::ShowWindowAsync($hwnd, 6)
    }
} else {
    Start-Process $exe
}
`)
	case "launch_fortnite":
		// Invoke-Item opens with the default handler (Epic Games Launcher / browser)
		return runPowerShell(false, `Invoke-Item "C:\Users\krr\Desktop\Fortnite.url"`)
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

	err := executeAction(action)
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

	// Wait for Windows to process the action before reading status
	time.Sleep(400 * time.Millisecond)
	broker.Publish(buildStatus())
	w.WriteHeader(http.StatusOK)
}
