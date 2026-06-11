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
	flag.Parse()

	// Restore the Kindle's launcher UI on exit, however we exit.
	defer RestoreKindleFramework()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		RestoreKindleFramework()
		os.Exit(0)
	}()

	cfg, cfgErr := LoadHassConfig()
	pcEnabled := strings.TrimSpace(cfg.PCMacroURL) != "" && strings.TrimSpace(cfg.PCMacroKey) != ""
	dash = NewDashboard(DashboardOptions{HardwareLandscape: *hwLandscape, HassLightEntities: cfg.LightEntities, PCEnabled: pcEnabled, LauncherButtons: cfg.LauncherButtons})
	dash.Show()
	dash.UpdateClock(time.Now())

	if pcEnabled {
		pcMacroClient = NewPCMacroClient(cfg, dash)
		// One-shot initial status fetch. The SSE stream is opened on-demand
		// when the user navigates to the launcher view.
		if err := pcMacroClient.RefreshStatus(); err != nil {
			log.Printf("pc macro: initial status: %v", err)
			dash.SetPCConnectionStatus("Disconnected")
		}
	} else {
		dash.SetPCConnectionStatus("Not configured")
	}

	if cfgErr == nil {
		hassClient = NewHassClient(cfg, dash)
		go hassClient.Run()
	} else {
		log.Printf("hass disabled: %v", cfgErr)
		dash.SetConnectionStatus("Config Missing")
	}

	if *suspendCycle {
		// Suspend-to-RAM cycle: handles its own clock/poll refresh on each
		// RTC wake, replacing the plain clock goroutine below.
		go runSuspendCycle(dash)
	} else {
		// Clock goroutine — sleeps precisely until the next minute boundary,
		// waking the CPU only when the UI needs to reflect a new minute.
		go func() {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day(),
					now.Hour(), now.Minute()+1, 0, 0, now.Location())
				timer := time.NewTimer(time.Until(next))
				<-timer.C
				dash.UpdateClock(time.Now())
			}
		}()
	}

	// Battery event-driven updates — decoupled from the clock loop.
	// Uses epoll/POLLPRI to wait for kernel sysfs_notify events.
	go WatchBatteryCapacity(context.Background(), dash.UpdateBattery)

	dash.Loop()
}
