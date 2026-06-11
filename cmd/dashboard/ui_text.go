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
