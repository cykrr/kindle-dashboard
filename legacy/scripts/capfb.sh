#!/bin/sh
dd if=/dev/fb0 bs=608 count=800 of=/tmp/fb.raw 2>/dev/null
echo "done"
