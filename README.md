# Kindle Dashboard (Native GTK)

A native GTK+ 2.24 dashboard for jailbroken e-ink Kindles. Replaces the Chromium-based dashboard with a ~9MB RSS native app for maximum battery life.

## Features
- E-ink optimized high-contrast UI
- Real-time clock
- Calendar
- Brightness control (direct sysfs)
- Battery status
- Home Assistant integration (WebSocket)
- PC media/macro control (SSE + HTTP)

## Architecture
- **Kindle**: Go + GTK2 binary (`cmd/dashboard/`) — ~1.7MB, 9MB RSS
- **Windows**: Go daemon (`cmd/daemon/`) — PC macros and media status
- **Home Assistant**: Direct WebSocket connection

## Build Pipeline
1. `./scripts/build/build-all.sh toolchain` — Download pre-built KHF toolchain
2. `./scripts/build/build-all.sh gtk` — Cross-compile GTK 2.24.33
3. `./scripts/build/build-all.sh test` — Build Go binary
4. `./scripts/kindle/install.sh` — Deploy to Kindle

## Repository Layout
```
cmd/daemon/        Windows Macro Daemon
cmd/dashboard/     Kindle GTK Dashboard
scripts/build/     Cross-compilation pipeline
scripts/kindle/    Kindle deployment scripts
scripts/windows/   Windows daemon installer
legacy/            Archived Chromium-based web dashboard
docs/              Documentation & screenshots
deploy/            Build output (gitignored)
```

## Quick Start
```bash
./scripts/build/build-all.sh bootstrap   # Full build
./scripts/kindle/install.sh              # Deploy to Kindle
```
