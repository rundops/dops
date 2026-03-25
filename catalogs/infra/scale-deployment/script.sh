#!/bin/sh
echo "Scaling $DEPLOYMENT to $REPLICAS replicas in $NAMESPACE..."
echo ""
CURRENT=3
echo "  Current replicas: $CURRENT"
echo "  Target replicas:  $REPLICAS"
sleep 1
for i in $(seq 1 "$REPLICAS"); do
  sleep 0.3
  echo "  Replica $i/$REPLICAS ready"
done
echo ""
echo "✓ Scaled $DEPLOYMENT to $REPLICAS replicas"
