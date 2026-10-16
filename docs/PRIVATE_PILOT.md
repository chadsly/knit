# Knit Private Pilot

This is a local pilot, not a public Plugin Directory release. These steps do not
publish to GitHub or registries, sign, or notarize a release.

## Prerequisites

- Codex Desktop and the official Codex CLI available as `codex` on PATH.
- Python 3 for installation and managed local speech-to-text.
- FFmpeg on the daemon host PATH for timestamped video frame review.
- Matching Knit Browser Composer extension. Development source is ahead of the
  Web Store build; use the supplied unpacked extension for this pilot.

The package includes its native Knit binary. No Go compiler, Node runtime,
source checkout, API key, or manual MCP configuration is needed. Python, FFmpeg,
the CLI and extension are not bundled. The installer does not install missing
prerequisites, bypass OS warnings, or change system/microphone permissions.

## Install And Connect

1. Extract the ZIP matching your OS and processor. Do not use earlier validation
   binaries built with Go 1.25.0; current pilot binaries use patched Go 1.26.9.
2. On macOS open `Install.command`. On Linux/Windows run
   `python3 install-pilot.py` from the folder (use `python` if that is your Python
   3 command). Double-click installation is currently macOS-only.
3. Restart Codex, enable Knit Browser Feedback for the conversation, and open its
   persistent conversation-side panel. Connect once to the intended workspace.
4. Copy its short-lived code into Chrome's Knit extension. Select that exact
   destination in Send to. Enable automatic delivery only if desired.
5. Keep the persistent panel visible. Capture, preview, submit, and check the
   factual Completed/Failed result. Delivered alone does not mean work completed.

The installer registers a local private marketplace, installs the plugin, and
resolves installed binary paths for older Codex builds. It does not manually edit
config.toml or embed credentials. Re-run it after upgrading; do not move files
inside the installed plugin cache.

## State And Recovery

`knit mcp --desktop` defaults to port 7778 and stores durable data in the OS user
configuration directory under knit/desktop, outside versioned plugin folders.
Normal `knit start` and `knit mcp` retain their existing defaults. Explicit
KNIT_ADDR, KNIT_DATA_DIR, KNIT_CONFIG_PATH and KNIT_SQLITE_PATH overrides win.

Existing source-checkout development data is not silently migrated. Close/disable
the development plugin before testing the pilot on the same port. Pair once to
the new pilot profile. The original profile is retained; do not delete it.
An incompatible daemon yields an error rather than being replaced or reconfigured.

After restarting Codex, reopen the existing persistent panel; do not create a new
destination for each request. Temporarily unavailable connections retain queued
feedback. Uncertain sends are not retried automatically: inspect the conversation
before explicitly resubmitting. Disconnect revokes the destination; select a new
explicit destination instead of expecting silent rerouting. Codex permissions
remain authoritative for workspace/read-only failures.

Video is sampled images plus narration text, not native video playback. Brief
interactions between samples may be missed; send a snapshot or precise text when
needed.

## Validation

```sh
node plugins/knit-desktop/package.cjs /path/to/daemon /existing/output darwin_arm64 1.0.0-rc.3
KNIT_PILOT_TEST_BINARY=/path/to/native/daemon node --test test/desktop-pilot-install.test.cjs
```

The fresh CODEX_HOME test installs through the actual Codex CLI, calls knit_status
through an ephemeral Codex runtime, checks desktop tool discovery and verifies
owned-daemon shutdown. No model turn, existing profile or user workspace is used.
This does not certify unassisted fresh-machine desktop UI onboarding. macOS
installation is locally verified; Linux/Windows are cross-built, not native-tested.

Public local-MCP directory distribution still requires approval:
https://developers.openai.com/plugins/deploy/desktop-only
