#!/bin/bash
# Cross-compile the Go GTK test for Kindle KHF
# Links against Kindle's system GTK 2.10.0 from the firmware sysroot

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/setup-env.sh"

SRC_DIR="${SCRIPT_DIR}/src"
DEPLOY_DIR="${SCRIPT_DIR}/deploy"
mkdir -p "${DEPLOY_DIR}"

SYSROOT_INC="${SYSROOT}/usr/include"

echo "=== Building Go GTK dashboard for Kindle KHF ==="

# Export cgo cross-compilation environment
export CC="${CROSS_TC}-gcc"
export GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1

# Point cgo to the Kindle's sysroot headers
export PKG_CONFIG_LIBDIR="${TC_BUILD_DIR}/lib/pkgconfig:${SYSROOT}/usr/lib/pkgconfig"
export PKG_CONFIG_SYSROOT_DIR="${SYSROOT}"

# cgo needs to find GTK headers in the sysroot
export CPATH="${SYSROOT_INC}:${SYSROOT_INC}/gtk-2.0:${SYSROOT_INC}/glib-2.0:${SYSROOT_INC}/pango-1.0:${SYSROOT_INC}/cairo:${SYSROOT_INC}/atk-1.0:${SYSROOT_INC}/gdk-pixbuf-2.0:${SYSROOT_INC}/pixman-1:${SYSROOT_INC}/freetype2:${SYSROOT_INC}/libpng16"
GLIB_CFG="${SYSROOT}/usr/lib/glib-2.0/include"
export CGO_CFLAGS="-I${SYSROOT_INC} -I${SYSROOT_INC}/gtk-2.0 -I${SYSROOT_INC}/glib-2.0 -I${SYSROOT_INC}/pango-1.0 -I${SYSROOT_INC}/cairo -I${SYSROOT_INC}/atk-1.0 -I${SYSROOT_INC}/gdk-pixbuf-2.0 -I${SYSROOT_INC}/pixman-1 -I${SYSROOT_INC}/freetype2 -I${SYSROOT_INC}/libpng16 -I${GLIB_CFG} -I${SYSROOT}/usr/lib/gtk-2.0/include"
export CGO_LDFLAGS="-L${SYSROOT}/usr/lib -L${SYSROOT}/lib -L${TC_BUILD_DIR}/lib -Wl,-rpath-link,${SYSROOT}/usr/lib -Wl,-rpath-link,${SYSROOT}/lib -lgtk-x11-2.0 -lgdk-x11-2.0 -lgdk_pixbuf-2.0 -lpangocairo-1.0 -lpango-1.0 -lcairo -latk-1.0 -lgio-2.0 -lgobject-2.0 -lglib-2.0 -lX11 -lXext -lXrender"

cd "${SRC_DIR}"
echo "Cross-compiling with: ${CC}"
echo "CPATH: ${CPATH}"
echo ""

go build -v \
    -ldflags="-s -w" \
    -o "${DEPLOY_DIR}/dashboard-native" \
    . 2>&1

echo ""
echo "=== Build complete! ==="
echo "Binary: ${DEPLOY_DIR}/dashboard-native"
file "${DEPLOY_DIR}/dashboard-native" 2>/dev/null || true
ls -lh "${DEPLOY_DIR}/dashboard-native"
echo ""
echo "To deploy:"
echo "scp -P2222 ${DEPLOY_DIR}/dashboard-native root@192.168.1.91:/mnt/us/documents/kindle-dashboard/"
echo ""
echo "On Kindle, stop the current dashboard (launch.sh) and run:"
echo "DISPLAY=:0 /mnt/us/documents/kindle-dashboard/dashboard-native"
