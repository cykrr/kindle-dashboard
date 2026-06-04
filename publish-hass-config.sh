#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${ENV_FILE:-.env}"
SSH_TARGET="${SSH_TARGET:-root@192.168.1.91}"
SSH_PORT="${SSH_PORT:-2222}"
REMOTE_DIR="${REMOTE_DIR:-/mnt/us/documents/kindle-dashboard}"
HASS_ENTITY="${HASS_ENTITY:-media_player.googlehome1844}"
HASS_MAIL_ENTITY="${HASS_MAIL_ENTITY:-sensor.imap_me_messages}"
HASS_MAIL_LABEL="${HASS_MAIL_LABEL:-Mail}"
HASS_CALENDAR_ENTITIES="${HASS_CALENDAR_ENTITIES:-calendar.it,calendar.calendario}"
OUT_FILE="${OUT_FILE:-hass-config.js}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE" >&2
  exit 2
fi

python3 - "$ENV_FILE" "$HASS_ENTITY" "$HASS_MAIL_ENTITY" "$HASS_CALENDAR_ENTITIES" "$OUT_FILE" <<'PY'
import json, shlex, sys
from pathlib import Path

env_path = Path(sys.argv[1])
entity = sys.argv[2]
mail_entity = sys.argv[3]
calendar_entities = sys.argv[4]
out_path = Path(sys.argv[5])
values = {}

for raw in env_path.read_text().splitlines():
    line = raw.strip()
    if not line or line.startswith('#') or '=' not in line:
        continue
    key, value = line.split('=', 1)
    key = key.strip()
    value = value.strip()
    try:
        value = shlex.split(value)[0] if value else ''
    except Exception:
        value = value.strip('"\'')
    values[key] = value

url = values.get('HASS_URL') or values.get('HA_URL')
token = values.get('HASS_TOKEN') or values.get('HA_TOKEN')
entity = values.get('HASS_ENTITY') or values.get('HA_ENTITY') or entity
mail_entity = values.get('HASS_MAIL_ENTITY') or values.get('HA_MAIL_ENTITY') or mail_entity
mail_label = values.get('HASS_MAIL_LABEL') or values.get('HA_MAIL_LABEL') or 'Mail'
calendar_entities = values.get('HASS_CALENDAR_ENTITIES') or values.get('HA_CALENDAR_ENTITIES') or calendar_entities
calendar_entities = [x.strip() for x in calendar_entities.split(',') if x.strip()]

if not url or not token:
    print('Missing HASS_URL/HASS_TOKEN in .env', file=sys.stderr)
    sys.exit(2)

config = {
    'url': url.rstrip('/'),
    'token': token,
    'entity': entity,
    'musicEntity': entity,
    'mailEntity': mail_entity,
    'mailLabel': mail_label,
    'calendarEntities': calendar_entities,
}
out_path.write_text('window.HASS_CONFIG = ' + json.dumps(config, ensure_ascii=False, indent=2) + ';\n')
print(f'wrote {out_path} for music={entity} mail={mail_entity} calendars={",".join(calendar_entities)} at {url.rstrip("/")}')
PY

scp -P"$SSH_PORT" "$OUT_FILE" "$SSH_TARGET:$REMOTE_DIR/$OUT_FILE" >/dev/null
echo "uploaded $OUT_FILE to Kindle"
