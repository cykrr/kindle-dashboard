#!/bin/bash
set -e
REMOTE_TARGET="root@192.168.1.91"
REMOTE_PORT="2222"
REMOTE_DIR="/mnt/us/documents/kindle-dashboard"

echo "Installing to Kindle..."
ssh -p "$REMOTE_PORT" "$REMOTE_TARGET" "mkdir -p $REMOTE_DIR"
scp -P "$REMOTE_PORT" index.html launch.sh stop.sh hass-config.js menu.json "$REMOTE_TARGET:$REMOTE_DIR"
echo "Done!"
