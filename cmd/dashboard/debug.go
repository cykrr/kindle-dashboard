package main

import (
	"log"
	"sync/atomic"
)

var debugLogging atomic.Bool

func setDebugLogging(enabled bool) {
	debugLogging.Store(enabled)
}

func debugLogf(format string, args ...interface{}) {
	if debugLogging.Load() {
		log.Printf(format, args...)
	}
}
