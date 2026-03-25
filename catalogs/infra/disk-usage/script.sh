#!/bin/sh
echo "Disk usage for $MOUNT:"
echo ""
df -h "$MOUNT" 2>/dev/null || echo "Mount point not found: $MOUNT"
echo ""
echo "Largest directories:"
du -sh "$MOUNT"/* 2>/dev/null | sort -rh | head -10
