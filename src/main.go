package main

import (
	"flag"
	"log"
	"strings"
	"time"
)

func main() {
	hwLandscape := flag.Bool("hw-landscape", false, "Ask Kindle window manager for hardware landscape orientation")
	flag.Parse()

	cfg, cfgErr := LoadHassConfig()
	pcEnabled := strings.TrimSpace(cfg.PCMacroURL) != "" && strings.TrimSpace(cfg.PCMacroKey) != ""
	dash = NewDashboard(DashboardOptions{HardwareLandscape: *hwLandscape, HassLightEntities: cfg.LightEntities, PCEnabled: pcEnabled})
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

	// Battery polling goroutine — decoupled from the clock loop.
	// Reads the sysfs capacity file every 5 minutes to minimise I/O.
	go func() {
		// Read immediately on startup, then every 5 minutes.
		dash.UpdateBattery(readBatteryCapacity())
		for {
			time.Sleep(5 * time.Minute)
			dash.UpdateBattery(readBatteryCapacity())
		}
	}()

	dash.Loop()
}
