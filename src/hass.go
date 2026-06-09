package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HassClient struct {
	cfg  HassConfig
	dash *Dashboard

	mu       sync.Mutex
	conn     *websocketConn
	nextID   int
	authFail bool
	pending  map[int]string
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
	return &HassClient{cfg: cfg, dash: dash, nextID: 1, pending: map[int]string{}}
}

func (h *HassClient) Run() {
	for {
		if h.authFail {
			return
		}
		if err := h.connectAndRead(); err != nil {
			log.Printf("hass: %v", err)
		}
		if h.authFail {
			return
		}
		h.setConnStatus("Disconnected")
		time.Sleep(30 * time.Second)
	}
}

func (h *HassClient) connectAndRead() error {
	wsURL := hassWSURL(h.cfg.URL)
	if wsURL == "" {
		return fmt.Errorf("missing HA websocket url")
	}
	h.setConnStatus("Connecting…")
	conn, err := dialWebSocket(wsURL, h.cfg.InsecureSkipVerify)
	if err != nil {
		h.setConnStatus("Error")
		return err
	}
	defer conn.Close()
	h.mu.Lock()
	h.conn = conn
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		if h.conn == conn {
			h.conn = nil
		}
		h.mu.Unlock()
	}()

	for {
		payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		h.handleMessage(payload)
	}
}

func (h *HassClient) handleMessage(payload []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	typeName, _ := msg["type"].(string)
	switch typeName {
	case "auth_required":
		h.setConnStatus("Authenticating…")
		h.send(map[string]interface{}{"type": "auth", "access_token": h.cfg.Token})
	case "auth_ok":
		h.setConnStatus("Connected")
		h.sendWithID(map[string]interface{}{"type": "get_states"})
		h.sendWithID(map[string]interface{}{"type": "subscribe_events", "event_type": "state_changed"})
		h.RequestCalendarEvents(true)
	case "auth_invalid":
		h.authFail = true
		h.setConnStatus("Auth Failed")
		h.mu.Lock()
		conn := h.conn
		h.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
	case "result":
		h.handleResult(msg)
	case "event":
		h.handleEvent(msg)
	}
}

func (h *HassClient) handleResult(msg map[string]interface{}) {
	id := intNumber(msg["id"])
	h.mu.Lock()
	kind := h.pending[id]
	delete(h.pending, id)
	h.mu.Unlock()

	if kind == "calendar_events" {
		if success, ok := msg["success"].(bool); ok && !success {
			return
		}
		h.dash.UpdateAgenda(parseCalendarData(msg["result"]))
		return
	}

	result, ok := msg["result"].([]interface{})
	if !ok {
		return
	}
	for _, item := range result {
		if st := decodeState(item); st.EntityID != "" {
			h.handleState(st)
		}
	}
}

func (h *HassClient) handleEvent(msg map[string]interface{}) {
	event, _ := msg["event"].(map[string]interface{})
	if typ, _ := event["event_type"].(string); typ != "state_changed" {
		return
	}
	data, _ := event["data"].(map[string]interface{})
	if st := decodeState(data["new_state"]); st.EntityID != "" {
		h.handleState(st)
	}
}

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

func (h *HassClient) ToggleEntity(entity string) {
	if entity == "" || !strings.Contains(entity, ".") {
		return
	}
	domain := strings.SplitN(entity, ".", 2)[0]
	h.sendWithID(map[string]interface{}{
		"type":    "call_service",
		"domain":  domain,
		"service": "toggle",
		"target":  map[string]interface{}{"entity_id": entity},
	})
}

func (h *HassClient) PublishBrightnessToHass(percent int) {
	entity := h.cfg.BrightnessEntity
	if entity == "" {
		return
	}
	domain := strings.SplitN(entity, ".", 2)[0]
	msg := map[string]interface{}{
		"type":   "call_service",
		"domain": domain,
		"target": map[string]interface{}{"entity_id": entity},
	}
	switch domain {
	case "number", "input_number":
		msg["service"] = "set_value"
		msg["service_data"] = map[string]interface{}{"value": percent}
	case "light":
		msg["service"] = "turn_on"
		msg["service_data"] = map[string]interface{}{"brightness_pct": percent}
	default:
		return
	}
	h.sendWithID(msg)
}

func (h *HassClient) RequestCalendarEvents(force bool) {
	if len(h.cfg.CalendarEntities) == 0 {
		return
	}
	start := time.Now()
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end := start.AddDate(0, 0, 7)
	id := h.nextMessageID()
	h.mu.Lock()
	h.pending[id] = "calendar_events"
	h.mu.Unlock()
	h.send(map[string]interface{}{
		"id":              id,
		"type":            "call_service",
		"domain":          "calendar",
		"service":         "get_events",
		"target":          map[string]interface{}{"entity_id": h.cfg.CalendarEntities},
		"service_data":    map[string]interface{}{"start_date_time": start.Format(time.RFC3339), "end_date_time": end.Format(time.RFC3339)},
		"return_response": true,
	})
}

func (h *HassClient) sendWithID(msg map[string]interface{}) {
	msg["id"] = h.nextMessageID()
	h.send(msg)
}

func (h *HassClient) send(msg map[string]interface{}) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	conn := h.conn
	h.mu.Unlock()
	if conn == nil {
		return
	}
	if err := conn.WriteText(b); err != nil {
		log.Printf("hass send: %v", err)
	}
}

func (h *HassClient) nextMessageID() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.nextID
	h.nextID++
	return id
}

func (h *HassClient) setConnStatus(status string) {
	if h.dash != nil {
		h.dash.SetConnectionStatus(status)
	}
}

func hassWSURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "wss://") || strings.HasPrefix(raw, "ws://") {
		if strings.HasSuffix(raw, "/api/websocket") {
			return raw
		}
		return raw + "/api/websocket"
	}
	if strings.HasPrefix(raw, "https://") {
		return "wss://" + strings.TrimPrefix(raw, "https://") + "/api/websocket"
	}
	if strings.HasPrefix(raw, "http://") {
		return "ws://" + strings.TrimPrefix(raw, "http://") + "/api/websocket"
	}
	return "wss://" + strings.TrimLeft(raw, "/") + "/api/websocket"
}

func decodeState(v interface{}) HassState {
	var st HassState
	b, err := json.Marshal(v)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(b, &st)
	return st
}

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
