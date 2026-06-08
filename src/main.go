package main

import "time"

func main() {
	dash = NewDashboard()
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
