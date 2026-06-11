package main

import (
	"strings"
	"testing"
)

func TestPrettyEntityName(t *testing.T) {
	got := prettyEntityName("light.office_lamp")
	if got != "Office Lamp" {
		t.Fatalf("prettyEntityName = %q", got)
	}
}

func TestBrightnessPercentFromState(t *testing.T) {
	got, ok := brightnessPercentFromState(HassState{EntityID: "light.desk", State: "on", Attributes: map[string]interface{}{"brightness": float64(128)}})
	if !ok || got != 50 {
		t.Fatalf("light brightness = %d, %v; want 50, true", got, ok)
	}

	got, ok = brightnessPercentFromState(HassState{EntityID: "number.kindle_brightness", State: "42"})
	if !ok || got != 42 {
		t.Fatalf("number brightness = %d, %v; want 42, true", got, ok)
	}
}

func TestParseCalendarDataSortsAndLimits(t *testing.T) {
	data := map[string]interface{}{
		"response": map[string]interface{}{
			"calendar.home": map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{"summary": "Late", "start_time": "2026-06-11 20:00:00"},
					map[string]interface{}{"summary": "Early", "start_time": "2026-06-11 08:00:00"},
					map[string]interface{}{"summary": "All Day", "start": map[string]interface{}{"date": "2026-06-11"}},
					map[string]interface{}{"summary": "Middle", "start_time": "2026-06-11 12:00:00"},
					map[string]interface{}{"summary": "Extra", "start_time": "2026-06-11 21:00:00"},
				},
			},
		},
	}

	agenda := parseCalendarData(data)
	if agenda.Summary != "Home Assistant calendar" {
		t.Fatalf("summary = %q", agenda.Summary)
	}
	if len(agenda.Events) != 4 {
		t.Fatalf("events len = %d; want 4", len(agenda.Events))
	}
	if agenda.Events[0].Title != "All Day" {
		t.Fatalf("first event = %q; want All Day", agenda.Events[0].Title)
	}
	if agenda.Events[1].Title != "Early" || agenda.Events[1].Time != "08.00" || agenda.Events[1].Day == "" {
		t.Fatalf("second event = %+v; want Early at 08.00 with day", agenda.Events[1])
	}
}

func TestAgendaDisplayRowsGroupsByDay(t *testing.T) {
	agenda := AgendaData{Events: []AgendaEvent{
		{Day: "Monday", Time: "19.00", Title: "Take dog to vet"},
		{Day: "Wednesday", Time: "12.00", Title: "Dentist appointment"},
	}}
	rows := agendaDisplayRows(agenda, 80)
	want := []string{
		"Monday " + strings.Repeat("─", 28) + "\n 19.00 Take dog to vet",
		"Wednesday " + strings.Repeat("─", 28) + "\n 12.00 Dentist appointment",
	}
	if len(rows) != len(want) {
		t.Fatalf("rows len = %d; want %d: %#v", len(rows), len(want), rows)
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Fatalf("row %d = %q; want %q", i, rows[i], want[i])
		}
	}
}

func TestNormalizePCMacroURL(t *testing.T) {
	cases := map[string]string{
		"host:8080/":         "http://host:8080",
		" http://host:8080 ": "http://host:8080",
		"https://host/":      "https://host",
		"":                   "",
	}
	for input, want := range cases {
		if got := normalizePCMacroURL(input); got != want {
			t.Fatalf("normalizePCMacroURL(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestShorten(t *testing.T) {
	if got := shorten("abcdef", 4); got != "abc..." {
		t.Fatalf("shorten = %q", got)
	}
	if got := shorten("abc", 4); got != "abc" {
		t.Fatalf("shorten short = %q", got)
	}
}
