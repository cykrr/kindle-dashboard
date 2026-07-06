#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "=== Building Windows Macro Daemon ==="
echo "Target: GOOS=windows GOARCH=amd64"

GOOS=windows GOARCH=amd64 go build -o ../../macro-daemon.exe -ldflags "-H windowsgui" .

echo "✓ Build complete!"
echo "Binary located at: $(cd ../../ && pwd)/macro-daemon.exe"
