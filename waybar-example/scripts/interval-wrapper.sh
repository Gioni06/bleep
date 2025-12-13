#!/bin/bash

# Config file to store interval parameters
CONFIG_FILE="$HOME/.config/waybar/interval-config"

# Read config or use defaults
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
else
    INTERVAL_MINUTES=25
    INTERVAL_SECONDS=0
fi

# Start bleep with configured parameters
exec bleep -json -paused -m "${INTERVAL_MINUTES}" -s "${INTERVAL_SECONDS}"
