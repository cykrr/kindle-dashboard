# Agent and Developer Guide (Native Branch)

Welcome! This branch contains the **native GTK+ 2.24** version of the Kindle Dashboard.

## Repository Structure

```
cmd/
├── daemon/          # Windows Macro Daemon (Go HTTP server)
│   ├── main.go, config.go, sse.go, media.go, actions.go, powershell.go
│   └── go.mod, go.sum
└── dashboard/       # Kindle GTK Dashboard (Go + cgo GTK2)
    ├── main.go, app.go, hass.go, hass_config.go, pc.go, sysfs.go
    └── go.mod, go.sum

scripts/
├── build/           # Cross-compilation pipeline
│   ├── build-all.sh, build-gtk.sh, build-test.sh, setup-env.sh
├── kindle/          # Kindle deployment & control
│   ├── install.sh, launch.sh, stop.sh, pw.sh
└── windows/         # Windows daemon installer
    └── install-startup.ps1

legacy/
├── web/             # Old Chromium-based web dashboard (archived)
└── scripts/         # Old settings server & HA update scripts

docs/
├── version.txt
└── images/          # Screenshots & design mockups

README.md            # This file
AGENTS.md            # This file

deploy/              # Build output (gitignored)
.env                 # Configuration (gitignored)
hass-config.js       # Generated HA config (gitignored)
```

### Kindle Dashboard (native binary — runs on Kindle)
- **[cmd/dashboard/main.go](cmd/dashboard/main.go)**: Go GTK2 dashboard — clock, calendar, brightness, battery, HASS, PC integration
- **[scripts/kindle/launch.sh](scripts/kindle/launch.sh)**: Stops Kindle GUI, starts native binary
- **[scripts/kindle/stop.sh](scripts/kindle/stop.sh)**: Stops native binary, restores Kindle GUI
- **[scripts/kindle/install.sh](scripts/kindle/install.sh)**: Cross-compiles and deploys to Kindle

### Build Pipeline
| Script | Purpose |
|--------|---------|
| **[scripts/build/setup-env.sh](scripts/build/setup-env.sh)** | Cross-compilation environment (KHF toolchain) |
| **[scripts/build/build-all.sh](scripts/build/build-all.sh)** | Orchestrator: toolchain → SDK → GTK → Go test |
| **[scripts/build/build-gtk.sh](scripts/build/build-gtk.sh)** | Cross-compiles GTK 2.24.33 + cairo with X11 |
| **[scripts/build/build-test.sh](scripts/build/build-test.sh)** | Cross-compiles the Go binary → `deploy/dashboard-native` |

### Windows Macro Daemon (Go backend — runs on Windows)
- **[cmd/daemon/main.go](cmd/daemon/main.go)**: HTTP server + route registration
- **[cmd/daemon/config.go](cmd/daemon/config.go)**: .env configuration
- **[cmd/daemon/sse.go](cmd/daemon/sse.go)**: SSE broker for real-time status
- **[cmd/daemon/media.go](cmd/daemon/media.go)**: Windows SMTC API integration
- **[cmd/daemon/actions.go](cmd/daemon/actions.go)**: Action dispatch (play/pause, mute, sleep, etc.)
- **[cmd/daemon/powershell.go](cmd/daemon/powershell.go)**: PowerShell execution helpers

## Quick Start
```bash
# One-time setup (on Linux build machine):
./scripts/build/build-all.sh bootstrap   # toolchain → SDK → GTK → Go binary

# Deploy after each code change:
./scripts/kindle/install.sh             # builds + SCPs to Kindle
```

## Important Rules
1. **Windows daemon stays unchanged** — same files as `main` branch
2. **Build pipeline uses koxtoolchain** — pre-built KHF toolchain
3. **Direct sysfs access** — no settings-server.sh needed
4. **No Chromium** — native GTK window, ~9MB RSS
