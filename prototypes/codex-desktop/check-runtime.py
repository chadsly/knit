"""Check plugin loading in a diagnostic runtime, without starting an agent turn.

This does not certify that an already-running desktop chat has refreshed plugins.
"""
import json
import os
import selectors
import subprocess
import sys
import time

binary = os.environ.get("CODEX_PROBE_CLI", "codex")
server_name = os.environ.get("KNIT_DIAGNOSTIC_SERVER", "knit_probe")
process = subprocess.Popen(
    [binary, "app-server"], stdin=subprocess.PIPE,
    stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True,
)
selector = selectors.DefaultSelector()
selector.register(process.stdout, selectors.EVENT_READ)


def call(identifier, method, params):
    process.stdin.write(json.dumps({"id": identifier, "method": method, "params": params}) + "\n")
    process.stdin.flush()
    deadline = time.monotonic() + 20
    while time.monotonic() < deadline:
        if not selector.select(timeout=0.5):
            continue
        line = process.stdout.readline()
        if not line:
            raise RuntimeError("Diagnostic runtime exited")
        response = json.loads(line)
        if response.get("id") == identifier:
            if "error" in response:
                raise RuntimeError(str(response["error"]))
            return response.get("result", {})
    raise TimeoutError(method)


try:
    call(1, "initialize", {"clientInfo": {"name": "knit_diagnostics", "version": "0.0.3"},
                           "capabilities": {"experimentalApi": True}})
    process.stdin.write('{"method":"initialized","params":{}}\n')
    process.stdin.flush()
    thread = call(2, "thread/start", {"cwd": os.getcwd(), "ephemeral": True,
                             "approvalPolicy": "untrusted", "sandbox": "read-only"})
    deadline = time.monotonic() + 20
    identifier = 3
    if os.getenv("KNIT_DIAGNOSTIC_TOOL"):
        call(identifier, "mcpServer/tool/call", {"server": server_name,
             "threadId": thread["thread"]["id"], "tool": os.environ["KNIT_DIAGNOSTIC_TOOL"], "arguments": {}})
        identifier += 1
    servers = []
    while time.monotonic() < deadline:
        result = call(identifier, "mcpServerStatus/list", {})
        identifier += 1
        servers = [row for row in result.get("data", []) if row.get("name") == server_name]
        if servers and servers[0].get("tools"):
            break
        time.sleep(0.5)
    print(json.dumps(servers, indent=2))
    if not servers or not servers[0].get("tools"):
        sys.exit(1)
finally:
    selector.close()
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()
