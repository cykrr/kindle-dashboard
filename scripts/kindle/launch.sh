#!/bin/sh
# Standalone launch script for the Kindle-native GTK dashboard.
#
# This is the OLD direct entry point. It now delegates to the KUAL extension
# helper so all paths go through the same logic.
#
# NOTE: USB networking (RNDIS gadget) is managed by the SSH Manager KUAL
# extension, NOT by this script. Run the SSH Manager's usbstart first if
# you need USB network access.

/mnt/us/extensions/Kindle-Dashboard/Kindle-Dashboard.sh "${@:-start}"
