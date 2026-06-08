# Kindle Dashboard

![Kindle Dashboard Screenshot](screenshot.png)


A fast, simple, and clean HTML/JS dashboard specifically designed for jailbroken e-ink Kindles. Features integration with Home Assistant via Long-Lived Access Tokens and WebSockets.

## Features
- E-ink optimized UI (high contrast, no animations)
- Displays Mail count
- Music currently playing
- Calendar/Agenda
- Two-way Kindle brightness synchronization with Home Assistant
- Can run natively via KUAL

## Setup
1. Create a `.env` file containing your Home Assistant URL, token, and entities (e.g. `HASS_URL`, `HASS_TOKEN`, `HASS_BRIGHTNESS_ENTITY`, etc).
2. Run `./publish-hass-config.sh` to compile your `.env` into `hass-config.js` and upload to your Kindle.
3. Deploy the application files using `./install.sh`.
4. Launch using `launch.sh` (or via KUAL).

## Files
- `index.html`: The main dashboard UI.
- `launch.sh`: Script to launch the browser in true fullscreen mode.
- `stop.sh`: Script to cleanly kill the dashboard and restore the Kindle GUI.
- `menu.json`: KUAL menu configuration.
