#!/bin/bash
set -e

cd "$(dirname "$0")"/../..
source ./.env

# Wait for Kindle to be reachable via SSH before deploying.
# Kindle doesn't respond to ping (ICMP firewalled), so probe the SSH port instead.
# Tap the Kindle's power button to wake it if it's asleep.
MAX_ATTEMPTS=30
attempt=1
echo "Waiting for Kindle to be reachable (tap power button to wake it)..."
while ! nc -zw 3 ${KINDLE_IP} ${KINDLE_PORT} 2>/dev/null; do
  if [ $attempt -ge $MAX_ATTEMPTS ]; then
    echo "Kindle not reachable after ${MAX_ATTEMPTS} attempts. Aborting."
    exit 1
  fi
  echo "  retrying... ($attempt/$MAX_ATTEMPTS)"
  sleep 2
  attempt=$((attempt + 1))
done
echo "Kindle is awake."

echo "=== Deploying native dashboard to Kindle ==="

# Build the Go binary first
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

# Deploy KUAL extension
echo "Copying KUAL extension..."
KUAL_EXTENSIONS_DIR="/mnt/us/extensions"
ssh -p ${KINDLE_PORT} root@${KINDLE_IP} "mkdir -p ${KUAL_EXTENSIONS_DIR}" || true
scp -P${KINDLE_PORT} -r kindle/kual/Kindle-Dashboard root@${KINDLE_IP}:${KUAL_EXTENSIONS_DIR}/

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
