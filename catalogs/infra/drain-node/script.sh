#!/bin/sh
echo "==> Cordoning node $NODE..."
sleep 0.5
echo "    Node $NODE cordoned"
echo ""
echo "==> Draining node $NODE..."
for i in 1 2 3 4; do
  sleep 0.5
  echo "    Evicting pod app-$(LC_ALL=C tr -dc 'a-z0-9' </dev/urandom | head -c 5)"
done
sleep 0.5
echo ""
echo "✓ Node $NODE drained successfully"
echo "  Pods evicted: 4"
echo "  Ready to decommission"
