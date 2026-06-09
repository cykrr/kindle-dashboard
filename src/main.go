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
		go pcMacroClient.Run()
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

	go func() {
		for {
			time.Sleep(1 * time.Second)
			dash.UpdateClock(time.Now())
		}
	}()

	dash.Loop()
}
