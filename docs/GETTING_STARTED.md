# Getting Started

This guide is scoped to the current Knit repository: a local desktop daemon, tray controller, browser companion, and local web UI for capturing feedback and submitting review packages to coding-agent adapters.

## What Runs

- `cmd/daemon`: starts the local HTTP server and serves the main UI on `http://127.0.0.1:7777`
- `cmd/tray`: launches a native tray controller and ensures the daemon is running in a detached child process
- `cmd/ui`: placeholder binary that prints the daemon URL; the real UI is still served by the daemon

## Prerequisites

- Go installed locally
- A supported desktop OS: macOS, Windows, or Linux
- Browser DevTools access for companion injection, or Chromium for the bundled extension flow
- Optional: `OPENAI_API_KEY` if you use the remote transcription path or the `codex_api` submit adapter
- Optional: `ANTHROPIC_API_KEY` if you use the `claude_api` submit adapter

## Install And Start

Use one of these methods to install and start the daemon.

Package-manager tabs below assume you are using the generated or published npm wrapper package named `@knit/daemon`. That wrapper installs and starts the daemon binary only.

:::tabs
@tab Go source
From repo root:

```bash
cd YOUR_REPO
go test ./...
go run ./cmd/daemon
```

@tab npm
Install the packaged daemon wrapper, then start it:

```bash
npm install -g @knit/daemon
knit-daemon start
```

@tab pnpm
Install the packaged daemon wrapper, then start it:

```bash
pnpm add -g @knit/daemon
knit-daemon start
```

@tab yarn
Install the packaged daemon wrapper, then start it:

```bash
yarn global add @knit/daemon
knit-daemon start
```

@tab bun
Install the packaged daemon wrapper, then start it:

```bash
bun add -g @knit/daemon
knit-daemon start
```
:::

Then open:

```text
http://127.0.0.1:7777
```

By default, Knit writes its persistent user config to `./knit.toml`. If you already have a legacy `.knit/knit.toml`, Knit will keep using that until you move it. Set `KNIT_CONFIG_PATH` if you want the file somewhere else. Use `.env` for secrets such as `OPENAI_API_KEY` and `ANTHROPIC_API_KEY`; the checked-in examples are `knit.toml.example` and `.env.example`.

If you want the tray controller instead of just the daemon, use one of these methods:

:::tabs
@tab Go source
From repo root:

```bash
cd YOUR_REPO
go run ./cmd/tray
```

@tab Built binary
Build and run the tray locally:

```bash
go build -o ./dist/local/tray ./cmd/tray
./dist/local/tray
```

@tab npm / pnpm / yarn / bun
The generated package-manager wrapper installs and starts the daemon only.

Use the tray from source or from a built release artifact when you need the native tray controller.
:::

Tray mode is only a controller. It starts or reconnects to the daemon, exposes `Open UI`, `Pause Capture`, `Resume Capture`, and `Kill Capture`, and leaves the daemon running when you choose `Quit Tray`.

## Core Environment

These are the main runtime variables wired through `internal/app/app.go` and the adapter/transcription packages:

- `KNIT_ADDR`: listen address for the local server
- `KNIT_DATA_DIR`: local state directory; defaults under `.knit`
- `KNIT_SQLITE_PATH`: SQLite file path, resolved under `KNIT_DATA_DIR` when relative
- `KNIT_CONTROL_TOKEN`: optional local API token; generated automatically when unset
- `KNIT_CONTROL_CAPABILITIES`: capability list for local API actions
- `KNIT_PROFILE`, `KNIT_ENVIRONMENT`, `KNIT_BUILD_ID`: profile and build metadata exposed through runtime state
- `KNIT_AUTO_START`: opt-in OS auto-start registration for built binaries
- `KNIT_POINTER_SAMPLE_HZ`: pointer sampling rate for browser companion tracking

Submission runtime:

- `KNIT_DEFAULT_PROVIDER`: default adapter selection in UI/runtime state
- `KNIT_CLI_ADAPTER_CMD`: required for `codex_cli`; use `scripts/knit-codex-cli-adapter.sh` for the bundled wrapper
- `KNIT_CLAUDE_CLI_ADAPTER_CMD`: optional Claude-compatible CLI adapter; use `scripts/knit-claude-cli-adapter.sh` for the bundled wrapper
- `KNIT_OPENCODE_CLI_ADAPTER_CMD`: optional OpenCode-compatible CLI adapter; use `scripts/knit-opencode-cli-adapter.sh` for the bundled wrapper
- `KNIT_CODEX_WORKDIR`: repo working directory for the Codex CLI wrapper
- `KNIT_SUBMIT_EXECUTION_MODE`: `series` or `parallel`
- `KNIT_ALLOW_REMOTE_SUBMISSION`: gates remote submit adapters

