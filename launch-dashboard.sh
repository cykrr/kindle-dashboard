#!/bin/sh
set -e
export XDG_CONFIG_HOME="/mnt/us/system/browser/"
export LD_LIBRARY_PATH="/usr/bin/chromium/lib:/usr/bin/chromium/usr/lib:/usr/lib/"
/usr/bin/chromium/bin/kindle_browser --app="file:///mnt/us/documents/kindle-dashboard/wrapper.html" --no-zygote --no-sandbox --single-process --skia-resource-cache-limit-mb=64 --disable-gpu --in-process-gpu --disable-gpu-sandbox --disable-gpu-compositing --enable-dom-distiller --enable-distillability-service --force-device-scale-factor=1 --js-flags=jitless --content-shell-hide-toolbar --content-shell-host-window-cord=0,215 --force-gpu-mem-available-mb=32 --enable-grayscale-mode --enable-low-end-device-mode --enable-low-res-tiling --disable-site-isolation-trials --user-agent="Mozilla/5.0 (X11; U; Linux armv7l like Android; en-us) AppleWebKit/531.2+ (KHTML, like Gecko) Version/5.0 Safari/533.2+ Kindle/3.0+" >/mnt/us/documents/kindle-dashboard/browser.out 2>&1 &
echo $! >/mnt/us/documents/kindle-dashboard/browser.pid
