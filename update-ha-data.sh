#!/usr/bin/env bash
set -euo pipefail

# Pull Home Assistant media_player state and publish it to the Kindle dashboard.
# Required:
#   export HA_URL="http://homeassistant.local:8123"   # or https://your-ha-host
#   export HA_TOKEN="<long-lived access token>"
# Optional:
#   export HA_ENTITY="media_player.googlehome1844"
#   export WATCH_SECONDS=60   # loop; 0 = run once

HA_URL="${HA_URL:-}"
HA_TOKEN="${HA_TOKEN:-}"
HA_ENTITY="${HA_ENTITY:-media_player.googlehome1844}"
WATCH_SECONDS="${WATCH_SECONDS:-0}"
SSH_TARGET="${SSH_TARGET:-root@192.168.1.91}"
SSH_PORT="${SSH_PORT:-2222}"
REMOTE_DIR="${REMOTE_DIR:-/mnt/us/documents/kindle-dashboard}"
OUT_FILE="${OUT_FILE:-data.js}"
STATE_CACHE="${STATE_CACHE:-.ha-data.last}"

if [[ -z "$HA_URL" || -z "$HA_TOKEN" ]]; then
  cat >&2 <<'EOF'
Missing HA_URL or HA_TOKEN.

Example:
  export HA_URL="http://homeassistant.local:8123"
  export HA_TOKEN="<Home Assistant long-lived access token>"
  ./update-ha-data.sh

Create token in Home Assistant:
  Profile → Security → Long-lived access tokens → Create Token
EOF
  exit 2
fi

HA_URL="${HA_URL%/}"

fetch_once() {
  local tmp json signature
  tmp="$(mktemp)"

  json="$(curl -fsS \
    -H "Authorization: Bearer $HA_TOKEN" \
    -H "Content-Type: application/json" \
    "$HA_URL/api/states/$HA_ENTITY")"

  python3 - "$HA_ENTITY" >"$tmp" <<'PY' <<<"$json"
import json, sys

entity = sys.argv[1]
state = json.load(sys.stdin)
attrs = state.get("attributes") or {}
player_state = state.get("state") or "unknown"


def js_string(value):
    return json.dumps("" if value is None else str(value), ensure_ascii=False)


def fmt_time(value):
    if value in (None, "", "unknown", "unavailable"):
        return ""
    try:
        seconds = int(float(value))
    except Exception:
        return ""
    return f"{seconds // 60}:{seconds % 60:02d}"


def badge_for(s):
    s = (s or "").lower()
    if s == "playing":
        return "PLAY"
    if s == "paused":
        return "PAUSE"
    if s in ("off", "standby"):
        return "OFF"
    return "IDLE"

track = attrs.get("media_title") or attrs.get("friendly_name") or entity
artist = attrs.get("media_artist") or attrs.get("media_album_artist") or ""
album = attrs.get("media_album_name") or ""
app = attrs.get("app_name") or attrs.get("source") or "Home Assistant"
device = attrs.get("friendly_name") or entity
position = fmt_time(attrs.get("media_position"))
duration = fmt_time(attrs.get("media_duration"))
last = state.get("last_updated") or state.get("last_changed") or ""
badge = badge_for(player_state)
summary_parts = [device, app, player_state]
summary = " • ".join([str(x) for x in summary_parts if x])

print("window.KINDLE_DASHBOARD_DATA = {")
print(f"  status: {js_string('Home Assistant • ' + device)},")
print("  music: {")
print(f"    badge: {js_string(badge)},")
print(f"    summary: {js_string(summary)},")
print(f"    state: {js_string(player_state)},")
print(f"    track: {js_string(track)},")
print(f"    artist: {js_string(artist)},")
print(f"    album: {js_string(album)},")
print(f"    position: {js_string(position)},")
print(f"    duration: {js_string(duration)},")
print(f"    source: {js_string(app)},")
print("    items: [")
items = [
    ("Device", device),
    ("Player", app),
    ("State", player_state),
    ("Updated", last.replace("T", " ").replace("+00:00", "Z")[:19] if last else ""),
]
for i, (label, value) in enumerate(items):
    comma = "," if i < len(items) - 1 else ""
    print(f"      {{ label: {js_string(label)}, value: {js_string(value)} }}{comma}")
print("    ]")
print("  }")
print("};")
PY

  signature="$(sha256sum "$tmp" | awk '{print $1}')"
  if [[ -f "$STATE_CACHE" && "$(cat "$STATE_CACHE")" == "$signature" ]]; then
    echo "no change: $HA_ENTITY"
    rm -f "$tmp"
    return 0
  fi

  cp "$tmp" "$OUT_FILE"
  scp -P"$SSH_PORT" "$OUT_FILE" "$SSH_TARGET:$REMOTE_DIR/data.js" >/dev/null
  echo "$signature" >"$STATE_CACHE"
  rm -f "$tmp"
  echo "updated Kindle data.js from $HA_ENTITY"
}

if [[ "$WATCH_SECONDS" =~ ^[0-9]+$ ]] && (( WATCH_SECONDS > 0 )); then
  while true; do
    fetch_once || true
    sleep "$WATCH_SECONDS"
  done
else
  fetch_once
fi
