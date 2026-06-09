package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
)

//go:generate go run github.com/saltosystems/winrt-go/cmd/winrt-go-gen

func main() {
	// Load configuration from environment
	cfg = loadConfig()

	// Set up logging
	logFile, err := os.OpenFile(cfg.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(logFile)
	} else {
		log.Printf("Warning: could not open log file %s: %v", cfg.LogPath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stop channel — closed when user clicks Exit in the systray
	stopCh := make(chan struct{})

	// Start SSE broker
	broker = NewSSEBroker()
	go broker.Run(ctx)

	// Start periodic status polling
	go startStatusPoller(ctx)

	// Register HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", handleExecute)
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/events", handleSSE)

	srv := &http.Server{Addr: cfg.Port, Handler: mux}

	// Run HTTP server in a goroutine
	go func() {
		fmt.Printf("Windows Macro Service starting on %s (key=%s)\n", cfg.Port, maskKey(cfg.APIKey))
		log.Printf("Starting on %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Show desktop notification that daemon is ready
	notifyReady()

	// Run the system tray (blocks on the main thread on Windows)
	startTray(stopCh)

	// User clicked Exit — graceful shutdown
	log.Println("Shutting down...")
	cancel()
	srv.Shutdown(context.Background())
}

// notifyReady sends a quick toast-like balloon notification via PowerShell.
func notifyReady() {
	// Use a low-power PowerShell popup — hidden, short-lived.
	_ = runPowerShell(false, `
	$pop = New-Object -ComObject Wscript.Shell
	$pop.Popup("Kindle Macro Daemon is running", 2, "Kindle Dashboard", 64)
	`)
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[:4] + "****"
}

// handleStatus serves GET /status as JSON.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Query().Get("key") != cfg.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Write(buildStatus())
}

// handleSSE serves GET /events as an SSE stream.
func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if r.URL.Query().Get("key") != cfg.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan []byte, 8)
	broker.register <- messageChan
	defer func() { broker.unregister <- messageChan }()

	ctx := r.Context()
	for {
		select {
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", string(msg))
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}
