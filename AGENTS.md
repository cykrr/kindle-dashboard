# Agent and Developer Guide

Welcome! This guide explains how this repository is structured, how to navigate the files, how to develop locally, and how to deploy updates to the Kindle dashboard.

---

## 📂 Repository Structure

The project has been modularized into separate, single-responsibility files for cleaner code and easier iteration:

- **[index.html](file:///home/krr/pw/index.html)**: Contains the semantic HTML structure of the dashboard layout (Header with Clock/Calendar, Devices, Mail, Agenda, and Music widgets, plus Settings drawer).
- **[index.css](file:///home/krr/pw/index.css)**: Holds all styling, responsive queries, and theming definitions (designed for e-ink friendly high-contrast palettes).
- **[app.js](file:///home/krr/pw/app.js)**: Handles all interactive logic including clock/calendar rendering, gestures for setting drawer dragging, Home Assistant WebSocket connection, on-device brightness/rotation API polling, and fallbacks.
- **[hass-config.js](file:///home/krr/pw/hass-config.js)** (generated): Injected configuration values containing credentials and endpoints for Home Assistant.
- **[data.js](file:///home/krr/pw/data.js)**: Fallback mock data containing default stats when Home Assistant or on-device APIs are disconnected.

### Kindle On-Device API & Control Scripts
- **[settings-server.sh](file:///home/krr/pw/settings-server.sh)** / **[settings-api.sh](file:///home/krr/pw/settings-api.sh)**: Simple HTTP server running on the Kindle (`http://127.0.0.1:8177`) to read and write Kindle system settings (brightness, battery, orientation).
- **[launch.sh](file:///home/krr/pw/launch.sh)**: Main orchestrator run on the Kindle. Stops the Kindle UI (`lab126_gui`), runs `settings-server.sh`, and starts Chromium in kiosk/fullscreen mode pointing to `index.html`.
- **[stop.sh](file:///home/krr/pw/stop.sh)**: Cleans up the Chromium process and local settings API, returning the Kindle to its standard interface.

---

## 🚀 Iteration & Deployment

### 1. Local Development
Because the application consists of vanilla HTML, CSS, and JS, you can test UI elements and logic by simply opening [index.html](file:///home/krr/pw/index.html) in any browser.

> [!NOTE]
> When testing locally, calls to the Kindle settings API (`http://127.0.0.1:8177`) will fail. The app gracefully falls back to displaying orientation/battery as unavailable without crashing.

### 2. Config Setup
Configuration for Home Assistant is derived from `.env`. Ensure your `.env` contains the required keys (e.g., `HASS_URL` and `HASS_TOKEN`).
Run the helper script to compile and upload the HASS configuration:
```bash
./publish-hass-config.sh
```

### 3. Deploying Code Changes
Whenever you update code (HTML, CSS, or JS) or scripts, deploy them to the Kindle device by running:
```bash
./install.sh
```
This script automates copying [index.html](file:///home/krr/pw/index.html), [index.css](file:///home/krr/pw/index.css), [app.js](file:///home/krr/pw/app.js), and supporting files to the remote path `/mnt/us/documents/kindle-dashboard` via SSH/SCP.

---

## ⚠️ Important Implementation Rules for Agents

1. **Keep CSS/JS External**: Do not bundle styles or scripts inline inside [index.html](file:///home/krr/pw/index.html) again. Keep HTML, CSS, and JS separate.
2. **ES5/Old Chromium Compatibility**: The Kindle browser runs an older version of Chromium. Avoid using modern syntax features (like native ES Modules `<script type="module">` or other experimental APIs) that might crash older browsers. Use standard, broadly compatible Javascript.
3. **Keep Deployment Scripts Synced**: If you add new assets (like images, external libraries, or additional files), you **must** update the `scp` line in [install.sh](file:///home/krr/pw/install.sh) to ensure they get deployed onto the Kindle.
