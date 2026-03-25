#!/bin/sh
echo "Scanning /tmp for files older than $DAYS days..."
COUNT=$(find /tmp -maxdepth 1 -type f -mtime +"$DAYS" 2>/dev/null | wc -l | tr -d ' ')
echo "Found $COUNT files"
echo "Cleanup complete."
