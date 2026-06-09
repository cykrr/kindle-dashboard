#!/bin/bash
# Cross-compile GTK+ 2.24.33 for Kindle KHF with X11 backend.
# Builds cairo (with Xlib) first, then GTK.
#
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/setup-env.sh" bare

SOURCES_DIR="${SCRIPT_DIR}/sources"
LOG_DIR="${SCRIPT_DIR}/logs"
mkdir -p "${SOURCES_DIR}" "${LOG_DIR}"

# Versions
CAIRO_VERSION="1.16.0"
GTK_VERSION="2.24.33"
CAIRO_URL="https://www.cairographics.org/releases/cairo-${CAIRO_VERSION}.tar.xz"
GTK_URL="https://download.gnome.org/sources/gtk+/2.24/gtk+-${GTK_VERSION}.tar.xz"

# Download helper
download() {
    local url="$1" dest="$2"
    if [ ! -f "$dest" ]; then
        wget -q --show-progress "$url" -O "$dest"
    fi
}

# === Common cross-compile environment ===
export CC="${CROSS_TC}-gcc"
export CXX="${CROSS_TC}-g++"
export AR="${CROSS_TC}-ar"
export RANLIB="${CROSS_TC}-ranlib"
export STRIP="${CROSS_TC}-strip"
export LD="${CROSS_TC}-ld"
export NM="${CROSS_TC}-nm"
export PKG_CONFIG="${SCRIPT_DIR}/kindle-pkg-config"

# Create sysroot-aware pkg-config wrapper
cat > "${PKG_CONFIG}" << PKGEOF
#!/bin/bash
SYSROOT="${SYSROOT}"
exec pkg-config --define-variable=prefix=/usr \
    --define-variable=includedir=\${SYSROOT}/usr/include \
    --define-variable=libdir=\${SYSROOT}/usr/lib \
    "\$@"
PKGEOF
chmod +x "${PKG_CONFIG}"

export PKG_CONFIG_LIBDIR="${TC_BUILD_DIR}/lib/pkgconfig:${SYSROOT}/usr/lib/pkgconfig"
export PKG_CONFIG_SYSROOT_DIR="${SYSROOT}"
export CFLAGS="-I${SYSROOT}/usr/include -I${TC_BUILD_DIR}/include -O2 -pipe"
export CXXFLAGS="${CFLAGS}"
export LDFLAGS="-L${SYSROOT}/usr/lib -L${SYSROOT}/lib -L${TC_BUILD_DIR}/lib -Wl,-rpath-link,${SYSROOT}/usr/lib -Wl,-rpath-link,${SYSROOT}/lib"

echo "======================================"
echo "Building GTK stack for Kindle KHF"
echo "======================================"
echo "Host:       $(uname -m)-linux-gnu"
echo "Target:     ${CROSS_TC}"
echo "Sysroot:    ${SYSROOT}"
echo "Prefix:     ${TC_BUILD_DIR}"
echo ""

# === Step 1: Build cairo with Xlib backend ===
echo "--- Step 1: Cairo ${CAIRO_VERSION} (with Xlib) ---"
CAIRO_TARBALL="${SOURCES_DIR}/cairo-${CAIRO_VERSION}.tar.xz"
CAIRO_SRC="${SOURCES_DIR}/cairo-${CAIRO_VERSION}"

download "${CAIRO_URL}" "${CAIRO_TARBALL}"
if [ ! -d "${CAIRO_SRC}" ]; then
    tar xJf "${CAIRO_TARBALL}" -C "${SOURCES_DIR}"
fi

if [ ! -f "${TC_BUILD_DIR}/lib/pkgconfig/cairo-xlib.pc" ]; then
    cd "${CAIRO_SRC}"
    ./configure \
        --host="${CROSS_TC}" \
        --build="$(uname -m)-linux-gnu" \
        --prefix="${TC_BUILD_DIR}" \
        --enable-xlib=yes \
        --enable-xlib-xrender=yes \
        --enable-png=yes \
        --enable-ft=yes \
        --enable-fc=yes \
        --enable-pdf=no \
        --enable-ps=no \
        --enable-svg=no \
        --enable-interpreter=no \
        --enable-trace=no \
        --enable-gobject=no \
        --enable-script=no \
        2>&1 | tee "${LOG_DIR}/cairo-configure.log"
    make -j$(nproc) 2>&1 | tee "${LOG_DIR}/cairo-make.log"
    make install 2>&1 | tee "${LOG_DIR}/cairo-install.log"
    echo "Cairo build complete."
else
    echo "Cairo already built."
fi

# Verify cairo-xlib is findable
echo "Verifying cairo-xlib..."
${PKG_CONFIG} --exists cairo-xlib || {
    echo "ERROR: cairo-xlib not found after build"
    exit 1
}
echo "  cairo-xlib: $(${PKG_CONFIG} --modversion cairo-xlib)"

# === Step 2: Build GTK+ 2.24.33 ===
echo ""
echo "--- Step 2: GTK+ ${GTK_VERSION} ---"

# Update pkg-config to also find our newly built cairo
export PKG_CONFIG_LIBDIR="${TC_BUILD_DIR}/lib/pkgconfig:${SYSROOT}/usr/lib/pkgconfig"

GTK_TARBALL="${SOURCES_DIR}/gtk+-${GTK_VERSION}.tar.xz"
GTK_SRC="${SOURCES_DIR}/gtk+-${GTK_VERSION}"

download "${GTK_URL}" "${GTK_TARBALL}"
if [ ! -d "${GTK_SRC}" ]; then
    tar xJf "${GTK_TARBALL}" -C "${SOURCES_DIR}"
fi

if [ ! -f "${TC_BUILD_DIR}/lib/pkgconfig/gtk+-2.0.pc" ]; then
    cd "${GTK_SRC}"
    
    # Verify ALL deps are findable
    echo "Checking GTK dependencies..."
    for dep in glib-2.0 atk cairo cairo-xlib gdk-pixbuf-2.0 pango pangoft2 gio-2.0; do
        if ${PKG_CONFIG} --exists "${dep}"; then
            echo "  ✅ $dep ($(${PKG_CONFIG} --modversion ${dep}))"
        else
            echo "  ❌ $dep (not found)"
        fi
    done
    
    ./configure \
        --host="${CROSS_TC}" \
        --build="$(uname -m)-linux-gnu" \
        --prefix="${TC_BUILD_DIR}" \
        --with-gdktarget=x11 \
        --disable-gtk-doc \
        --disable-cups \
        --disable-papi \
        --disable-modules \
        --disable-glibtest \
        2>&1 | tee "${LOG_DIR}/gtk-configure.log"
    
    echo "Building GTK+ (this will take a while)..."
    make -j$(nproc) 2>&1 | tee "${LOG_DIR}/gtk-make.log"
    make install 2>&1 | tee "${LOG_DIR}/gtk-install.log"
    echo "GTK+ build complete."
else
    echo "GTK+ already built."
fi

echo ""
echo "======================================"
echo "Build complete!"
echo "======================================"
echo ""
echo "Output:"
ls -la "${TC_BUILD_DIR}/lib/libgtk-x11-2.0.so"* 2>/dev/null && echo "✅ libgtk-x11-2.0.so" || echo "❌ libgtk-x11-2.0.so not found"
ls -la "${TC_BUILD_DIR}/lib/libcairo.so"* 2>/dev/null && echo "✅ libcairo.so" || echo "❌ libcairo.so not found"
echo ""
echo "pkg-config gtk+-2.0:"
${PKG_CONFIG} --modversion gtk+-2.0 2>/dev/null || echo "not found"
