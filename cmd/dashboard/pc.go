package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const pcInactivityTimeout = 15 * time.Second

type PCStatus struct {
	GamingMode string `json:"gaming_mode"`
	MonitorOn  bool   `json:"monitor_on"`
	Track      string `json:"track"`
	Artist     string `json:"artist"`
	Status     string `json:"status"`
}

type PCDashboard interface {
	SetPCConnectionStatus(string)
	UpdatePCStatus(PCStatus)
}

type PCMacroClient struct {
	baseURL string
	apiKey  string
	dash    PCDashboard
	http    *http.Client
	stream  *http.Client

	mu         sync.Mutex
	lastStatus PCStatus

	// Streaming state
	streamBody    io.ReadCloser
	streamCancel  context.CancelFunc
	inactiveTimer *time.Timer
	streaming     bool
}

func NewPCMacroClient(cfg HassConfig, dash PCDashboard) *PCMacroClient {
	return &PCMacroClient{
		baseURL: normalizePCMacroURL(cfg.PCMacroURL),
		apiKey:  strings.TrimSpace(cfg.PCMacroKey),
		dash:    dash,
		http:    &http.Client{Timeout: 5 * time.Second},
		stream:  &http.Client{},
	}
}

// RefreshStatus makes a one-shot HTTP request to get the current PC status.
func (c *PCMacroClient) RefreshStatus() error {
	if c == nil || c.baseURL == "" || c.apiKey == "" {
		return fmt.Errorf("pc macro service not configured")
	}
	body, err := c.get("/status", url.Values{"key": {c.apiKey}, "ts": {fmt.Sprintf("%d", time.Now().UnixNano())}})
	if err != nil {
		return err
	}
	var status PCStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return err
	}
	c.mu.Lock()
	c.lastStatus = status
	c.mu.Unlock()
	c.applyStatus(status)
	if c.dash != nil {
		c.dash.SetPCConnectionStatus("Connected")
	}
	return nil
}

// Touch signals user interaction with the PC Macro panel.
// Opens the SSE stream if closed, and resets the 15-second inactivity timer.
func (c *PCMacroClient) Touch() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.baseURL == "" || c.apiKey == "" {
		return
	}

	if !c.streaming {
		c.startStreamingLocked()
	}
	c.resetInactiveTimerLocked()
}

// StopStreaming forcibly closes the SSE stream.
func (c *PCMacroClient) StopStreaming() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopStreamingLocked()
}

func (c *PCMacroClient) startStreamingLocked() {
	if c.streaming {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.streamCancel = cancel
	c.streaming = true
	go c.streamLoop(ctx)

	if c.dash != nil {
		c.dash.SetPCConnectionStatus("Streaming")
	}
}

func (c *PCMacroClient) stopStreamingLocked() {
	if !c.streaming {
		return
	}
	c.streaming = false
	if c.streamCancel != nil {
		c.streamCancel()
		c.streamCancel = nil
	}
	if c.inactiveTimer != nil {
		c.inactiveTimer.Stop()
		c.inactiveTimer = nil
	}
	if c.streamBody != nil {
		_ = c.streamBody.Close()
		c.streamBody = nil
	}
	if c.dash != nil {
		c.dash.SetPCConnectionStatus("Connected")
	}
}

func (c *PCMacroClient) resetInactiveTimerLocked() {
	if c.inactiveTimer != nil {
		c.inactiveTimer.Stop()
	}
	c.inactiveTimer = time.AfterFunc(pcInactivityTimeout, func() {
		c.mu.Lock()
		c.stopStreamingLocked()
		c.mu.Unlock()
	})
}

// streamLoop runs the SSE event stream until the context is cancelled.
func (c *PCMacroClient) streamLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		endpoint := strings.TrimRight(c.baseURL, "/") + "/events?" + url.Values{"key": {c.apiKey}}.Encode()
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			log.Printf("pc: sse req: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		req = req.WithContext(ctx)

		resp, err := c.stream.Do(req)
		if err != nil {
			log.Printf("pc: sse dial: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		c.mu.Lock()
		if !c.streaming {
			resp.Body.Close()
			c.mu.Unlock()
			return
		}
		c.streamBody = resp.Body
		c.mu.Unlock()

		err = c.readSSE(ctx, resp.Body)
		resp.Body.Close()

		c.mu.Lock()
		c.streamBody = nil
		c.mu.Unlock()

		if err != nil && err != io.EOF && ctx.Err() == nil {
			log.Printf("pc: sse read: %v", err)
		}

		// If still streaming, reconnect after a brief delay
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// readSSE reads SSE events from the response body until context cancellation.
func (c *PCMacroClient) readSSE(ctx context.Context, body io.ReadCloser) error {
	done := make(chan struct{})
	defer close(done)

	// Goroutine to cancel the read if context is cancelled.
	go func() {
		select {
		case <-ctx.Done():
			body.Close()
		case <-done:
		}
	}()

	scanner := bufio.NewScanner(body)
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(dataLines) > 0 {
				var status PCStatus
				if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &status); err == nil {
					c.applyStatus(status)
					// New SSE data counts as activity — reset the timer.
					c.mu.Lock()
					if c.streaming {
						c.resetInactiveTimerLocked()
					}
					c.mu.Unlock()
				}
				dataLines = dataLines[:0]
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(line[5:]))
		}
	}
	return scanner.Err()
}

// Execute sends a macro action command via HTTP. It also calls Touch() to
// extend the SSE window since the user is actively interacting.
func (c *PCMacroClient) Execute(action string) {
	if c == nil || c.baseURL == "" || c.apiKey == "" || strings.TrimSpace(action) == "" {
		if c != nil && c.dash != nil {
			c.dash.SetPCConnectionStatus("Not configured")
		}
		return
	}
	// Extend the streaming window — user is interacting.
	c.Touch()

	if action == "pc_mode_toggle" {
		action = c.modeToggleAction()
	}
	if c.dash != nil {
		c.dash.SetPCConnectionStatus("Sending…")
	}
	if err := c.call("/execute", url.Values{"action": {action}, "key": {c.apiKey}}); err != nil {
		if c.dash != nil {
			c.dash.SetPCConnectionStatus(fmt.Sprintf("Failed: %v", err))
		}
		return
	}
	_ = c.RefreshStatus()
}

func (c *PCMacroClient) modeToggleAction() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.EqualFold(c.lastStatus.GamingMode, "power") {
		return "save_mode"
	}
	return "power_mode"
}

func (c *PCMacroClient) applyStatus(status PCStatus) {
	c.mu.Lock()
	c.lastStatus = status
	c.mu.Unlock()
	if c.dash != nil {
		c.dash.UpdatePCStatus(status)
	}
}

func (c *PCMacroClient) call(path string, values url.Values) error {
	_, err := c.get(path, values)
	return err
}

func (c *PCMacroClient) get(path string, values url.Values) ([]byte, error) {
	endpoint := strings.TrimRight(c.baseURL, "/") + path
	if encoded := values.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(body) == 0 {
			return nil, fmt.Errorf("pc macro http %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("pc macro http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func normalizePCMacroURL(raw string) string {
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return "http://" + raw
}
