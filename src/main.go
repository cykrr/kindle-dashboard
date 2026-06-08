package main

import (
	"flag"
	"time"
)

func main() {
	hwLandscape := flag.Bool("hw-landscape", false, "Ask Kindle window manager for hardware landscape orientation")
	flag.Parse()

	dash = NewDashboard(DashboardOptions{HardwareLandscape: *hwLandscape})
	dash.Show()
	dash.UpdateClock(time.Now())

	go func() {
		for {
			time.Sleep(1 * time.Second)
			dash.UpdateClock(time.Now())
		}
	}()

	dash.Loop()
}
