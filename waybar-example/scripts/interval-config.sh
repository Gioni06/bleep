#!/bin/bash

CONFIG_FILE="$HOME/.config/waybar/interval-config"

# Load current configuration
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
    current_minutes="${INTERVAL_MINUTES:-25}"
    current_seconds="${INTERVAL_SECONDS:-0}"
else
    current_minutes=25
    current_seconds=0
fi

echo "Interval Timer Configuration"
echo "============================="
echo "Current: -m $current_minutes -s $current_seconds"
echo ""
echo "Enter comma-separated values for rotating intervals"
echo "Example: '1,2' and '0,30' → 1min0sec, 2min30sec"
echo "Leave blank to keep current value"
echo ""

# Prompt for minutes
echo -n "Enter minutes (comma-separated) [$current_minutes]: "
read minutes

# Prompt for seconds
echo -n "Enter seconds (comma-separated) [$current_seconds]: "
read seconds

# Use current values if blank
if [ -z "$minutes" ]; then
    minutes="$current_minutes"
fi

if [ -z "$seconds" ]; then
    seconds="$current_seconds"
fi

# Validate input - allow numbers and commas, ensure not empty
if ! [[ "$minutes" =~ ^[0-9]+(,[0-9]+)*$ ]]; then
    echo "Invalid minutes value. Using: 25"
    minutes=25
fi

if ! [[ "$seconds" =~ ^[0-9]+(,[0-9]+)*$ ]]; then
    echo "Invalid seconds value. Using: 0"
    seconds=0
fi

# Validate that each interval is positive (minutes > 0 OR seconds > 0)
IFS=',' read -ra MIN_ARRAY <<< "$minutes"
IFS=',' read -ra SEC_ARRAY <<< "$seconds"

# Ensure arrays have same length, pad with last value (matching bleep behavior)
max_len=${#MIN_ARRAY[@]}
if [ ${#SEC_ARRAY[@]} -gt $max_len ]; then
    max_len=${#SEC_ARRAY[@]}
fi

last_min="${MIN_ARRAY[-1]}"
last_sec="${SEC_ARRAY[-1]:-0}"

for ((i=${#MIN_ARRAY[@]}; i<$max_len; i++)); do
    MIN_ARRAY[$i]=$last_min
done

for ((i=${#SEC_ARRAY[@]}; i<$max_len; i++)); do
    SEC_ARRAY[$i]=$last_sec
done

# Check each interval
invalid=false
for i in "${!MIN_ARRAY[@]}"; do
    min="${MIN_ARRAY[$i]}"
    sec="${SEC_ARRAY[$i]:-0}"
    if [ "$min" -eq 0 ] && [ "$sec" -eq 0 ]; then
        echo "Error: Interval $((i+1)) would be 0m0s (invalid)"
        invalid=true
        break
    fi
done

if [ "$invalid" = true ]; then
    echo "Using default: 25m0s"
    minutes=25
    seconds=0
fi

# Save configuration
echo "INTERVAL_MINUTES=$minutes" > "$CONFIG_FILE"
echo "INTERVAL_SECONDS=$seconds" >> "$CONFIG_FILE"

echo ""
echo "Configuration saved: -m $minutes -s $seconds"
echo "Restarting waybar..."

# Restart waybar to apply new interval
pkill -SIGUSR2 waybar

echo "Done! Press Enter to close..."
read
