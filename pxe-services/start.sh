#!/bin/sh

# Start dnsmasq in background
dnsmasq -d --log-queries --log-dhcp &
PID=$!

echo "Started dnsmasq with PID $PID"

# Watch for reload trigger
while true; do
    if [ -f /etc/dnsmasq.d/.reload ]; then
        echo "Configuration change detected, reloading dnsmasq..."
        rm -f /etc/dnsmasq.d/.reload
        kill -HUP $PID
    fi
    # Check if dnsmasq is still running
    if ! kill -0 $PID 2>/dev/null; then
        echo "dnsmasq process died, exiting..."
        exit 1
    fi
    sleep 2
done
