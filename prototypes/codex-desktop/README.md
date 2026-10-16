# Desktop Delivery Gate

Development probe, not the Knit 1.0 product. No daemon connection, credentials, browser pairing, automatic pickup, or result tracking is included. It tests whether the installed host can accept a message from a conversation panel.

Build from repository root:

```sh
node prototypes/codex-desktop/setup.cjs
```

The repo marketplace exposes Knit Desktop Probe under Knit Development. Install it with the desktop Plugins directory, then enable it in a fresh local Codex conversation. Ask to open the connection check and click Send test message. Confirm the marker appears as both a message and an agent response in the same conversation. Record the app version and marker. Do not mark this gate passed based only on CLI installation or mocked host tests.

Protocol/browser contract verification requires Playwright available in NODE_PATH and a local Chrome installation (override CHROME_PATH for another executable):

```sh
node --test prototypes/codex-desktop/delivery.test.mjs
node prototypes/codex-desktop/verify.cjs
```

verify.cjs launches the real compiled MCP process, reads its panel resource, then tests the panel in an isolated Chromium context with an injected host bridge. It explicitly does not certify the installed desktop host.

Installation repaired on October 15, 2026 as version 0.0.4. The installed desktop runtime successfully lists open_connection_check through mcpServerStatus/list after initialization of an ephemeral diagnostic thread, without an agent turn. Run `python3 prototypes/codex-desktop/check-runtime.py` to check runtime loading. This diagnostic starts a separate server solely to validate plugin loading and does not establish desktop UI delivery.

The compatibility manifest explicitly references .mcp.json. setup.cjs generates that file with resolved local executable and working-directory paths because placeholder paths failed to launch in the installed runtime. Generated launch paths and the binary are excluded from Git; run setup before tests/install on another machine. The Knit Development marketplace must already be registered. Remove the plugin with `codex plugin remove knit-desktop-probe@knit-development` when finished.
# Retired Diagnostic

Knit Desktop Probe was the initial connection diagnostic, not the product
integration. It is no longer listed in Knit Development. Use
[Knit Browser Feedback Preview](../../plugins/knit-desktop/README.md) instead.
This directory is retained as historical development evidence.
