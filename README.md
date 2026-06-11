# Kindle Dashboard (Native GTK)

A native GTK dashboard for jailbroken e-ink Kindles. It replaces the old Chromium dashboard with a small Go/cgo GTK app focused on low memory use and low power draw.

> Status: experimental / personal-alpha. Tested on the author's Kindle setup only. Suspend, frontlight, and sysfs paths may vary by Kindle model.

## Features

- E-ink optimized high-contrast UI
- Clock, date, mini calendar, battery, and frontlight brightness
- Optional Home Assistant REST polling for mail, calendar, lights, and brightness sync
- Optional Windows macro daemon integration for PC media status and macro buttons
- Experimental RTC suspend-cycle mode that wakes near minute boundaries to refresh the clock

## Architecture

- **Kindle dashboard**: Go + cgo + GTK (`cmd/dashboard/`)
- **Windows daemon**: Go HTTP/SSE service (`cmd/daemon/`) for media status and macro actions
- **Home Assistant**: REST API polling, configured by `hass-config.js` / `hass-config.json`

## Build and deploy

```bash
# One-time build machine setup
./scripts/build/build-all.sh bootstrap

# Build and deploy to Kindle using .env values
./scripts/kindle/install.sh
```

Expected `.env` deployment values:

```sh
KINDLE_IP=192.168.x.x
KINDLE_PORT=2222
DASHBOARD_DIR=/mnt/us/documents/kindle-dashboard
HASS_URL=http://homeassistant.local:8123
HASS_TOKEN=your_long_lived_access_token
PC_MACRO_URL=http://your-pc-ip:8765
PC_MACRO_KEY=your_macro_key
```

Useful Kindle-side commands:

```sh
/mnt/us/documents/kindle-dashboard/launch.sh     # start dashboard
/mnt/us/documents/kindle-dashboard/stop.sh       # stop dashboard and restore Kindle UI
tail -f /tmp/dashboard-native.log                # dashboard logs
```

Add `-debug` to the dashboard command line for verbose UI timing logs.

## Configuration

The dashboard loads config from, in order:

1. `$HASS_CONFIG`
2. `hass-config.js` / `hass-config.json` next to the binary
3. current working directory variants
4. `/mnt/us/documents/kindle-dashboard/hass-config.js`

See:

- [`docs/dashboard-config.example.json`](docs/dashboard-config.example.json)
- [`docs/hass-config.example.js`](docs/hass-config.example.js)

Minimal config:

```json
{
  "url": "http://homeassistant.local:8123",
  "token": "YOUR_LONG_LIVED_ACCESS_TOKEN"
}
```

Home Assistant widgets are optional. If `musicEntity`, `mailEntity`, or `calendarEntities` are omitted, those HA-backed widgets are disabled/empty instead of defaulting to the author's personal entities.

### Launcher buttons

Launcher buttons are configurable:

```json
{
  "launcherButtons": [
    { "action": "mute_mic", "icon": "mute_mic", "label": "Mic" },
    { "action": "launch_chrome", "icon": "launch_chrome", "label": "Browser" },
    { "action": "sleep", "icon": "sleep", "label": "Sleep" }
  ]
}
```

- `action`: command sent to the Windows macro daemon
- `icon`: icon name drawn by the dashboard; defaults to `action`
- `label`: reserved/documentary today; launcher buttons are currently icon-only

If `launcherButtons` is omitted, the current built-in default launcher is used. Set `launcherButtons: []` for an empty launcher.

Known built-in daemon actions include: `mute_mic`, `play_pause`, `prev_track`, `next_track`, `sleep`, `power_mode`, `save_mode`, `pc_mode_toggle`, `monitor_toggle`, `launch_chrome`, `launch_mail`, `launch_fortnite`, `restart`, `shutdown`.

## Suspend-cycle mode

`scripts/kindle/launch.sh` starts the dashboard with:

```sh
-hw-landscape -suspend-cycle
```

Suspend-cycle mode dims the frontlight, suspends to RAM, wakes shortly before the next minute, redraws the clock around `:00`, then polls optional network integrations after a grace period. Manual/power-button wakes restore the previous frontlight brightness and keep the device awake briefly.

This mode is experimental and hardware-dependent.

## Repository layout

```text
cmd/daemon/        Windows Macro Daemon
cmd/dashboard/     Kindle GTK Dashboard
scripts/build/     Cross-compilation pipeline
scripts/kindle/    Kindle deployment/control scripts
scripts/windows/   Windows daemon installer
legacy/            Archived Chromium-based web dashboard
docs/              Documentation, examples, screenshots
deploy/            Build output (gitignored)
```
