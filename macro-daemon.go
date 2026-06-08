package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/media/control"
)

const (
	API_KEY = "your-super-secret-key" // CHANGE THIS
	PORT    = ":8080"
	GSUDO   = `C:\Users\krr\scoop\apps\gsudo\current\gsudo.exe`
)

type StatusResponse struct {
	GamingMode string `json:"gaming_mode"`
	MonitorOn  bool   `json:"monitor_on"`
	Track      string `json:"track"`
	Artist     string `json:"artist"`
	Status     string `json:"status"`
}

type SSEBroker struct {
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
	mu         sync.RWMutex
	lastStatus []byte
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 16),
	}
}

func (b *SSEBroker) Run(ctx context.Context) {
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			if b.lastStatus != nil {
				select {
				case client <- b.lastStatus:
				default:
				}
			}
			b.mu.Unlock()

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
			}
			b.mu.Unlock()

		case msg := <-b.broadcast:
			b.mu.RLock()
			b.lastStatus = msg
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					close(client)
					delete(b.clients, client)
				}
			}
			b.mu.RUnlock()

		case <-ctx.Done():
			return
		}
	}
}

func (b *SSEBroker) Publish(status []byte) {
	select {
	case b.broadcast <- status:
	default:
	}
}

var broker *SSEBroker

