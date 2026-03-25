#!/bin/sh
if [ "$DECODE" = "true" ]; then
  echo "$INPUT" | base64 -d 2>/dev/null || echo "$INPUT" | base64 -D
else
  echo -n "$INPUT" | base64
fi
