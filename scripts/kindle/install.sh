#!/bin/bash
set -e


echo "=== Deploying native dashboard to Kindle ==="

# Build the Go binary first
cd "$(dirname "$0")"/../..
source ./.env
./scripts/build/build-test.sh

# Stop existing process to avoid busy file
echo "Stopping any running dashboard..."
ssh -p ${KINDLE_PORT} root@${KINDLE_IP} "killall -9 dashboard-native 2>/dev/null || true" || true

# Deploy binary
echo "Copying dashboard-native..."
scp -P${KINDLE_PORT} deploy/dashboard-native root@${KINDLE_IP}:${DASHBOARD_DIR}/

# Deploy launch script
echo "Copying launch.sh..."
scp -P${KINDLE_PORT} scripts/kindle/launch.sh root@${KINDLE_IP}:${DASHBOARD_DIR}/

# Deploy Home Assistant config if present
echo "Copying hass-config.js..."
if [ -f hass-config.js ]; then
  scp -P${KINDLE_PORT} hass-config.js root@${KINDLE_IP}:${DASHBOARD_DIR}/
else
  echo "hass-config.js not found; set HASS_URL/HASS_TOKEN or copy config manually"
fi

echo ""
echo "=== Deploy complete ==="
echo "Restarting dashboard via launch.sh (keeps device awake)..."
ssh -p ${KINDLE_PORT} root@${KINDLE_IP} "nohup ${DASHBOARD_DIR}/launch.sh > /tmp/launch.log 2>&1 &"
echo "Dashboard restarted."
