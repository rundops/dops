#!/bin/sh
cd "$PATH_VAR" 2>/dev/null || cd "$PATH" 2>/dev/null || { echo "Cannot cd to repo"; exit 1; }
echo "=== Recent Commits ==="
git log --oneline -10 2>/dev/null || echo "(not a git repo)"
echo ""
echo "=== Contributors ==="
git shortlog -sn --no-merges -10 2>/dev/null || echo "(not a git repo)"
