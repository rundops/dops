#!/bin/sh
LEN="${LENGTH:-32}"
case "$CHARSET" in
  alphanumeric) LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c "$LEN" ;;
  alphanumeric-symbols) LC_ALL=C tr -dc 'A-Za-z0-9!@#$%^&*' </dev/urandom | head -c "$LEN" ;;
  hex) LC_ALL=C tr -dc 'a-f0-9' </dev/urandom | head -c "$LEN" ;;
  numeric) LC_ALL=C tr -dc '0-9' </dev/urandom | head -c "$LEN" ;;
esac
echo ""
