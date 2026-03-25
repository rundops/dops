#!/bin/sh
echo "==> Rotating secrets at $PATH"
echo ""
echo "  Authenticating with vault..."
sleep 0.5
echo "  Reading current secret..."
sleep 0.5
echo "  Generating new secret value..."
sleep 0.5
echo "  Writing new secret..."
sleep 0.5
echo "  Verifying..."
sleep 0.3
echo ""
echo "✓ Secret rotated at $PATH"
echo "  Version: $(( RANDOM % 100 + 1 ))"
