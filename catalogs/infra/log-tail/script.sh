#!/bin/sh
echo "Tailing last $LINES lines from $SERVICE..."
echo ""
for i in $(seq 1 "$LINES"); do
  HOUR=$(( RANDOM % 24 ))
  MIN=$(( RANDOM % 60 ))
  SEC=$(( RANDOM % 60 ))
  LEVEL="INFO"
  [ $(( RANDOM % 5 )) -eq 0 ] && LEVEL="WARN"
  [ $(( RANDOM % 10 )) -eq 0 ] && LEVEL="ERROR"
  printf "2026-03-24T%02d:%02d:%02d [%s] %s: request processed in %dms\n" \
    "$HOUR" "$MIN" "$SEC" "$LEVEL" "$SERVICE" "$(( RANDOM % 500 + 10 ))"
done
