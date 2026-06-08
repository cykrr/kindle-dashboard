# Agent and Developer Guide

Welcome! This guide explains how this repository is structured, how to navigate the files, how to develop locally, and how to deploy updates to the Kindle dashboard.

---

## 📂 Repository Structure

The project has been modularized into separate, single-responsibility files for cleaner code and easier iteration:

### Kindle Dashboard (frontend — runs in Chromium on the Kindle)
- **[index.html](file:///home/krr/pw/index.html)**: Semantic HTML structure of the dashboard layout (Header with Clock/Calendar, Devices, Mail, Agenda, and Music widgets, plus Settings drawer).
- **[index.css](file:///home/krr/pw/index.css)**: All styling, responsive queries, and theming definitions (e-ink friendly high-contrast palettes).
- **[app.js](file:///home/krr/pw/app.js)**: Interactive logic — clock/calendar rendering, gesture handling, Home Assistant WebSocket connection, on-device brightness/rotation API polling, fallbacks.
- **[hass-config.js](file:///home/krr/pw/hass-config.js)** (generated): Injected config values with HASS credentials and endpoints.
- **[data.js](file:///home/krr/pw/data.js)**: Fallback mock data when HASS or on-device APIs are disconnected.

### Windows Macro Daemon (Go backend — runs on Windows)
The daemon provides an HTTP API (`:8080`) that the Kindle dashboard queries for system status and actions. It is modularized into single-responsibility files:

| File | Responsibility |
|---|---|
| **[main.go](file:///home/krr/pw/main.go)** | Entry point, HTTP server, route registration |
| **[config.go](file:///home/krr/pw/config.go)** | Configuration loaded from `.env` |
| **[sse.go](file:///home/krr/pw/sse.go)** | SSE (Server-Sent Events) broker for real-time status streaming |
| **[media.go](file:///home/krr/pw/media.go)** | Windows SMTC API integration — detects currently playing media (title, artist, playback status) via WinRT COM interop |
| **[actions.go](file:///home/krr/pw/actions.go)** | Action dispatch — play/pause, mute mic, sleep, screenshot, etc. |
| **[powershell.go](file:///home/krr/pw/powershell.go)** | PowerShell execution helpers (hidden window, optional elevation via gsudo) with CLIXML error decoding |
| **[go.mod](file:///home/krr/pw/go.mod)** | Go module definition |

### Kindle On-Device API & Control Scripts
- **[settings-server.sh](file:///home/krr/pw/settings-server.sh)** / **[settings-api.sh](file:///home/krr/pw/settings-api.sh)**: Simple HTTP server on the Kindle (`http://127.0.0.1:8177`) to read/write Kindle system settings (brightness, battery, orientation).
- **[launch.sh](file:///home/krr/pw/launch.sh)**: Main orchestrator — stops Kindle UI (`lab126_gui`), runs `settings-server.sh`, starts Chromium in kiosk mode pointing to `index.html`.
- **[stop.sh](file:///home/krr/pw/stop.sh)**: Stops Chromium and the settings API, returning the Kindle to its standard interface.

---

## 🚀 Iteration & Deployment

### 1. Local Development

**Web UI**: Open `index.html` in any browser. Kindle API calls will gracefully fall back to showing `unavailable`.

**Macro Daemon**: Build and run on Windows:
```bash
# From WSL:
GOOS=windows GOARCH=amd64 go build -o macro-daemon.exe .
# Then copy to Windows and run:
cp macro-daemon.exe /mnt/c/Users/krr/temp-macro.exe
pwsh.exe -Command "Start-Process -FilePath C:\KindleDashboard\macro-daemon.exe"
```

### 2. Config Setup

Configuration for both HASS and the macro daemon comes from `.env`:

| Variable | Purpose |
|---|---|
| `HASS_URL` / `HASS_TOKEN` | Home Assistant connection |
| `HASS_BRIGHTNESS_ENTITY` | Optional: entity for two-way brightness sync |
| `MACRO_API_KEY` | Shared secret for daemon API auth |
| `MACRO_PORT` | HTTP listen port (default `8080`) |
| `MACRO_GSUDO` | Path to `gsudo.exe` for privilege elevation |
| `MACRO_LOG_PATH` | Log file path on Windows |

```bash
# Compile HASS config for Kindle:
./publish-hass-config.sh
```

### 3. Deploying Code Changes

**Kindle dashboard files** (HTML/CSS/JS, scripts):
```bash
./install.sh
```
Copies to `/mnt/us/documents/kindle-dashboard` on the Kindle via SSH/SCP.

**Macro daemon** (Go binary):
```bash
GOOS=windows GOARCH=amd64 go build -o macro-daemon.exe .
cp macro-daemon.exe /mnt/c/Users/krr/temp-macro.exe
pwsh.exe -Command "taskkill /F /IM macro-daemon.exe /T 2>null; Start-Sleep -Milliseconds 500; Copy-Item -Path C:\Users\krr\temp-macro.exe -Destination C:\KindleDashboard\macro-daemon.exe -Force; Start-Process -FilePath C:\KindleDashboard\macro-daemon.exe"
```

---

## ⚠️ Important Implementation Rules for Agents

1. **Keep CSS/JS External**: Do not bundle styles or scripts inline inside [index.html](file:///home/krr/pw/index.html). Keep HTML, CSS, and JS separate.
2. **ES5/Old Chromium Compatibility**: The Kindle browser runs an older Chromium. Avoid modern syntax (ES Modules, optional chaining, etc.). Use broadly compatible Javascript.
3. **Keep Deployment Scripts Synced**: If you add new assets, update the `scp` line in [install.sh](file:///home/krr/pw/install.sh).
4. **Go Files Are Modular**: Keep `main.go`, `config.go`, `sse.go`, `media.go`, `actions.go`, `powershell.go` as single-responsibility files. Do not merge them back into a monolith.
5. **Cross-Compile Reminder**: The daemon targets Windows. Always build with `GOOS=windows GOARCH=amd64`. The `syscall.SysProcAttr` fields (`HideWindow`, `CreationFlags`) are Windows-only and will not compile on Linux — that's expected.
