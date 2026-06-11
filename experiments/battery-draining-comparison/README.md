# Battery Draining Comparison

This directory contains a small Go tool to measure the CPU and battery usage of the Kindle dashboard over a set period of time (e.g., 10 minutes).

## How to build

From the root of the repository, cross-compile the tool for the Kindle:

```bash
export GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0
go build -o deploy/battery-experiment experiments/battery-draining-comparison/main.go
```

## How to deploy and run

Copy it to the Kindle:
```bash
scp -P 2222 deploy/battery-experiment root@192.168.1.84:/mnt/us/documents/kindle-dashboard/
```

SSH into the Kindle and run the experiment for the two different configurations.

**1. Run with standard polling (no suspend-cycle)**
Make sure the dashboard is launched **without** the `-suspend-cycle` flag.
```bash
# Find the PID of the dashboard
PID=$(pidof dashboard-native)

# Run a 10-minute experiment
/mnt/us/documents/kindle-dashboard/battery-experiment -pid $PID -mode "polling" -duration 10m -interval 1m -out /tmp/battery-experiment.csv
```

**2. Run with suspend-cycle**
Restart the dashboard **with** the `-suspend-cycle` flag.
```bash
PID=$(pidof dashboard-native)

# Note: Since the device will suspend, the experiment tool might pause execution while suspended 
# and log bursts of data when it wakes up. This is fine; it will still capture CPU ticks and battery stats.
/mnt/us/documents/kindle-dashboard/battery-experiment -pid $PID -mode "suspend-cycle" -duration 10m -interval 1m -out /tmp/battery-experiment.csv
```

After both runs, you can download `/tmp/battery-experiment.csv` from the Kindle to compare the CPU ticks (`pid_utime_ticks`, `pid_stime_ticks`), and voltage drops (`battery_voltage_uv`).
