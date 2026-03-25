#!/bin/sh
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
FILENAME="${DATABASE}-${TIMESTAMP}.${FORMAT}"
echo "==> Backing up database: $DATABASE"
echo "    Format: $FORMAT"
echo "    Output: /backups/$FILENAME"
echo ""
echo "  Connecting to database..."
sleep 0.5
echo "  Dumping tables..."
sleep 2
echo "    users:       1,234 rows"
sleep 0.3
echo "    orders:      5,678 rows"
sleep 0.3
echo "    products:    892 rows"
sleep 0.3
echo "    sessions:    12,345 rows"
sleep 0.5
echo ""
echo "  Compressing backup..."
sleep 1
echo ""
echo "✓ Backup complete"
echo "  File: /backups/$FILENAME"
echo "  Size: 23.4 MB"
echo "  Duration: 4.2s"