// runHiddenCmd runs any command with its console window completely hidden.
func runHiddenCmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.Output()
}

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
		cmd = exec.Command(GSUDO, allArgs...)
	} else {
		cmd = exec.Command("powershell.exe", psArgs...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

func encodeUTF16LE(s string) []byte {
	runes := []rune(s)
	u16 := make([]uint16, len(runes))
	for i, r := range runes {
		u16[i] = uint16(r)
	}
	b := make([]byte, len(u16)*2)
	for i, r := range u16 {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return b
}

func getActivePowerScheme() string {
	out, err := runHiddenCmd("powercfg", "/getactivescheme")
	if err != nil {
		return "unknown"
	}
	s := string(out)
	if strings.Contains(s, "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c") {
		return "power"
	}
	return "normal"
}

// awaitAsyncOp polls an IAsyncOperation until it completes, then returns the raw result pointer.
func awaitAsyncOp(asyncOp *foundation.IAsyncOperation) (unsafe.Pointer, error) {
	// QI for IAsyncInfo to poll completion status
	var asyncInfo *foundation.IAsyncInfo
	iidAsyncInfo := ole.NewGUID(foundation.GUIDIAsyncInfo)
	iunk := (*ole.IUnknown)(unsafe.Pointer(asyncOp))
	if err := iunk.PutQueryInterface(iidAsyncInfo, &asyncInfo); err != nil {
		return nil, fmt.Errorf("QI for IAsyncInfo: %w", err)
	}
	defer asyncInfo.Release()

	for {
		status, err := asyncInfo.GetStatus()
		if err != nil {
			return nil, fmt.Errorf("get status: %w", err)
		}
		switch status {
		case foundation.AsyncStatusCompleted:
			ptr, err := asyncOp.GetResults()
			if err != nil {
				return nil, fmt.Errorf("get results: %w", err)
			}
			return ptr, nil
		case foundation.AsyncStatusError:
			errCode, _ := asyncInfo.GetErrorCode()
			return nil, fmt.Errorf("async error: 0x%08X", errCode)
		case foundation.AsyncStatusCanceled:
			return nil, fmt.Errorf("async canceled")
		case foundation.AsyncStatusStarted:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func getMediaInfo() (string, string, string) {
	// Pin the goroutine to this OS thread for COM lifetime
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Initialize COM (MTA = 0 so Go goroutines can migrate between threads safely)
	if err := ole.CoInitializeEx(0, 0); err != nil {
		log.Printf("[media] CoInitializeEx failed: %v", err)
		return "Idle", "", ""
	}
	defer ole.CoUninitialize()

	asyncOp, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
	if err != nil {
		log.Printf("[media] RequestAsync failed: %v", err)
		return "Idle", "", ""
	}

	rawPtr, err := awaitAsyncOp(asyncOp)
	asyncOp.Release()
	if err != nil {
		log.Printf("[media] await manager failed: %v", err)
		return "Idle", "", ""
	}
	if rawPtr == nil {
		log.Printf("[media] manager rawPtr is nil")
		return "Idle", "", ""
	}

	// QI for the manager interface
	var managerIface *control.GlobalSystemMediaTransportControlsSessionManager
	iidManager := ole.NewGUID(control.GUIDiGlobalSystemMediaTransportControlsSessionManager)
	iunk := (*ole.IUnknown)(rawPtr)
	if err := iunk.PutQueryInterface(iidManager, &managerIface); err != nil {
		log.Printf("[media] QI for manager failed: %v", err)
		iunk.Release()
		return "Idle", "", ""
	}
	iunk.Release()
	defer managerIface.Release()

	log.Printf("[media] got manager, querying current session")

	session, err := managerIface.GetCurrentSession()
	if err != nil {
		log.Printf("[media] GetCurrentSession error: %v", err)
		return "Idle", "", ""
	}
	if session == nil {
		log.Printf("[media] no current session (no media playing)")
		return "Idle", "", ""
	}
	defer session.Release()

	log.Printf("[media] got session, fetching media properties")

	propsAsync, err := session.TryGetMediaPropertiesAsync()
	if err != nil {
		log.Printf("[media] TryGetMediaPropertiesAsync error: %v", err)
		return "Playing", "", ""
	}

	propsRaw, err := awaitAsyncOp(propsAsync)
	propsAsync.Release()
	if err != nil {
		log.Printf("[media] await props failed: %v", err)
		return "Playing", "", ""
	}
	if propsRaw == nil {
		log.Printf("[media] props rawPtr is nil")
		return "Playing", "", ""
	}

	var props *control.GlobalSystemMediaTransportControlsSessionMediaProperties
	iidProps := ole.NewGUID(control.GUIDiGlobalSystemMediaTransportControlsSessionMediaProperties)
	propsIunk := (*ole.IUnknown)(propsRaw)
	if err := propsIunk.PutQueryInterface(iidProps, &props); err != nil {
		log.Printf("[media] QI for props failed: %v", err)
		propsIunk.Release()
		return "Playing", "", ""
	}
	propsIunk.Release()
	defer props.Release()

	title, _ := props.GetTitle()
	artist, _ := props.GetArtist()
	log.Printf("[media] title=%q artist=%q", title, artist)

	playbackInfo, err := session.GetPlaybackInfo()
	if err != nil {
		log.Printf("[media] GetPlaybackInfo error: %v", err)
		status := "Playing"
		if title == "" {
			status = "Idle"
		}
		return status, title, artist
	}
	defer playbackInfo.Release()

	pstatus, err := playbackInfo.GetPlaybackStatus()
	if err != nil {
		log.Printf("[media] GetPlaybackStatus error: %v", err)
		status := "Playing"
		if title == "" {
			status = "Idle"
		}
		return status, title, artist
	}

	log.Printf("[media] playback status=%v", pstatus)

	status := "Idle"
	switch pstatus {
	case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying:
		status = "Playing"
	case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPaused:
		status = "Idle"
	case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusStopped:
		status = "Idle"
	}

	return status, title, artist
}

func buildStatus() []byte {
	status, track, artist := getMediaInfo()
	resp := StatusResponse{
		GamingMode: getActivePowerScheme(),
		MonitorOn:  true,
		Status:     status,
		Track:      track,
		Artist:     artist,
	}
	data, _ := json.Marshal(resp)
	return data
}

func startStatusPoller(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	broker.Publish(buildStatus())

	for {
		select {
		case <-ticker.C:
			broker.Publish(buildStatus())
		case <-ctx.Done():
			return
		}
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if r.URL.Query().Get("key") != API_KEY {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan []byte, 8)
	broker.register <- messageChan
	defer func() { broker.unregister <- messageChan }()

	ctx := r.Context()
	for {
		select {
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", string(msg))
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Query().Get("key") != API_KEY {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Write(buildStatus())
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.URL.Query().Get("key") != API_KEY {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	action := r.URL.Query().Get("action")
	log.Printf("Executing action: %s", action)
	var err error

	switch action {
	case "mute_mic":
		err = runPowerShell(false, `Set-AudioDevice -Index 8 -ToggleMute`)
	case "play_pause":
		err = runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]179)`)
	case "prev_track":
		err = runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]177)`)
	case "next_track":
		err = runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]176)`)
	case "sleep":
		err = runPowerShell(true, `Add-Type -Assembly System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend', $false, $false)`)
	case "gaming_mode":
		err = runPowerShell(false, `& "C:\Users\krr\bin\toggle-mode.ps1" -mode power`)
	case "screenshot":
		err = runPowerShell(false, `(New-Object -ComObject WScript.Shell).SendKeys([char]44)`)
	case "launch_chrome":
		err = runPowerShell(false, `Start-Process "chrome.exe"`)
	case "launch_mail":
		err = runPowerShell(false, `Start-Process "mailto:"`)
	case "monitor_toggle":
		err = runPowerShell(false, `& "C:\Users\krr\bin\toggle-monitor.ps1"`)
	case "restart":
		err = runPowerShell(true, `Restart-Computer -Force`)
	case "shutdown":
		err = runPowerShell(true, `Stop-Computer -Force`)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Action %s failed: %v", action, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Wait for Windows to process the action before reading status
	time.Sleep(400 * time.Millisecond)
	broker.Publish(buildStatus())
	w.WriteHeader(http.StatusOK)
}

func main() {
	logFile, err := os.OpenFile(`C:\KindleDashboard\macro-daemon.log`, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(logFile)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker = NewSSEBroker()
	go broker.Run(ctx)

	go startStatusPoller(ctx)

	http.HandleFunc("/execute", handleExecute)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/events", handleSSE)

	fmt.Printf("Windows Macro Service started on %s\n", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}
