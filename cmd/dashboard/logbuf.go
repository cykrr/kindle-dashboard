package main

import (
	"log"
	"os"
	"strings"
	"sync"
)

const (
	// logRingLines is how many recent lines are kept in memory for the
	// on-screen log panel.
	logRingLines = 300
	// logMaxBytes is the on-disk size at which the log file is rotated.
	// One backup is kept, so disk use is bounded at ~2×logMaxBytes.
	logMaxBytes = 512 * 1024
)

// logStore is the process-wide log sink installed as the stdlib logger's
// output. It fans each line into two bounded places:
//
//   - an in-memory ring of the last logRingLines lines, for the settings
//     view's log panel, and
//   - a size-capped file on disk (rotated at logMaxBytes, one .1 backup) so
//     the log can't grow without bound on the Kindle's small tmpfs — the
//     root of prod-readiness item #3.
type logStore struct {
	mu    sync.Mutex
	ring  []string
	path  string
	f     *os.File
	size  int64
	maxSz int64
}

var logs *logStore

// initLogStore installs the global log sink and points the stdlib logger at
// it. Any failure to open the file is non-fatal: logging continues in memory.
func initLogStore(path string, maxBytes int64) {
	ls := &logStore{
		ring:  make([]string, 0, logRingLines),
		path:  path,
		maxSz: maxBytes,
	}
	ls.open()
	logs = ls
	log.SetOutput(ls)
}

// open (re)opens the on-disk log for appending and records its current size.
func (ls *logStore) open() {
	f, err := os.OpenFile(ls.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	if fi, err := f.Stat(); err == nil {
		ls.size = fi.Size()
	}
	ls.f = f
}

// Write implements io.Writer for the stdlib log package.
func (ls *logStore) Write(p []byte) (int, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Ring buffer: one entry per line, trailing newline trimmed.
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		ls.ring = append(ls.ring, line)
	}
	if over := len(ls.ring) - logRingLines; over > 0 {
		ls.ring = append(ls.ring[:0], ls.ring[over:]...)
	}

	if ls.f != nil {
		n, _ := ls.f.Write(p)
		ls.size += int64(n)
		if ls.size >= ls.maxSz {
			ls.rotate()
		}
	}
	return len(p), nil
}

// rotate closes the current file, renames it to path+".1" (replacing any
// previous backup), and reopens a fresh empty file. Caller holds ls.mu.
func (ls *logStore) rotate() {
	if ls.f != nil {
		ls.f.Close()
		ls.f = nil
	}
	_ = os.Rename(ls.path, ls.path+".1")
	f, err := os.OpenFile(ls.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	ls.f = f
	ls.size = 0
}

// snapshot returns the buffered log lines, oldest first, as one string.
func (ls *logStore) snapshot() string {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return strings.Join(ls.ring, "\n")
}
