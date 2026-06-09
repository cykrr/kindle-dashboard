package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type PCStatus struct {
	GamingMode string `json:"gaming_mode"`
	MonitorOn  bool   `json:"monitor_on"`
	Track      string `json:"track"`
	Artist     string `json:"artist"`
	Status     string `json:"status"`
}

type PCMacroClient struct {
	baseURL string
	apiKey  string
	dash    *Dashboard
	http    *http.Client
	stream  *http.Client

	mu         sync.Mutex
	lastStatus PCStatus
}

func NewPCMacroClient(cfg HassConfig, dash *Dashboard) *PCMacroClient {
	return &PCMacroClient{
		baseURL: normalizePCMacroURL(cfg.PCMacroURL),
		apiKey:  strings.TrimSpace(cfg.PCMacroKey),
		dash:    dash,
		http:    &http.Client{Timeout: 5 * time.Second},
		stream:  &http.Client{},
	}
}

func (c *PCMacroClient) Run() {
	for {
		if err := c.RefreshStatus(); err != nil {
			if c.dash != nil {
				c.dash.SetPCConnectionStatus("Disconnected")
			}
			time.Sleep(15 * time.Second)
			continue
		}
		if c.dash != nil {
			c.dash.SetPCConnectionStatus("Streaming")
		}
		if err := c.streamEvents(); err != nil && c.dash != nil {
			c.dash.SetPCConnectionStatus("Reconnecting…")
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *PCMacroClient) Execute(action string) {
	if c == nil || c.baseURL == "" || c.apiKey == "" || strings.TrimSpace(action) == "" {
		if c != nil && c.dash != nil {
			c.dash.SetPCConnectionStatus("Not configured")
		}
		return
	}
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

func (c *PCMacroClient) streamEvents() error {
	endpoint := strings.TrimRight(c.baseURL, "/") + "/events?" + url.Values{"key": {c.apiKey}}.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.stream.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			return fmt.Errorf("pc macro sse http %d", resp.StatusCode)
		}
		return fmt.Errorf("pc macro sse http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(dataLines) > 0 {
				var status PCStatus
				if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &status); err == nil {
					c.applyStatus(status)
				}
				dataLines = dataLines[:0]
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(line[5:]))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}

func (c *PCMacroClient) applyStatus(status PCStatus) {
	c.mu.Lock()
	c.lastStatus = status
	c.mu.Unlock()
	if c.dash != nil {
		c.dash.UpdatePCStatus(status)
	}
}

func (c *PCMacroClient) modeToggleAction() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.EqualFold(c.lastStatus.GamingMode, "power") {
		return "save_mode"
	}
	return "power_mode"
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
