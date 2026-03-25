#!/bin/sh
echo "==> Canary Deploy: $VERSION to $ENVIRONMENT ($PERCENTAGE% traffic)"
echo ""
echo "  [1/4] Building canary image..."
sleep 1
echo "  [2/4] Deploying canary pod..."
sleep 1
echo "  [3/4] Configuring traffic split: ${PERCENTAGE}% canary, $((100 - PERCENTAGE))% stable"
sleep 0.5
echo "  [4/4] Monitoring canary health..."
sleep 2
echo ""
echo "  Canary metrics (30s window):"
echo "    Success rate: 99.7%"
echo "    p50 latency:  12ms"
echo "    p99 latency:  89ms"
echo "    Error rate:   0.3%"
echo ""
echo "✓ Canary healthy at ${PERCENTAGE}% traffic"
