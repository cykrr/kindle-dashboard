package main

import (
	"os"
	"strconv"
)

// Config holds all configurable parameters for the daemon.
type Config struct {
	APIKey string
	Port   string
	Gsudo  string

	LogPath string
}

func loadConfig() Config {
	cfg := Config{
		APIKey:  envOrDefault("MACRO_API_KEY", "your-super-secret-key"),
		Port:    ":" + envOrDefault("MACRO_PORT", "8080"),
		Gsudo:   envOrDefault("MACRO_GSUDO", `C:\Users\krr\scoop\apps\gsudo\current\gsudo.exe`),
		LogPath: envOrDefault("MACRO_LOG_PATH", `C:\KindleDashboard\macro-daemon.log`),
	}
	return cfg
}

// cfg is the package-level config, initialized in main().
var cfg Config

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrDefaultInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
