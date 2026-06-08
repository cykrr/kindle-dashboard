#!/bin/sh
export DISPLAY=:0
xwininfo -root -tree 2>/dev/null | grep "Kindle Dashboard" | head -1
