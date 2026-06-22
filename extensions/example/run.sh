#!/bin/bash
# PwnDuck extension entrypoint
# Available env vars:
#   PWNDUCK_BASE  — /opt/pwnduck
#   PWNDUCK_EVENT — USB_CONNECTED etc
echo "Extension fired at $(date)" >> "$PWNDUCK_BASE/loot/extension.log"
