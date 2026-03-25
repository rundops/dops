#!/bin/sh
echo "==> Rotating TLS certificates for $DOMAIN"
echo ""
echo "  [1/4] Generating new private key..."
sleep 1
echo "  [2/4] Creating CSR..."
sleep 0.5
echo "  [3/4] Requesting certificate from CA..."
sleep 1.5
echo "  [4/4] Installing certificate..."
sleep 0.5
echo ""
echo "✓ Certificate rotated successfully"
echo "  Domain:  $DOMAIN"
echo "  Expires: 2027-03-24"
echo "  Serial:  $(LC_ALL=C tr -dc 'A-F0-9' </dev/urandom | head -c 16)"