If you launch Knit from this repository, the runtime auto-fills those three CLI command fields with the bundled scripts when they are blank. You only need to override them when you want a custom adapter command.

Transcription runtime:

- `KNIT_TRANSCRIPTION_MODE`: `faster_whisper`, `remote`, `local`, or `lmstudio`
- `KNIT_ALLOW_REMOTE_STT`: gates remote transcription
- `KNIT_LOCAL_STT_CMD`: required for `local`
- `KNIT_FASTER_WHISPER_*`: managed local faster-whisper configuration
- `KNIT_LMSTUDIO_*`: LM Studio endpoint and model settings
- `OPENAI_API_KEY`, `OPENAI_STT_MODEL`, `OPENAI_BASE_URL`: remote STT configuration
- `OPENAI_API_KEY`, `CODEX_MODEL`, `OPENAI_BASE_URL`: `codex_api` submit adapter
- `ANTHROPIC_API_KEY`, `KNIT_CLAUDE_API_MODEL`, `ANTHROPIC_BASE_URL`: `claude_api` submit adapter

Security and egress:

- `KNIT_ENCRYPTION_KEY_B64`: optional override for the local encryption key
- `KNIT_OUTBOUND_ALLOWLIST`, `KNIT_OUTBOUND_BLOCKLIST`: outbound domain controls
- `KNIT_ALLOWED_SUBMIT_PROVIDERS`: provider allowlist
- `KNIT_HTTP_PROXY`, `KNIT_NO_PROXY`, `KNIT_TLS_CA_FILE`, `KNIT_TLS_PINNED_CERT_SHA256`: outbound HTTP transport policy

Retention and audit:

- `KNIT_AUDIO_RETENTION`, `KNIT_SCREENSHOT_RETENTION`, `KNIT_VIDEO_RETENTION`, `KNIT_TRANSCRIPT_RETENTION`, `KNIT_STRUCTURED_RETENTION`
- `KNIT_PURGE_INTERVAL`, `KNIT_PURGE_SCHEDULE`, `KNIT_ARTIFACT_MAX_FILES`
- `KNIT_SIEM_JSONL_PATH`: optional mirrored audit sink

## First Session

1. Start the daemon or tray.
2. Open the local UI and choose a workspace directory.
3. Use the `Docs` button on the main page whenever you need the local operator guides in a separate tab.
4. Start a session with a target window and target URL.
5. In `Capture, review, and send`, pick one of the three Step 2 paths:
   - `Chrome Extension` (`easy`) if you want the browser-native side panel flow.
   - `Popout Composer` (`intermediate`) if you want the detached composer near the page you are reviewing.
   - `Main UI Interface` (`Main UI`) if you want to stay in the main page and capture from there.
6. Enable visual capture if you want screenshots or clips.
7. Configure audio and transcription mode if you plan to capture voice notes.
8. Add feedback notes, then open `Settings -> Agent` and choose a prompt template:
   - `Implement changes`
   - `Draft plan`
   - `Create Jira tickets`
9. Knit will load the selected template into the prompt text box. Edit that text directly if you need to change what the agent should do with the approved feedback.
10. Review the preview package, then approve and submit.
11. If you need to stop a send, use `Queue and delivery` on the main page:
   - `Stop request` cancels the run that is currently in progress.
   - `Remove from queue` cancels a waiting request before it starts.

The daemon serves both the main page and the floating composer. The control surface behind that UI includes `/api/session/*`, `/api/runtime/*`, `/api/audio/*`, `/api/extension/*`, `/api/config/*`, `/api/purge/*`, `/api/fs/*`, and `/api/state`.

## Browser Extension Flow

The Chromium extension popup is the browser-native alternative to the floating composer.

The same delivery-intent controls are available in the main UI and floating composer. The extension side panel submits against the active main-UI session, so choose the delivery intent there before you preview or submit.

Install it first:

1. Open a Chromium-family browser such as Chrome, Brave, Edge, or Arc.
2. Go to `chrome://extensions`.
3. Turn on `Developer mode`.
4. Click `Load unpacked`.
5. Select the repo folder at `extension/chromium`.
6. Pin the extension if you want the popup to stay easy to reach while reviewing pages.

The extension assets ship in-repo under `extension/chromium/`, so there is no separate store install flow in this tree right now.

Use it when:

- you want capture, preview, and submit controls closer to the tab you are reviewing
- you prefer pairing a browser popup once instead of injecting the companion snippet manually
- you want browser notes to stay attached to the same local daemon and current review session

Pair it from the main UI:

