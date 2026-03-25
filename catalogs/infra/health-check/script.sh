#!/bin/sh
echo "Health check: $ENDPOINT"
echo ""
STATUS=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$ENDPOINT" 2>/dev/null)
if [ "$STATUS" = "200" ] || [ "$STATUS" = "301" ] || [ "$STATUS" = "302" ]; then
  echo "✓ Status: $STATUS OK"
else
  echo "✗ Status: $STATUS"
  exit 1
fi
