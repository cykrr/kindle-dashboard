package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	batteryCap     = "/sys/class/power_supply/bd71827_bat/capacity"
	batteryVoltage = "/sys/class/power_supply/bd71827_bat/voltage_now"
)

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func getPIDCPU(pid int) (utime, stime int64, err error) {
	statStr := readFile(fmt.Sprintf("/proc/%d/stat", pid))
	if statStr == "" {
		return 0, 0, fmt.Errorf("could not read stat for pid %d", pid)
	}

	// Comm field can contain spaces and parentheses, e.g., (my proc).
	// Find the last closing parenthesis.
	rparen := strings.LastIndex(statStr, ")")
	if rparen < 0 {
		return 0, 0, fmt.Errorf("invalid stat format")
	}

	parts := strings.Fields(statStr[rparen+1:])
	// After "(comm) ", the first field is State (index 0).
	// utime is field 14 in /proc/[pid]/stat, which is index 11 after the closing parenthesis.
	// stime is field 15, index 12.
	if len(parts) < 13 {
		return 0, 0, fmt.Errorf("not enough fields in stat")
	}

	utime, _ = strconv.ParseInt(parts[11], 10, 64)
	stime, _ = strconv.ParseInt(parts[12], 10, 64)
	return utime, stime, nil
}

func main() {
	pid := flag.Int("pid", 0, "PID of the dashboard process to monitor")
	mode := flag.String("mode", "unknown", "Experiment mode (e.g., polling, suspend)")
	duration := flag.Duration("duration", 10*time.Minute, "Total duration to run the experiment")
	interval := flag.Duration("interval", 1*time.Minute, "Interval between measurements")
	outPath := flag.String("out", "battery-experiment.csv", "Path to CSV output file")
	flag.Parse()

	if *pid == 0 {
		log.Fatal("-pid is required")
	}

	file, err := os.OpenFile(*outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open output file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is empty
	info, _ := file.Stat()
	if info.Size() == 0 {
		writer.Write([]string{
			"timestamp",
			"mode",
			"elapsed_sec",
			"pid_utime_ticks",
			"pid_stime_ticks",
			"battery_capacity_pct",
			"battery_voltage_uv",
		})
	}

	log.Printf("Starting experiment '%s' for %v, measuring PID %d every %v", *mode, *duration, *pid, *interval)
	log.Printf("Logging to %s", *outPath)

	startTime := time.Now()
	endTime := startTime.Add(*duration)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Perform initial measurement
	measureAndLog(writer, *mode, startTime, startTime, *pid)

	for {
		now := time.Now()
		if now.After(endTime) {
			break
		}

		select {
		case t := <-ticker.C:
			measureAndLog(writer, *mode, startTime, t, *pid)
		}
	}

	log.Println("Experiment completed successfully.")
}

func measureAndLog(w *csv.Writer, mode string, start, now time.Time, pid int) {
	elapsed := now.Sub(start).Seconds()

	utime, stime, err := getPIDCPU(pid)
	if err != nil {
		log.Printf("warning: %v", err)
	}

	capStr := readFile(batteryCap)
	volStr := readFile(batteryVoltage)

	w.Write([]string{
		now.Format(time.RFC3339),
		mode,
		fmt.Sprintf("%.1f", elapsed),
		fmt.Sprintf("%d", utime),
		fmt.Sprintf("%d", stime),
		capStr,
		volStr,
	})
	w.Flush()

	log.Printf("Logged: elapsed=%.1fs, utime=%d, stime=%d, cap=%s, vol=%s", elapsed, utime, stime, capStr, volStr)
}
