#!/bin/sh
set -e
ENV="$ENVIRONMENT"
VER="$VERSION"
FEATURES_VAL="$FEATURES"
DRY="$DRY_RUN"

[ "$DRY" = "true" ] && echo "[DRY RUN] Simulating deploy of $VER to $ENV" && echo ""

echo "==> Stage 1/4: Build"
echo "    Building application $VER..."
sleep 1
echo "    Compiling source..."
sleep 0.5
echo "    Build artifact: app-${VER}-linux-amd64.tar.gz (42.3 MB)"
echo "    Build complete."
echo ""

echo "==> Stage 2/4: Test"
echo "    Running test suite..."
sleep 1
echo "    Unit tests:        148 passed, 0 failed"
sleep 0.3
echo "    Integration tests:  37 passed, 0 failed"
echo "    All tests passed."
echo ""

echo "==> Stage 3/4: Push"
echo "    Pushing artifact to registry..."
sleep 1
echo "    registry.example.com/app:$VER ... done"
echo "    Push complete."
echo ""

echo "==> Stage 4/4: Deploy to $ENV"
echo "    Target cluster: $ENV-cluster-01"
sleep 0.5
for i in 0 1 2 3; do
  sleep 0.8
  echo "    Rolling update: $i/3 replicas ready"
done
echo ""

if [ -n "$FEATURES_VAL" ]; then
  echo "==> Enabling platform features"
  OLD_IFS="$IFS"; IFS=','
  for f in $FEATURES_VAL; do
    IFS="$OLD_IFS"; f=$(echo "$f" | xargs)
    echo "    Enabling $f... done"
    sleep 0.3
  done
  IFS="$OLD_IFS"
  echo ""
fi

echo "========================================="
echo "  Deployment Summary"
echo "========================================="
echo "  Version:     $VER"
echo "  Environment: $ENV"
echo "  Status:      SUCCESS"
[ -n "$FEATURES_VAL" ] && echo "  Features:    $FEATURES_VAL"
[ "$DRY" = "true" ] && echo "  Mode:        DRY RUN"
echo "========================================="
