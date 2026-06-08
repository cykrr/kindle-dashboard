#!/bin/bash
# Full build pipeline for native GTK dashboard on Kindle
#
# Usage:
#   ./build-all.sh bootstrap   # Full build: toolchain → GTK → Go test
#   ./build-all.sh toolchain   # Step 1: Build koxtoolchain (30 min)
#   ./build-all.sh sdk         # Step 2: Install KMC SDK (needs toolchain)
#   ./build-all.sh gtk         # Step 3: Cross-compile GTK 2.24.33
#   ./build-all.sh test        # Step 4: Build Go hello-world test
#   ./build-all.sh dashboard   # Step 5: Build the full dashboard
#
# Prerequisites:
#   Arch: sudo pacman -S base-devel curl git gperf help2man unzip wget \
#                        meson bison flex texinfo gtk2-compat
#   Debian: sudo apt-get install build-essential autoconf automake bison flex \
#                        gawk libtool libtool-bin libncurses-dev curl file git \
#                        gperf help2man texinfo unzip wget meson libgtk2.0-dev

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Source env for cross-compilation variables (not flags, just paths)
source "${SCRIPT_DIR}/setup-env.sh" bare

KOOX_DIR="${HOME}/koxtoolchain"
SDK_DIR="${HOME}/kindle-sdk"
BUILD_DIR="${TC_BUILD_DIR}"
LOG_DIR="${SCRIPT_DIR}/logs"
mkdir -p "${LOG_DIR}"

echo "=============================================="
echo "Kindle Native Dashboard - Build Pipeline"
echo "Target:       ${CROSS_TC}"
echo "Build Root:   ${BUILD_DIR}"
echo "=============================================="

cmd_bootstrap() {
    cmd_toolchain
    cmd_sdk
    cmd_gtk
    cmd_test
    echo ""
    echo "=== Bootstrap complete! ==="
}

cmd_toolchain() {
    echo ""
    echo "=== Step 1: Build cross-compiler toolchain ==="
    if [ -f "${HOME}/x-tools/${CROSS_TC}/bin/${CROSS_TC}-gcc" ]; then
        echo "Toolchain already exists at ~/x-tools/${CROSS_TC}"
        return
    fi
    cd "${KOOX_DIR}"
    echo "Starting toolchain build for ${CROSS_TC} (takes ~30 min)..."
    ./gen-tc.sh kindlehf 2>&1 | tee "${LOG_DIR}/toolchain.log"
    echo "Toolchain build complete."
}

cmd_sdk() {
    echo ""
    echo "=== Step 2: Install KMC Kindle SDK ==="
    if [ -f "${HOME}/x-tools/${CROSS_TC}/meson-crosscompile.txt" ]; then
        echo "SDK appears to be installed (meson-crosscompile.txt found)"
        return
    fi
    cd "${SDK_DIR}"
    echo "Installing SDK for kindlehf..."
    sudo echo "Sudo authenticated for firmware mount operations."
    ./gen-sdk.sh kindlehf 2>&1 | tee "${LOG_DIR}/sdk.log"
    echo "SDK installation complete."
}

cmd_gtk() {
    echo ""
    echo "=== Step 3: Cross-compile GTK 2.24.33 ==="
    "${SCRIPT_DIR}/build-gtk.sh" 2>&1 | tee "${LOG_DIR}/gtk-build.log"
}

cmd_test() {
    echo ""
    echo "=== Step 4: Test Go cross-compilation ==="
    "${SCRIPT_DIR}/build-test.sh" 2>&1 | tee "${LOG_DIR}/test-build.log"
}

cmd_dashboard() {
    echo ""
    echo "=== Step 5: Build dashboard ==="
    "${SCRIPT_DIR}/build-dashboard.sh" 2>&1 | tee "${LOG_DIR}/dashboard-build.log"
}

# Dispatch
case "${1:-bootstrap}" in
    bootstrap) cmd_bootstrap ;;
    toolchain) cmd_toolchain ;;
    sdk)       cmd_sdk ;;
    gtk)       cmd_gtk ;;
    test)      cmd_test ;;
    dashboard) cmd_dashboard ;;
    *)
        echo "Usage: $0 {bootstrap|toolchain|sdk|gtk|test|dashboard}"
        exit 1
        ;;
esac
