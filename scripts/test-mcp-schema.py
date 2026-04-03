#!/usr/bin/env python3
"""Query MCP server for tool schemas and print them.

Usage:
    python3 scripts/test-mcp-schema.py [tool-name-filter]
    python3 scripts/test-mcp-schema.py default.hello-world
    python3 scripts/test-mcp-schema.py              # shows all tools
"""
import json
import subprocess
import sys


def send_msg(proc, msg):
    data = json.dumps(msg)
    frame = f"Content-Length: {len(data)}\r\n\r\n{data}"
    proc.stdin.write(frame.encode())
    proc.stdin.flush()


def read_msg(proc):
    # Read Content-Length header
    headers = b""
    while True:
        ch = proc.stdout.read(1)
        if not ch:
            return None
        headers += ch
        if headers.endswith(b"\r\n\r\n"):
            break
    length = int(headers.decode().split(":")[1].strip().split("\r\n")[0])
    body = proc.stdout.read(length)
    return json.loads(body)


def main():
    tool_filter = sys.argv[1] if len(sys.argv) > 1 else None

    proc = subprocess.Popen(
        ["./dops", "mcp", "serve", "--transport", "stdio"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
    )

    # Initialize
    send_msg(proc, {
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "test", "version": "1.0"},
        },
    })

    # Read responses until we get id=1
    while True:
        msg = read_msg(proc)
        if msg and msg.get("id") == 1:
            break

    # Send initialized notification
    send_msg(proc, {"jsonrpc": "2.0", "method": "notifications/initialized"})

    # List tools
    send_msg(proc, {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})

    # Read responses until we get id=2
    while True:
        msg = read_msg(proc)
        if msg and msg.get("id") == 2:
            tools = msg.get("result", {}).get("tools", [])
            for tool in tools:
                if tool_filter and tool["name"] != tool_filter:
                    continue
                print(json.dumps(tool, indent=2))
            break

    proc.stdin.close()
    proc.terminate()
    proc.wait()


if __name__ == "__main__":
    main()
