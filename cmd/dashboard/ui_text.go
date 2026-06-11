package main

import (
	"fmt"
	"html"
	"strings"
)

func lightButtonLabel(name, state string) string {
	name = shorten(name, 16)
	on := strings.EqualFold(strings.TrimSpace(state), "on") || strings.EqualFold(strings.TrimSpace(state), "open")
	if on {
		return fmt.Sprintf("<span weight='bold'>● %s</span>", esc(name))
	}
	return fmt.Sprintf("<span>○ %s</span>", esc(name))
}

func esc(s string) string { return html.EscapeString(s) }

func shorten(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "..."
	}
	return string(r[:max-1]) + "..."
}

func agendaDayHeader(day string, itemMax int) string {
	lineLen := itemMax - len([]rune(day)) - 1
	if lineLen < 8 {
		lineLen = 8
	}
	if lineLen > 28 {
		lineLen = 28
	}
	return day + " " + strings.Repeat("─", lineLen)
}

func agendaDisplayRows(a AgendaData, itemMax int) []string {
	rows := make([]string, 0, len(a.Events))
	lastDay := ""
	for _, e := range a.Events {
		line := e.Title
		if e.Time != "" {
			line = e.Time + " " + e.Title
		}
		line = " " + shorten(line, itemMax)
		if e.Day != "" && e.Day != lastDay {
			line = agendaDayHeader(e.Day, itemMax) + "\n" + line
			lastDay = e.Day
		}
		rows = append(rows, line)
	}
	return rows
}
