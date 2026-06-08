#!/bin/sh
killall -9 dashboard-native 2>/dev/null || true
lipc-set-prop -i com.lab126.powerd preventScreenSaver 0 2>/dev/null || true
# Restart Kindle GUI
start lab126_gui 2>/dev/null || /etc/init.d/framework start 2>/dev/null || true
