#!/bin/sh
TIMEOUT_VAL="${TIMEOUT:-5}"
echo "Testing connection to $HOST:$PORT (timeout: ${TIMEOUT_VAL}s)..."
if nc -z -w "$TIMEOUT_VAL" "$HOST" "$PORT" 2>/dev/null; then
  echo "✓ Connection successful"
else
  echo "✗ Connection failed"
  exit 1
fi
