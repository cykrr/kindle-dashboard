#!/bin/bash
# Source this to set up the cross-compilation environment for Kindle KHF
# Usage: source setup-env.sh [bare]
#   bare - only set PATH, no build-specific flags

# === Toolchain paths ===
export CROSS_TC="arm-kindlehf-linux-gnueabihf"
export TC_BUILD_DIR="${HOME}/Kindle/CrossTool/Build_KHF"
export SYSROOT="${HOME}/x-tools/${CROSS_TC}/${CROSS_TC}/sysroot"
export CROSS_PREFIX="${CROSS_TC}-"

# Add toolchain to PATH
export PATH="${HOME}/x-tools/${CROSS_TC}/bin:${PATH}"

# === Architecture flags ===
export ARCH_FLAGS="-march=armv7-a -mtune=cortex-a7 -mfpu=neon -mfloat-abi=hard -mthumb"
export BASE_CFLAGS="-O2 ${ARCH_FLAGS} -pipe -fomit-frame-pointer"
export BASE_CXXFLAGS="${BASE_CFLAGS}"
export BASE_LDFLAGS="-L${TC_BUILD_DIR}/lib -Wl,-O1 -Wl,--as-needed"

# === Tool aliases ===
export CC="${CROSS_TC}-gcc"
export CXX="${CROSS_TC}-g++"
export AR="${CROSS_TC}-ar"
export RANLIB="${CROSS_TC}-ranlib"
export STRIP="${CROSS_TC}-strip"
export LD="${CROSS_TC}-ld"
export NM="${CROSS_TC}-nm"

# === pkg-config setup ===
export PKG_CONFIG="pkg-config"
export PKG_CONFIG_PATH=""
export PKG_CONFIG_LIBDIR="${TC_BUILD_DIR}/lib/pkgconfig"
export PKG_CONFIG_SYSROOT_DIR="${SYSROOT}"

# === Go cross-compile ===
export GOOS="linux"
export GOARCH="arm"
export GOARM="7"
export CGO_ENABLED="1"
export CC_FOR_TARGET="${CC}"

# Print status
echo "=== Kindle KHF Cross-Compilation Environment ==="
echo "CROSS_TC:       ${CROSS_TC}"
echo "Toolchain:      ${HOME}/x-tools/${CROSS_TC}"
echo "Sysroot:        ${SYSROOT}"
echo "TC_BUILD_DIR:   ${TC_BUILD_DIR}"
echo "CC:             $(which ${CC} 2>/dev/null || echo 'NOT FOUND')"
echo "Go target:      ${GOOS}/${GOARCH} (ARM ${GOARM})"
echo "pkg-config:     ${PKG_CONFIG_LIBDIR}"
echo ""
echo "Verify with: ${CC} --version"
