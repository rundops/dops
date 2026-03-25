#!/bin/sh
echo "Checking SSL certificate for $DOMAIN..."
echo ""
CERT_INFO=$(echo | openssl s_client -servername "$DOMAIN" -connect "$DOMAIN:443" 2>/dev/null | openssl x509 -noout -dates -subject -issuer 2>/dev/null)
if [ -n "$CERT_INFO" ]; then
  echo "$CERT_INFO"
  if [ "$VERBOSE" = "true" ]; then
    echo ""
    echo "=== Full Certificate ==="
    echo | openssl s_client -servername "$DOMAIN" -connect "$DOMAIN:443" 2>/dev/null | openssl x509 -noout -text 2>/dev/null
  fi
else
  echo "Could not retrieve certificate"
  exit 1
fi
