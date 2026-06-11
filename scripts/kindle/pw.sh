#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
if [[ -f "$ROOT_DIR/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

KINDLE_PORT="${KINDLE_PORT:-2222}"
if [[ -z "${KINDLE_IP:-}" ]]; then
  echo "Set KINDLE_IP in .env or the environment" >&2
  exit 2
fi

ssh -p "$KINDLE_PORT" "root@$KINDLE_IP" "$@"
