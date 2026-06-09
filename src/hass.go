package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const hassPollInterval = 2 * time.Minute

type HassClient struct {
	cfg  HassConfig
	dash *Dashboard

	mu     sync.Mutex
	http   *http.Client
	stopCh chan struct{}
}

type HassState struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged string                 `json:"last_changed"`
	LastUpdated string                 `json:"last_updated"`
}

type MusicData struct {
	Badge   string
	Device  string
	Summary string
	State   string
	Track   string
	Artist  string
	Album   string
	Source  string
}

type MailData struct {
	Unread  int
	Summary string
}

type AgendaData struct {
	Summary string
	Events  []AgendaEvent
}

type AgendaEvent struct {
	Time   string
	Title  string
	Detail string
}

type LightData struct {
	EntityID string
	Name     string
	State    string
}

func NewHassClient(cfg HassConfig, dash *Dashboard) *HassClient {
	return &HassClient{
		cfg:  cfg,
		dash: dash,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// Run starts the periodic polling loop. Blocks until Stop() is called.
func (h *HassClient) Run() {
	// Initial fetch on startup so the UI populates quickly.
	if err := h.fetchAll(); err != nil {
		log.Printf("hass: initial fetch failed: %v", err)
		h.setConnStatus("Error")
	} else {
		h.setConnStatus("Connected")
	}

	ticker := time.NewTicker(hassPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.fetchAll(); err != nil {
				log.Printf("hass: poll failed: %v", err)
				h.setConnStatus("Error")
				continue
			}
			h.setConnStatus("Connected")
		case <-h.stopCh:
			return
		}
	}
}

// Stop signals the polling loop to exit.
func (h *HassClient) Stop() {
	if h.stopCh != nil {
		close(h.stopCh)
	}
}

// fetchAll retrieves all entity states from the REST API and dispatches them.
func (h *HassClient) fetchAll() error {
	states, err := h.getStates("")
	if err != nil {
		return err
	}
	for _, st := range states {
		h.handleState(st)
	}
	return nil
}

// fetchOne retrieves a single entity state by ID.
func (h *HassClient) fetchOne(entityID string) (HassState, error) {
	states, err := h.getStates(entityID)
	if err != nil {
		return HassState{}, err
	}
	if len(states) == 0 {
		return HassState{}, fmt.Errorf("entity %q not found", entityID)
	}
	return states[0], nil
}

// getStates fetches entity states from the REST API.
// If entityID is empty, fetches all states; otherwise a single entity.
func (h *HassClient) getStates(entityID string) ([]HassState, error) {
	path := "/api/states"
	if entityID != "" {
		path += "/" + entityID
	}
	body, err := h.restGet(path)
	if err != nil {
		return nil, err
	}
	if entityID != "" {
		var st HassState
		if err := json.Unmarshal(body, &st); err != nil {
			return nil, fmt.Errorf("decode state: %w", err)
		}
		return []HassState{st}, nil
	}
	var states []HassState
	if err := json.Unmarshal(body, &states); err != nil {
		return nil, fmt.Errorf("decode states: %w", err)
	}
	return states, nil
}

// restGet performs an authenticated GET request to the HASS API.
func (h *HassClient) restGet(path string) ([]byte, error) {
	url := strings.TrimRight(h.cfg.URL, "/") + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d %s: %s", resp.StatusCode, path, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// restPost performs an authenticated POST request to the HASS API.
func (h *HassClient) restPost(path string, payload interface{}) ([]byte, error) {
	url := strings.TrimRight(h.cfg.URL, "/") + path
	var b []byte
	if payload != nil {
		var err error
		b, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d %s: %s", resp.StatusCode, path, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// ToggleEntity toggles a light/entity via REST and immediately fetches its state.
func (h *HassClient) ToggleEntity(entity string) {
	if entity == "" || !strings.Contains(entity, ".") {
		return
	}
	domain := strings.SplitN(entity, ".", 2)[0]
	payload := map[string]interface{}{
		"entity_id": entity,
	}
	if _, err := h.restPost("/api/services/"+domain+"/toggle", payload); err != nil {
		log.Printf("hass: toggle %s: %v", entity, err)
		return
	}
	// Brief pause for HA to process the state change, then fetch the result.
	time.Sleep(300 * time.Millisecond)
	if st, err := h.fetchOne(entity); err == nil {
		h.handleState(st)
	} else {
		log.Printf("hass: fetch after toggle %s: %v", entity, err)
	}
}

// PublishBrightnessToHass publishes brightness via REST and fetches the result.
func (h *HassClient) PublishBrightnessToHass(percent int) {
	entity := h.cfg.BrightnessEntity
	if entity == "" {
		return
	}
	domain := strings.SplitN(entity, ".", 2)[0]
	payload := map[string]interface{}{
		"entity_id": entity,
	}
	switch domain {
	case "number", "input_number":
		payload["value"] = percent
		if _, err := h.restPost("/api/services/"+domain+"/set_value", payload); err != nil {
			log.Printf("hass: brightness set %s: %v", entity, err)
			return
		}
	case "light":
		payload["brightness_pct"] = percent
		if _, err := h.restPost("/api/services/"+domain+"/turn_on", payload); err != nil {
			log.Printf("hass: brightness set %s: %v", entity, err)
			return
		}
	default:
		return
	}
	time.Sleep(300 * time.Millisecond)
	if st, err := h.fetchOne(entity); err == nil {
		h.handleState(st)
	} else {
		log.Printf("hass: fetch after brightness %s: %v", entity, err)
	}
}

// RequestCalendarEvents fetches the next 7 days of calendar events.
func (h *HassClient) RequestCalendarEvents(force bool) {
	if len(h.cfg.CalendarEntities) == 0 {
		return
	}
	start := time.Now()
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end := start.AddDate(0, 0, 7)
	payload := map[string]interface{}{
		"entity_id":         h.cfg.CalendarEntities,
		"start_date_time":   start.Format(time.RFC3339),
		"end_date_time":     end.Format(time.RFC3339),
		"return_response":   true,
	}
	body, err := h.restPost("/api/services/calendar/get_events", payload)
	if err != nil {
		log.Printf("hass: calendar events: %v", err)
		return
	}

	// REST API returns an array of entity state objects, not the
	// WebSocket-style map with a "response" key.  Each state object
	// carries calendar events in attributes.events.
	var states []map[string]interface{}
	if err := json.Unmarshal(body, &states); err != nil {
		log.Printf("hass: calendar decode: %v", err)
		return
	}

	// Re-shape into the format parseCalendarData expects:
	//   response[entityID] = {events: [...]}
	response := make(map[string]interface{})
	for _, st := range states {
		eid, _ := st["entity_id"].(string)
		if eid == "" {
			continue
		}
		attrs, _ := st["attributes"].(map[string]interface{})
		if attrs == nil {
			continue
		}
		response[eid] = map[string]interface{}{
			"events": attrs["events"],
		}
	}

	wrapped := map[string]interface{}{
		"response": response,
	}
	h.dash.UpdateAgenda(parseCalendarData(wrapped))
}

// handleState dispatches state updates to the dashboard.
func (h *HassClient) handleState(st HassState) {
	switch st.EntityID {
	case h.cfg.MusicEntity:
		h.dash.UpdateMusic(parseMusicData(st))
	case h.cfg.MailEntity:
		h.dash.UpdateMail(parseMailData(st, h.cfg.MailLabel))
	}
	if containsEntity(h.cfg.LightEntities, st.EntityID) {
		h.dash.UpdateLight(parseLightData(st))
	}
	if containsEntity(h.cfg.CalendarEntities, st.EntityID) {
		h.RequestCalendarEvents(true)
	}
	if h.cfg.BrightnessEntity != "" && st.EntityID == h.cfg.BrightnessEntity {
		h.handleBrightnessState(st)
	}
}

func (h *HassClient) handleBrightnessState(st HassState) {
	percent, ok := brightnessPercentFromState(st)
	if !ok || h.dash == nil {
		return
	}
	current := h.dash.BrightnessPercent(readBrightness())
	if math.Abs(float64(percent-current)) > 2 {
		h.dash.SetBrightnessPercent(percent)
	}
}

func (h *HassClient) setConnStatus(status string) {
	if h.dash != nil {
		h.dash.SetConnectionStatus(status)
	}
}

// ── Parsers — unchanged from original ──

func parseMusicData(st HassState) MusicData {
	attrs := st.Attributes
	playerState := fallback(st.State, "unknown")
	device := fallback(attrString(attrs, "friendly_name"), st.EntityID, "Home Assistant player")
	source := fallback(attrString(attrs, "app_name"), attrString(attrs, "source"), "Home Assistant")
	badge := "IDLE"
	switch strings.ToLower(playerState) {
	case "playing":
		badge = "PLAYING"
	case "paused":
		badge = "PAUSED"
	case "off", "standby":
		badge = "OFF"
	}
	track := fallback(attrString(attrs, "media_title"), device)
	artist := fallback(attrString(attrs, "media_artist"), attrString(attrs, "media_album_artist"))
	return MusicData{
		Badge:   badge,
		Device:  device,
		Summary: strings.Join(nonEmpty([]string{source, playerState}), " • "),
		State:   playerState,
		Track:   track,
		Artist:  artist,
		Album:   attrString(attrs, "media_album_name"),
		Source:  source,
	}
}

func parseMailData(st HassState, label string) MailData {
	unread := intNumber(st.State)
	if unread == 0 {
		unread = intNumber(st.Attributes["unread"])
	}
	if unread == 0 {
		unread = intNumber(st.Attributes["unseen"])
	}
	if label == "" {
		label = fallback(attrString(st.Attributes, "friendly_name"), "Mail")
	}
	summary := "Inbox quiet"
	if unread > 0 {
		summary = fmt.Sprintf("%d unread from %s", unread, label)
	}
	return MailData{Unread: unread, Summary: summary}
}

func parseLightData(st HassState) LightData {
	name := fallback(attrString(st.Attributes, "friendly_name"), prettyEntityName(st.EntityID), st.EntityID)
	state := strings.ToUpper(fallback(st.State, "unknown"))
	return LightData{EntityID: st.EntityID, Name: name, State: state}
}

func parseCalendarData(result interface{}) AgendaData {
	root, _ := result.(map[string]interface{})
	response, _ := root["response"].(map[string]interface{})
	if response == nil {
		response = root
	}
	var events []AgendaEvent
	for entity, rawBucket := range response {
		bucket, _ := rawBucket.(map[string]interface{})
		rawEvents := bucket["events"]
		arr, ok := rawEvents.([]interface{})
		if !ok && rawEvents != nil {
			arr = []interface{}{rawEvents}
		}
		for _, rawEvent := range arr {
			evt, _ := rawEvent.(map[string]interface{})
			title := fallback(attrString(evt, "summary"), attrString(evt, "title"), "Calendar event")
			events = append(events, AgendaEvent{Time: formatEventTime(evt), Title: title, Detail: fallback(attrString(evt, "location"), attrString(evt, "description"), entity)})
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Time < events[j].Time })
	if len(events) > 4 {
		events = events[:4]
	}
	summary := "No upcoming events"
	if len(events) > 0 {
		summary = "Home Assistant calendar"
	}
	return AgendaData{Summary: summary, Events: events}
}

func formatEventTime(event map[string]interface{}) string {
	var raw string
	if start, ok := event["start"].(map[string]interface{}); ok {
		raw = fallback(attrString(start, "dateTime"), attrString(start, "date"))
	}
	if raw == "" {
		raw = fallback(attrString(event, "start_time"), attrString(event, "start"))
	}
	if raw == "" {
		return ""
	}
	if len(raw) == 10 && raw[4] == '-' && raw[7] == '-' {
		return "all day"
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Local().Format("15:04")
	}
	if t, err := time.Parse("2006-01-02 15:04:05", raw); err == nil {
		return t.Format("15:04")
	}
	return ""
}

func brightnessPercentFromState(st HassState) (int, bool) {
	if strings.HasPrefix(st.EntityID, "light.") {
		if st.State == "off" {
			return 0, true
		}
		if v, ok := st.Attributes["brightness"]; ok {
			return int(math.Round(float64(intNumber(v)) / 255 * 100)), true
		}
		return 100, true
	}
	v, err := strconv.ParseFloat(st.State, 64)
	if err != nil {
		return 0, false
	}
	return int(math.Round(v)), true
}

func attrString(attrs map[string]interface{}, key string) string {
	if attrs == nil {
		return ""
	}
	switch v := attrs[key].(type) {
	case string:
		return v
	case float64:
		if math.Trunc(v) == v {
			return strconv.Itoa(int(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

func intNumber(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case json.Number:
		i, _ := x.Int64()
		return int(i)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return int(f)
	default:
		return 0
	}
}

func fallback(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func nonEmpty(values []string) []string {
	out := values[:0]
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func prettyEntityName(entity string) string {
	name := entity
	if idx := strings.Index(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.NewReplacer("_", " ", "-", " ").Replace(name)
	parts := strings.Fields(name)
	for i, part := range parts {
		if part == "" {
			continue
		}
		r := []rune(strings.ToLower(part))
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		parts[i] = string(r)
	}
	if len(parts) == 0 {
		return entity
	}
	return strings.Join(parts, " ")
}

func containsEntity(entities []string, entity string) bool {
	for _, v := range entities {
		if v == entity {
			return true
		}
	}
	return false
}
