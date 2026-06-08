#!/bin/bash
set -e
REMOTE_TARGET="root@192.168.1.91"
REMOTE_PORT="2222"
REMOTE_DIR="/mnt/us/documents/kindle-dashboard"

echo "Installing to Kindle..."
date +%s > version.txt
ssh -p "$REMOTE_PORT" "$REMOTE_TARGET" "mkdir -p $REMOTE_DIR"
scp -P "$REMOTE_PORT" index.html index.css state.js utils.js time.js ui.js api.js hass.js app.js launch.sh stop.sh hass-config.js menu.json settings-api.sh settings-server.sh version.txt "$REMOTE_TARGET:$REMOTE_DIR"
ssh -p "$REMOTE_PORT" "$REMOTE_TARGET" "chmod +x $REMOTE_DIR/launch.sh $REMOTE_DIR/stop.sh $REMOTE_DIR/settings-api.sh $REMOTE_DIR/settings-server.sh"
echo "Done!"
