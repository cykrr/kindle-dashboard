package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"syscall"
	"os"
	"os/exec"

	"github.com/getlantern/systray"
)

// generateIcon returns a 16×16 RGBA PNG icon for the system tray.
// A high-contrast Kindle silhouette: warm e-ink screen with a charcoal
// bezel and a bright accent, visible on both light and dark taskbars.
func generateIcon() ([]byte, error) {
	const size = 16
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Over)

	// Colours — tuned for contrast on light and dark backgrounds
	black := color.RGBA{0x00, 0x00, 0x00, 0xff}
	eink  := color.RGBA{0xf5, 0xf0, 0xe8, 0xff} // warm page white
	accent := color.RGBA{0x2e, 0x9a, 0xd4, 0xff} // blue accent (Kindle logo area)

	// ── Device body (kindle silhouette) ──
	// 14w × 14h rounded rect, 1px margin
	for y := 1; y <= 14; y++ {
		for x := 1; x <= 14; x++ {
			// Rounded corners: skip corner pixels
			if (x == 1 || x == 14) && (y == 1 || y == 14) {
				continue
			}
			if (x == 2 || x == 13) && (y <= 1 || y >= 14) {
				continue
			}
			if (y == 2 || y == 13) && (x <= 1 || x >= 14) {
				continue
			}
			img.Set(x, y, black)
		}
	}

	// ── Screen (e-ink coloured inset) ──
	for y := 3; y <= 11; y++ {
		for x := 3; x <= 12; x++ {
			img.Set(x, y, eink)
		}
	}

	// ── Text lines (ink grey) ──
	ink := color.RGBA{0x88, 0x80, 0x72, 0xff}
	for _, row := range []int{5, 7, 9} {
		for x := 4; x <= 10; x++ {
			img.Set(x, row, ink)
		}
	}
	// Short line at row 10 (just a partial line)
	for x := 4; x <= 7; x++ {
		img.Set(x, 10, ink)
	}

	// ── Accent bar (Kindle logo zone on the chin) ──
	for x := 6; x <= 9; x++ {
		img.Set(x, 13, accent)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func startTray(stopCh chan<- struct{}) {
	systray.Run(func() {
		onTrayReady()
	}, func() {
		// onExit: signal the daemon to shut down
		close(stopCh)
	})
}

func onTrayReady() {
	icon, err := generateIcon()
	if err == nil {
		systray.SetIcon(icon)
	} else {
		log.Printf("warning: failed to generate tray icon: %v", err)
	}
	systray.SetTooltip("Kindle Macro Daemon")

	// ── Menu items ──
	mShowLog := systray.AddMenuItem("Show Log", "Tail the daemon log file")
	mOpenFolder := systray.AddMenuItem("Open Log Folder", "Open folder containing the log")
	systray.AddSeparator()
	mOpenConfig := systray.AddMenuItem("Edit Config (.env)", "Open .env file for editing")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Stop the daemon")

	// Menu event loop
	go func() {
		for {
			select {
			case <-mShowLog.ClickedCh:
				openLogTail()
			case <-mOpenFolder.ClickedCh:
				openLogFolder()
			case <-mOpenConfig.ClickedCh:
				openConfigFile()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// logPath returns the log file path from the config, with a sensible fallback.
func logPath() string {
	p := cfg.LogPath
	if p == "" {
		p = `C:\ProgramData\KindleDashboard\macro-daemon.log`
	}
	return p
}

// envPath returns the .env file path — first next to the binary, then ProgramData.
func envPath() string {
	// Try next to binary first
	exe, err := os.Executable()
	if err == nil {
		env := exe[:len(exe)-len(".exe")] + ".env"
		if _, err := os.Stat(env); err == nil {
			return env
		}
	}
	// Try ProgramData
	dataEnv := `C:\ProgramData\KindleDashboard\.env`
	if _, err := os.Stat(dataEnv); err == nil {
		return dataEnv
	}
	// Fallback to ProgramData anyway (it will be created)
	return dataEnv
}

// openLogTail launches a new console window tailing the daemon log.
func openLogTail() {
	lp := logPath()
	// PowerShell script that waits for the log file, then tails it.
	// Using -ErrorAction Ignore so it doesn't crash if the file is missing at first.
	script := fmt.Sprintf(`
$log = '%s'
Write-Host "Tailing: $log" -ForegroundColor Cyan
Write-Host "(close this window to stop)" -ForegroundColor Gray
Write-Host ""
# Wait for the file to appear
while (-not (Test-Path $log)) {
    Start-Sleep 1
}
Get-Content $log -Wait -Tail 50
`, lp)

	// Encode as base64 UTF-16 LE for clean quoting
	encoded := base64.StdEncoding.EncodeToString(encodeUTF16LE(script))
	cmd := exec.Command("powershell.exe",
		"-NoExit", "-ExecutionPolicy", "Bypass",
		"-EncodedCommand", encoded,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000010, // CREATE_NEW_CONSOLE — new window, detached from daemon
	}
	if err := cmd.Start(); err != nil {
		log.Printf("failed to open log tail: %v", err)
	}
}

// openLogFolder opens Explorer at the log file's directory.
func openLogFolder() {
	cmd := exec.Command("explorer.exe", "/select,", logPath())
	if err := cmd.Start(); err != nil {
		log.Printf("failed to open log folder: %v", err)
	}
}

// openConfigFile opens the .env file in the default editor.
func openConfigFile() {
	cmd := exec.Command("cmd.exe", "/c", "start", "", envPath())
	if err := cmd.Start(); err != nil {
		log.Printf("failed to open config: %v", err)
	}
}
