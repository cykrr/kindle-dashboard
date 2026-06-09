package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
)

// runHiddenCmd runs any command with its console window completely hidden.
func runHiddenCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.Output()
}

// runPowerShell executes a PowerShell script, optionally elevated via gsudo.
// It handles CLIXML error output (UTF-16 LE) that PowerShell emits on failures.
func runPowerShell(privileged bool, script string) error {
	encoded := base64.StdEncoding.EncodeToString(encodeUTF16LE(script))
	psArgs := []string{
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass",
		"-EncodedCommand", encoded,
	}

	var cmd *exec.Cmd
	if privileged {
		allArgs := append([]string{"powershell.exe"}, psArgs...)
		cmd = exec.Command(cfg.Gsudo, allArgs...)
	} else {
		cmd = exec.Command("powershell.exe", psArgs...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, cleanupPSOutput(output))
	}
	return nil
}

// decodeCLIXML attempts to decode PowerShell CLIXML output (UTF-16 LE) and
// extract human-readable error messages. Returns empty string if parsing fails.
func decodeCLIXML(raw []byte) string {
	// PowerShell CLIXML starts with "#< CLIXML"
	marker := []byte("#< CLIXML")
	idx := indexOf(raw, marker)
	if idx < 0 {
		return ""
	}
	body := raw[idx+len(marker):]

	// Try to decode as UTF-16 LE
	if len(body) >= 2 {
		u16 := make([]uint16, 0, len(body)/2)
		for i := 0; i+1 < len(body); i += 2 {
			u16 = append(u16, uint16(body[i])|uint16(body[i+1])<<8)
		}
		decoded := string(utf16.Decode(u16))

		// Extract <S S="Error">text</S>
		var errors []string
		for {
			start := strings.Index(decoded, `<S S="Error">`)
			if start < 0 {
				break
			}
			start += len(`<S S="Error">`)
			end := strings.Index(decoded[start:], `</S>`)
			if end < 0 {
				break
			}
			errors = append(errors, decoded[start:start+end])
			decoded = decoded[start+end:]
		}
		if len(errors) > 0 {
			return strings.Join(errors, "; ")
		}
	}
	return ""
}

func indexOf(b, sub []byte) int {
	for i := 0; i <= len(b)-len(sub); i++ {
		if string(b[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}

// cleanupPSOutput sanitizes PowerShell output for logging.
// Detects CLIXML and falls back to raw string trimming.
func cleanupPSOutput(raw []byte) string {
	if msg := decodeCLIXML(raw); msg != "" {
		return msg
	}
	return strings.TrimSpace(string(raw))
}

// encodeUTF16LE encodes a string as UTF-16 LE bytes.
func encodeUTF16LE(s string) []byte {
	runes := []rune(s)
	u16 := utf16.Encode(runes)
	b := make([]byte, len(u16)*2)
	for i, r := range u16 {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return b
}
