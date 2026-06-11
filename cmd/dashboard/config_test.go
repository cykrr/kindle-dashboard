package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigNoPersonalHADefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hass-config.json")
	if err := os.WriteFile(path, []byte(`{"url":"http://ha","token":"token"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HASS_CONFIG", path)
	t.Setenv("HASS_MUSIC_ENTITY", "")
	t.Setenv("HASS_CALENDAR_ENTITIES", "")

	cfg, err := LoadHassConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MusicEntity != "" {
		t.Fatalf("MusicEntity defaulted to %q; want empty", cfg.MusicEntity)
	}
	if len(cfg.CalendarEntities) != 0 {
		t.Fatalf("CalendarEntities defaulted to %v; want empty", cfg.CalendarEntities)
	}
}

func TestParseConfigLauncherButtons(t *testing.T) {
	var cfg HassConfig
	if err := parseHassConfigBytes([]byte(`{
		"url":"http://ha",
		"token":"token",
		"launcherButtons":[
			{"action":"launch_browser","icon":"launch_chrome","label":"Browser"},
			{"action":"sleep"}
		]
	}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.LauncherButtons) != 2 {
		t.Fatalf("launcher buttons len = %d; want 2", len(cfg.LauncherButtons))
	}
	if cfg.LauncherButtons[0].Action != "launch_browser" || cfg.LauncherButtons[0].Icon != "launch_chrome" {
		t.Fatalf("unexpected first launcher button: %+v", cfg.LauncherButtons[0])
	}
}

func TestLauncherButtonsDefaultOnlyWhenOmitted(t *testing.T) {
	if got := launcherButtons(nil); len(got) != 9 {
		t.Fatalf("default launcher buttons len = %d; want 9", len(got))
	}
	if got := launcherButtons([]LauncherButtonConfig{}); len(got) != 0 {
		t.Fatalf("explicit empty launcher buttons len = %d; want 0", len(got))
	}
	got := launcherButtons([]LauncherButtonConfig{{Action: ""}, {Action: " sleep "}})
	if len(got) != 1 || got[0].Action != "sleep" {
		t.Fatalf("launcherButtons did not trim/filter actions: %+v", got)
	}
}
