#!/bin/bash
set -e

DASHBOARD_DIR="/mnt/us/documents/kindle-dashboard"
KINDLE_IP="192.168.1.91"
KINDLE_PORT="2222"

echo "=== Deploying native dashboard to Kindle ==="

# Build the Go binary first
cd "$(dirname "$0")"
./build-test.sh

# Deploy binary
echo "Copying dashboard-native..."
scp -P${KINDLE_PORT} deploy/dashboard-native root@${KINDLE_IP}:${DASHBOARD_DIR}/

# Deploy launch script
echo "Copying launch.sh..."
scp -P${KINDLE_PORT} launch.sh root@${KINDLE_IP}:${DASHBOARD_DIR}/

echo ""
echo "=== Deploy complete ==="
echo "SSH into Kindle and run:"
echo "  sh ${DASHBOARD_DIR}/launch.sh"
