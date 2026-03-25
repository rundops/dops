#!/bin/sh
echo "==> Rolling back to $VERSION in $NAMESPACE"
[ "$FORCE" = "true" ] && echo "    ⚠ Force mode: skipping health checks"
echo ""
echo "  [1/3] Pulling image registry.example.com/app:$VERSION..."
sleep 1
echo "  [2/3] Updating deployment..."
sleep 1
echo "  [3/3] Waiting for rollout..."
sleep 1.5
echo "    0/3 ready"
sleep 0.5
echo "    1/3 ready"
sleep 0.5
echo "    2/3 ready"
sleep 0.5
echo "    3/3 ready"
echo ""
echo "✓ Rollback to $VERSION complete"
