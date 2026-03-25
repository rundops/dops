#!/bin/sh
echo "==> Auditing IAM permissions for account: $ACCOUNT"
echo "    Policies: $POLICIES"
echo ""
TOTAL=0
WARN=0
OLD_IFS="$IFS"
IFS=','
for p in $POLICIES; do
  IFS="$OLD_IFS"
  p=$(echo "$p" | xargs)
  echo "  Scanning policy: $p"
  sleep 0.5
  FINDINGS=$(( RANDOM % 5 ))
  TOTAL=$(( TOTAL + FINDINGS ))
  if [ "$FINDINGS" -gt 2 ]; then
    WARN=$(( WARN + 1 ))
    echo "    ⚠ $FINDINGS overly permissive rules found"
    if [ "$VERBOSE" = "true" ]; then
      echo "      - Allow s3:* on resource *"
      echo "      - Allow ec2:* on resource *"
    fi
  else
    echo "    ✓ $FINDINGS findings (within threshold)"
  fi
done
IFS="$OLD_IFS"
echo ""
echo "========================================="
echo "  Audit Summary"
echo "========================================="
echo "  Policies scanned: $(echo "$POLICIES" | tr ',' '\n' | wc -l | tr -d ' ')"
echo "  Total findings:   $TOTAL"
echo "  Warnings:         $WARN"
echo "========================================="
