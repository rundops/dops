#!/bin/sh
# Test MCP tool schema defaults by sending JSON-RPC via stdio.
# Usage: ./scripts/test-mcp-schema.sh [runbook-id]
#
# Example: ./scripts/test-mcp-schema.sh default.hello-world

RUNBOOK_ID="${1:-default.hello-world}"

# Send initialize + tools/list, then close stdin immediately.
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}\n' \
  | timeout 5 ./dops mcp serve --transport stdio 2>/dev/null \
  | while IFS= read -r line; do
      echo "$line" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if 'result' in data and 'tools' in data.get('result', {}):
        for tool in data['result']['tools']:
            if tool['name'] == '$RUNBOOK_ID':
                print(json.dumps(tool, indent=2))
except:
    pass
"
    done
