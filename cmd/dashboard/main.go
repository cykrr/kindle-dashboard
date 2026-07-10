package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	hwLandscape := flag.Bool("hw-landscape", false, "Ask Kindle window manager for hardware landscape orientation")
	suspendCycle := flag.Bool("suspend-cycle", false, "Suspend to RAM each minute, waking via RTC alarm (experimental power saving)")
	debug := flag.Bool("debug", false, "Enable verbose dashboard debug logging")
	flag.Parse()
	// Own the log file in-process with a size cap + one backup so it can't
	// grow without bound on tmpfs, and keep the last lines in memory for the
	// settings-view log panel.
	initLogStore("/tmp/dashboard-native.log", logMaxBytes)
	setDebugLogging(*debug)

	// Restore the Kindle's launcher UI on exit, however we exit.
	defer RestoreKindleFramework()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		RestoreKindleFramework()
		os.Exit(0)
	}()

	// Re-exec loop: if the user clicks Restart, restartFlag is set and we
	// loop back after gtk_main_quit() instead of exiting.
	for {
		restartFlag.Store(false)
		runDashboard(*hwLandscape, *suspendCycle)
		if !restartFlag.Load() {
			break
		}
		log.Println("main: restarting dashboard")
	}
}

func runDashboard(hwLandscape, suspendCycle bool) {
	cfg, cfgErr := LoadHassConfig()
	pcEnabled := strings.TrimSpace(cfg.PCMacroURL) != "" && strings.TrimSpace(cfg.PCMacroKey) != ""

	if pcEnabled {
		pcMacroClient = NewPCMacroClient(cfg, nil)
		// Sync icons synchronously so w_make_icon can find them when building the UI
		pcMacroClient.SyncCatalogIcons()
	}

	dash = NewDashboard(DashboardOptions{HardwareLandscape: hwLandscape, HassLightEntities: cfg.LightEntities, PCEnabled: pcEnabled, LauncherButtons: cfg.LauncherButtons})
	dash.Show()
	dash.UpdateClock(time.Now())

	if pcEnabled {
		pcMacroClient.dash = dash
		// One-shot initial status fetch. The SSE stream is opened on-demand
		// when the user navigates to the launcher view.
		if err := pcMacroClient.RefreshStatus(); err != nil {
			log.Printf("pc macro: initial status: %v", err)
			dash.SetPCConnectionStatus("Disconnected")
		}
	} else {
		dash.SetPCConnectionStatus("Not configured")
	}

	// Cancelled when this dashboard instance tears down (exit or in-app
	// restart) so every long-lived goroutine below stops instead of leaking
	// into the next run. Without this, a Restart would stack a second power
	// button watcher (which grabs input exclusively) on top of the old one.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfgErr == nil {
		hassClient = NewHassClient(cfg, dash)
		go hassClient.Run()
		defer hassClient.Stop()
	} else {
		log.Printf("hass disabled: %v", cfgErr)
		dash.SetConnectionStatus("Config Missing")
	}

	if pcEnabled {
		defer pcMacroClient.StopStreaming()
	}

	if suspendCycle {
		// Suspend-to-RAM cycle: handles its own clock/poll refresh on each
		// RTC wake, replacing the plain clock goroutine below.
		go runSuspendCycle(ctx, dash)
	} else {
		// Clock goroutine — sleeps precisely until the next minute boundary,
		// waking the CPU only when the UI needs to reflect a new minute.
		go func() {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(),
					now.Hour(), now.Minute()+1, 0, 0, now.Location())
				timer := time.NewTimer(time.Until(next))
				select {
				case <-timer.C:
					dash.UpdateClock(time.Now())
				case <-ctx.Done():
					timer.Stop()
					return
				}
			}
		}()
	}

	// Power button monitor — intercepts the physical power button press
	// before the Kindle OS can suspend, and jumps to the rest screen (ViewHome)
	// instead. Runs regardless of suspend-cycle mode.
	go WatchPowerButton(ctx, dash)

	// Battery event-driven updates — decoupled from the clock loop.
	// Uses epoll/POLLPRI to wait for kernel sysfs_notify events.
	go WatchBatteryCapacity(ctx, dash.UpdateBattery)

	// Configure static IP for wlan0 (safe — runtime only, lost on reboot)
	if err := setWifiStaticIP(); err != nil {
		log.Printf("network: static IP: %v", err)
	}

	// Refresh network labels on startup
	dash.updateNetworkLabels()

	dash.Loop()
}
