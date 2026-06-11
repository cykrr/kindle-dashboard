# TODO

## Before public release

1. Documentation debt
   - [x] Add `dashboard-config.example.json` / `hass-config.example.js` with supported public fields.
   - [x] Update README to match current implementation: native GTK, HA REST polling, PC macro SSE/HTTP.
   - [x] Document Kindle prerequisites, jailbreak assumptions, deployment flow, logs, and recovery/stop scripts.
   - [x] Document personalization: launcher buttons, Home Assistant entities, PC macro URL/key, brightness entity, suspend-cycle behavior.
   - [x] Remove remaining personal deployment defaults such as Kindle IPs and SSH env vars from helper scripts.

2. Debug flag / logging cleanup
   - [x] Add a `-debug` flag or config option.
   - [x] Gate noisy logs such as per-clock-redraw timing behind debug mode.
   - [x] Keep important operational logs: launch, suspend/wake, early wake, errors.

3. Button confirmation support
   - [x] Add `needsConfirmation` / `needs_confirmation` to launcher button config.
   - [x] Use confirmation for destructive actions such as `shutdown`, `restart`, and `sleep`.
   - [x] Use second-tap confirmation with a temporary status message.
   - [x] Make public default launcher safe by requiring confirmation for destructive actions.
