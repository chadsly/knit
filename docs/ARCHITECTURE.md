# Architecture

`Knit` is a local-first feedback capture system. The executable surface is small, but the daemon coordinates capture state, local persistence, packaging, audit, and submission to agent adapters.

## Entry Points

- `cmd/daemon`
  Starts the main application, configures process logging, constructs `internal/app.App`, and runs the local HTTP server until shutdown.
- `cmd/tray`
  Acts as a native controller around the daemon. It ensures the daemon is running, then drives pause, resume, kill-capture, and UI-open actions over the local HTTP API.
- `cmd/ui`
  Placeholder binary; there is no separate native webview runtime in this tree.

## Application Assembly

`internal/app` is the composition root:

- resolves environment-backed config
- validates startup integrity and managed version constraints
- initializes encryption and the SQLite-backed store
- restores persisted operator state and session history
- constructs the capture broker, session service, agent registry, audit logger, transcription provider, retention worker, and HTTP server

The resulting process is a single local daemon rather than a distributed service mesh.

## Main Subsystems

- `internal/server`
  Owns the UI HTML, floating composer UI, companion script payload, and the local HTTP API.
- `internal/session`
  Tracks session lifecycle, feedback events, approval state, and canonical package generation.
- `internal/capture`
  Tracks capture state and source health.
- `internal/companion`
  Maintains browser interaction evidence and replay timeline state.
- `internal/audio`
  Handles audio device/config state and level reporting.
- `internal/transcription`
  Abstracts remote, local-command, LM Studio, and managed faster-whisper STT paths.
- `internal/agents`
  Provides pluggable submit adapters and payload-preview generation.
- `internal/storage`
  Persists encrypted structured state and encrypted artifacts.
- `internal/security`
  Wraps key resolution, encryption helpers, startup integrity checks, and outbound HTTP client policy.
- `internal/audit`
  Writes encrypted, hash-chained audit events and optional SIEM mirrors.
- `internal/retention`
  Purges retained artifacts and structured records by policy.
- `internal/platform`
  Reports OS profile, packaging hints, permission model, and auto-start support.
- `internal/privileged`
  Maintains the boundary between raw capture/audio facilities and the HTTP/UI layer.

## Local HTTP Surface

The daemon registers the UI and local control plane in `internal/server/http.go`.

UI routes:

- `/`
- `/floating-composer`
- `/companion.js`
- `/healthz`

Operational API groups:

- `/api/state`
- `/api/session/*`
- `/api/audio/*`
- `/api/capture/*`
- `/api/companion/*`
- `/api/runtime/codex*`
- `/api/runtime/transcription*`
- `/api/extension/*`
- `/api/config/*`
- `/api/purge/*`
- `/api/audit/export`
- `/api/fs/*`

This API is intended for the local UI, tray controller, and paired browser extension. It is not a multi-tenant network service.

## Persistence Model

The daemon stores state under `KNIT_DATA_DIR`:

- SQLite state database from `internal/storage/sqlite.go`
- encrypted artifacts under the artifact store directory
- submit queue recovery file
- process logs such as `daemon.log` and `tray.log`

Persisted state includes:

- operator runtime settings
- sessions and feedback events
- approved canonical packages
- submission attempts and queue recovery metadata
- audit records

## Security Model

The security model is local-control first:

- local control APIs can require `X-Knit-Token` or bearer auth
- mutating requests enforce nonce and timestamp replay protection
- capabilities constrain read, capture, submit, config, and purge actions
- artifacts and structured state are encrypted at rest
- startup integrity hooks can verify binary hashes, checksum files, signatures, and release manifests
- outbound providers inherit allowlist, blocklist, proxy, CA, and pinning policy from `internal/security/httpclient.go`

## Packaging Model

Packaging is script-driven rather than embedded in the daemon:

- `scripts/build-cross-platform.sh` builds host binaries
- `scripts/package-release.sh` produces portable archives and installer-helper wrappers
- `scripts/runtime-smoke.sh`, `scripts/reliability-gate.sh`, `scripts/perf-gate.sh`, and `scripts/dependency-scan.sh` validate release quality
- `packaging/npm/knit-daemon/` contains the npm wrapper scaffold that installs host-specific packaged binaries

## Current Boundary

This repo currently centers on browser-first capture and local orchestration. It does not implement a separate cloud backend, native packaged installers with platform-specific installers, or a standalone desktop UI process beyond the tray controller and daemon-served web UI.
