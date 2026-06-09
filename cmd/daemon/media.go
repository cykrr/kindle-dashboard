package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/media/control"
)

// StatusResponse is the JSON payload sent to SSE clients.
type StatusResponse struct {
	GamingMode string `json:"gaming_mode"`
	MonitorOn  bool   `json:"monitor_on"`
	Track      string `json:"track"`
	Artist     string `json:"artist"`
	Status     string `json:"status"`
}

// getActivePowerScheme returns "power", "save", or "normal" based on the active Windows power plan.
func getActivePowerScheme() string {
	out, err := runHiddenCmd("powercfg", "/getactivescheme")
	if err != nil {
		return "unknown"
	}
	s := strings.ToLower(string(out))
	if strings.Contains(s, "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c") {
		return "power"
	}
	if strings.Contains(s, "a1841308-3541-4fab-bc81-f71556f20b4a") {
		return "save"
	}
	return "normal"
}

// awaitAsyncOp polls an IAsyncOperation until it completes, then returns the raw result pointer.
func awaitAsyncOp(asyncOp *foundation.IAsyncOperation) (unsafe.Pointer, error) {
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

// getMediaInfo queries the Windows SMTC API for the currently playing media.
// Returns (status, track, artist).
func getMediaInfo() (string, string, string) {
	// Pin the goroutine to this OS thread for COM lifetime
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Initialize COM (MTA = 0 so goroutines can migrate between threads safely)
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

	session, err := managerIface.GetCurrentSession()
	if err != nil {
		log.Printf("[media] GetCurrentSession error: %v", err)
		return "Idle", "", ""
	}
	if session == nil {
		return "Idle", "", ""
	}
	defer session.Release()

	propsAsync, err := session.TryGetMediaPropertiesAsync()
	if err != nil {
		return "Playing", "", ""
	}

	propsRaw, err := awaitAsyncOp(propsAsync)
	propsAsync.Release()
	if err != nil || propsRaw == nil {
		return "Playing", "", ""
	}

	var props *control.GlobalSystemMediaTransportControlsSessionMediaProperties
	iidProps := ole.NewGUID(control.GUIDiGlobalSystemMediaTransportControlsSessionMediaProperties)
	propsIunk := (*ole.IUnknown)(propsRaw)
	if err := propsIunk.PutQueryInterface(iidProps, &props); err != nil {
		propsIunk.Release()
		return "Playing", "", ""
	}
	propsIunk.Release()
	defer props.Release()

	title, _ := props.GetTitle()
	artist, _ := props.GetArtist()

	playbackInfo, err := session.GetPlaybackInfo()
	if err != nil {
		return statusFromTitle(title), title, artist
	}
	defer playbackInfo.Release()

	pstatus, err := playbackInfo.GetPlaybackStatus()
	if err != nil {
		return statusFromTitle(title), title, artist
	}

	status := "Idle"
	switch pstatus {
	case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying:
		status = "Playing"
	}

	return status, title, artist
}

func statusFromTitle(title string) string {
	if title != "" {
		return "Playing"
	}
	return "Idle"
}

// buildStatus gathers all status information and returns JSON bytes.
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

// startStatusPoller periodically polls status and broadcasts to SSE clients.
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
