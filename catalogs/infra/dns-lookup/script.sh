#!/bin/sh
echo "Querying $RECORD_TYPE record for $DOMAIN..."
echo ""
dig +short "$DOMAIN" "$RECORD_TYPE" 2>/dev/null || nslookup "$DOMAIN" 2>/dev/null || echo "dig/nslookup not available"
