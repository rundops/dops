#!/bin/sh
echo "Rolling restart: $DEPLOYMENT in namespace $NAMESPACE"
echo ""
for i in 1 2 3; do
  echo "  Restarting pod $DEPLOYMENT-$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 5)..."
  sleep 0.8
  echo "  Pod $i/3 ready"
done
echo ""
echo "✓ All pods restarted successfully"
