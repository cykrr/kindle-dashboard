# Agent and Developer Guide (Native Branch)

Welcome! This branch contains the **native GTK+ 2.24** version of the Kindle Dashboard.

## Repository Structure

### Kindle Dashboard (native binary — runs on Kindle)
- **[src/main.go](src/main.go)**: Go GTK2 dashboard — clock, calendar, brightness, battery, HASS, PC integration
- **[src/go.mod](src/go.mod)**: Go module (no external deps)
- **[launch.sh](launch.sh)**: Stops Kindle GUI, starts native binary
- **[stop.sh](stop.sh)**: Stops native binary, restores Kindle GUI
- **[install.sh](install.sh)**: Cross-compiles and deploys to Kindle

### Build Pipeline
| Script | Purpose |
|--------|---------|
| **[setup-env.sh](setup-env.sh)** | Cross-compilation environment (KHF toolchain) |
| **[build-all.sh](build-all.sh)** | Orchestrator: toolchain → SDK → GTK → Go test |
| **[build-gtk.sh](build-gtk.sh)** | Cross-compiles GTK 2.24.33 + cairo with X11 |
| **[build-test.sh](build-test.sh)** | Cross-compiles the Go binary |

### Windows Macro Daemon (Go backend — runs on Windows)
- **[main.go](main.go)**: HTTP server + route registration
- **[config.go](config.go)**: .env configuration
- **[sse.go](sse.go)**: SSE broker for real-time status
- **[media.go](media.go)**: Windows SMTC API integration
- **[actions.go](actions.go)**: Action dispatch (play/pause, mute, sleep, etc.)
- **[powershell.go](powershell.go)**: PowerShell execution helpers

## Quick Start
```bash
# One-time setup (on Linux build machine):
./build-all.sh bootstrap   # toolchain → SDK → GTK → Go binary

# Deploy after each code change:
./install.sh               # builds + SCPs to Kindle
```

## Important Rules
1. **Windows daemon stays unchanged** — same files as `main` branch
2. **Build pipeline uses koxtoolchain** — pre-built KHF toolchain
3. **Direct sysfs access** — no settings-server.sh needed
4. **No Chromium** — native GTK window, ~9MB RSS
