package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LauncherButtonConfig struct {
	Action string `json:"action"`
	Icon   string `json:"icon,omitempty"`
	Label  string `json:"label,omitempty"`
}

type HassConfig struct {
	URL                string                 `json:"url"`
	Token              string                 `json:"token"`
	HassURL            string                 `json:"HASS_URL"`
	HassToken          string                 `json:"HASS_TOKEN"`
	Entity             string                 `json:"entity"`
	MusicEntity        string                 `json:"musicEntity"`
	MailEntity         string                 `json:"mailEntity"`
	MailLabel          string                 `json:"mailLabel"`
	CalendarEntity     string                 `json:"calendarEntity"`
	CalendarEntities   []string               `json:"calendarEntities"`
	LightEntity        string                 `json:"lightEntity"`
	LightEntities      []string               `json:"lightEntities"`
	PCMacroURL         string                 `json:"pcMacroUrl"`
	PCMacroKey         string                 `json:"pcMacroKey"`
	BrightnessEntity   string                 `json:"brightnessEntity"`
	LauncherButtons    []LauncherButtonConfig `json:"launcherButtons"`
	InsecureSkipVerify bool                   `json:"insecureSkipVerify"`
}

func (c *HassConfig) UnmarshalJSON(b []byte) error {
	type alias HassConfig
	var aux struct {
		alias
		CalendarEntities json.RawMessage `json:"calendarEntities"`
		LightEntities    json.RawMessage `json:"lightEntities"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*c = HassConfig(aux.alias)
	if entities, err := parseStringListField(aux.CalendarEntities, "calendarEntities"); err != nil {
		return err
	} else if entities != nil {
		c.CalendarEntities = entities
	}
	if entities, err := parseStringListField(aux.LightEntities, "lightEntities"); err != nil {
		return err
	} else if entities != nil {
		c.LightEntities = entities
	}
	return nil
}

func parseStringListField(raw json.RawMessage, field string) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return splitEntities(single), nil
	}
	return nil, fmt.Errorf("%s must be a string or array", field)
}

func LoadHassConfig() (HassConfig, error) {
	var cfg HassConfig
	paths := hassConfigPaths()
	var lastErr error
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		if err := parseHassConfigBytes(b, &cfg); err != nil {
			lastErr = fmt.Errorf("%s: %w", path, err)
			continue
		}
		break
	}

	applyHassEnv(&cfg)
	if cfg.URL == "" {
		cfg.URL = cfg.HassURL
	}
	if cfg.Token == "" {
		cfg.Token = cfg.HassToken
	}
	if cfg.MusicEntity == "" {
		cfg.MusicEntity = cfg.Entity
	}
	if len(cfg.CalendarEntities) == 0 && cfg.CalendarEntity != "" {
		cfg.CalendarEntities = splitEntities(cfg.CalendarEntity)
	}
	cfg.CalendarEntities = splitEntities(strings.Join(cfg.CalendarEntities, ","))
	if len(cfg.LightEntities) == 0 && cfg.LightEntity != "" {
		cfg.LightEntities = splitEntities(cfg.LightEntity)
	}
	cfg.LightEntities = splitEntities(strings.Join(cfg.LightEntities, ","))

	if cfg.URL == "" || cfg.Token == "" || strings.HasPrefix(cfg.Token, "YOUR_") || cfg.Token == "placeholder" {
		if lastErr != nil {
			return cfg, lastErr
		}
		return cfg, fmt.Errorf("missing Home Assistant url/token")
	}
	return cfg, nil
}

func hassConfigPaths() []string {
	seen := map[string]bool{}
	var paths []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	if p := os.Getenv("HASS_CONFIG"); p != "" {
		add(p)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, "hass-config.js"))
		add(filepath.Join(dir, "hass-config.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "hass-config.js"))
		add(filepath.Join(cwd, "hass-config.json"))
		add(filepath.Join(cwd, "..", "hass-config.js"))
	}
	add("/mnt/us/documents/kindle-dashboard/hass-config.js")
	return paths
}

func parseHassConfigBytes(b []byte, cfg *HassConfig) error {
	s := strings.TrimSpace(string(b))
	if strings.Contains(s, "window.HASS_CONFIG") {
		start := strings.Index(s, "{")
		end := strings.LastIndex(s, "}")
		if start < 0 || end <= start {
			return fmt.Errorf("could not find JSON object")
		}
		s = s[start : end+1]
	}
	return json.Unmarshal([]byte(s), cfg)
}

func applyHassEnv(cfg *HassConfig) {
	if v := os.Getenv("HASS_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("HASS_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("HASS_MUSIC_ENTITY"); v != "" {
		cfg.MusicEntity = v
	}
	if v := os.Getenv("HASS_MAIL_ENTITY"); v != "" {
		cfg.MailEntity = v
	}
	if v := os.Getenv("HASS_CALENDAR_ENTITIES"); v != "" {
		cfg.CalendarEntities = splitEntities(v)
	}
	if v := os.Getenv("HASS_LIGHT_ENTITIES"); v != "" {
		cfg.LightEntities = splitEntities(v)
	}
	if v := os.Getenv("PC_MACRO_URL"); v != "" {
		cfg.PCMacroURL = v
	}
	if v := os.Getenv("PC_MACRO_KEY"); v != "" {
		cfg.PCMacroKey = v
	} else if v := os.Getenv("MACRO_API_KEY"); v != "" {
		cfg.PCMacroKey = v
	}
	if v := os.Getenv("HASS_BRIGHTNESS_ENTITY"); v != "" {
		cfg.BrightnessEntity = v
	}
	if v := os.Getenv("HASS_INSECURE_SKIP_VERIFY"); v == "1" || strings.EqualFold(v, "true") {
		cfg.InsecureSkipVerify = true
	}
}

func splitEntities(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
