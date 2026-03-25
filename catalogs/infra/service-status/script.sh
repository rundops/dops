#!/bin/sh
echo "Checking service '$SERVICE' on $HOST..."
sleep 0.5
echo ""
echo "Service: $SERVICE"
echo "Host:    $HOST"
echo "Status:  active (running)"
echo "PID:     $(( RANDOM % 50000 + 1000 ))"
echo "Uptime:  $(( RANDOM % 30 + 1 )) days"
echo "Memory:  $(( RANDOM % 512 + 64 )) MB"
