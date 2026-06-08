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
- **Kindle**: Go + GTK2 binary (~1.7MB, 9MB RSS)
- **Windows**: Go daemon for PC macros and media status
- **Home Assistant**: Direct WebSocket connection

## Build Pipeline
1. `./build-all.sh toolchain` — Download pre-built KHF toolchain
2. `./build-all.sh gtk` — Cross-compile GTK 2.24.33
3. `./build-all.sh test` — Build Go binary
4. `./install.sh` — Deploy to Kindle

## Files
| File | Purpose |
|------|---------|
| `src/main.go` | Dashboard app with cgo GTK bindings |
| `launch.sh` | Start native dashboard on Kindle |
| `install.sh` | Deploy to Kindle |
| `build-test.sh` | Cross-compile Go binary |
| `setup-env.sh` | Cross-compilation environment |
| Windows daemon | `main.go`, `config.go`, `sse.go`, `media.go`, `actions.go`, `powershell.go` |
