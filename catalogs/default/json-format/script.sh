#!/bin/sh
echo "$INPUT" | python3 -m json.tool 2>/dev/null || echo "Invalid JSON"