1. On the main page in `Capture, review, and send`, use the `Chrome Extension` option and click `Generate extension token`.
2. Copy the one-time code shown right under that button before it expires.
3. Open the Chromium extension popup and enter that code.
4. Use `Settings -> Browser Extension` to confirm the popup is paired or revoke it later.

The pairing happens in two steps:

- the main UI generates a short-lived pairing code
- the extension exchanges that code for its own scoped browser token tied to the local daemon

![Main UI Browser Extension section with Generate pairing code and the active code shown](/docs/assets/browser-extension-pairing-code.png)

![Chromium extension popup showing the daemon URL and pairing code fields before clicking Pair extension](/docs/assets/extension-popup-pairing-compact.png)

Important: the extension does not use the main UI control token directly during normal use. After pairing, it stores a separate browser token returned by the daemon.

What the pairing gives you:

- the extension gets a scoped browser token tied to the local daemon
- after pairing, the popup becomes a thin launcher for the extension side panel
- the full browser composer lives in the side panel for the tab you opened it on: type notes, queue snapshots, record audio, record tab video, preview, and submit against the active main-UI session
- runtime configuration stays in the main UI; the extension remains focused on browser-review capture instead of full daemon administration
- preview and submit still handle approval automatically, so there is no separate approve control in the extension
- after a successful submit, the extension shows a queued-request notice and marks the extension icon with a badge until you reopen the popup or side panel

Side-panel workflow:

1. Pair the popup once.
2. The popup opens the extension side panel automatically for the current tab, or you can reopen it later with `Open browser composer` from that tab.
3. Start the review session from the main Knit UI.
4. Use the side-panel composer buttons for typed notes, snapshots, audio, and video.
5. Refresh the preview in that same side panel, then submit from there.
6. If a queued request in preview is wrong, use the trash-can icon in the upper-right corner of that request card to remove it before submitting.

Snapshot-first flow in the side panel:

1. Click the camera button to capture the current webpage immediately.
2. The side panel queues that snapshot and keeps it ready for the next note.
3. Type your request and press `Cmd+Enter` or `Ctrl+Enter`, or click the keyboard button again to save the typed note.
4. If you prefer voice or a tab recording, click the microphone or video button instead; the queued snapshot is attached to that next note automatically.

Audio and video capture in the side panel:

1. Click the microphone button for a voice note or the video button for a current-tab recording with voice.
2. Chromium may prompt for microphone or tab-capture access; allow it when asked.
3. Click the same button again to stop that recording and attach it to the current review session.

If the extension is already paired but not behaving correctly:

- regenerate the pairing code and pair again if the old token was revoked
- check that the daemon is still the same one the extension was paired to
- reopen the side panel from the exact tab you are reviewing if you switched away and want the tab-bound composer again
- confirm the current review session is active in the main Knit UI before trying to capture or submit from the side panel

## Build And Package

Local builds:

```bash
go build -o ./dist/local/daemon ./cmd/daemon
go build -o ./dist/local/ui ./cmd/ui
go build -o ./dist/local/tray ./cmd/tray
```

Cross-platform matrix:

```bash
./scripts/build-cross-platform.sh
```

Release packaging:

```bash
VERSION=0.1.0 ./scripts/package-release.sh
```

Verification helpers:

```bash
./scripts/runtime-smoke.sh ./dist
./scripts/generate-sbom.sh ./dist/sbom.spdx.json
./scripts/dependency-scan.sh
./scripts/reliability-gate.sh
./scripts/perf-gate.sh
```

The release scripts emit portable archives, platform installer helpers, release metadata, and the npm wrapper scaffold under `packaging/npm/knit-daemon/`.

## Troubleshooting

Refreshing after code changes:

- If you changed daemon/UI code, restart the daemon to pick up the new build.
- `Quit Tray` closes only the tray UI; it does not refresh the detached daemon.
- Direct daemon mode: stop the current process and run `go run ./cmd/daemon` again.
- Tray mode: stop the daemon that is still listening on `127.0.0.1:7777`, then run `go run ./cmd/tray` again.
- To check which process still owns the daemon port:

```bash
lsof -iTCP:7777 -sTCP:LISTEN
```

- If the tray opens but the app does not load, check `daemon.log` and `tray.log` under `KNIT_DATA_DIR`.
- If the tray restarts but you still see old behavior, the detached daemon is probably still running. Check `lsof -iTCP:7777 -sTCP:LISTEN`, stop that process, then relaunch `go run ./cmd/tray`.
- If submit preview works but submit fails, verify the selected adapter command or remote credentials first.
- If screenshots or clips are missing, confirm the browser companion or extension is paired and the OS screen-capture permission is available for your platform profile.
- If voice notes fail, use the transcription health check in the UI and verify the active mode-specific environment variables.
