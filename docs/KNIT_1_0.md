# Knit 1.0: Browser Feedback in Your Codex Conversation

Status (October 16): the user accepts repeated use with the persistent Codex connection panel open. Real screenshot/replay feedback, source edits, tests, live reload and completion have been observed. Background delivery is deferred. Native audio/video interpretation, public distribution eligibility and remaining safety/recovery gates are incomplete. This document is a release gate, not a statement of shipped capabilities.

See [local verification evidence and blockers](KNIT_1_0_VERIFICATION.md) for current results.

## User Experience

1. Install Knit in the desktop plugin directory. The integration starts or attaches to the local daemon without terminal commands.
2. Choose Connect browser in the plugin and enter a short-lived code in the Chrome extension. Normal onboarding never exposes a raw secret.
3. Connect this conversation from the desktop conversation panel. The extension shows its title and project as the destination.
4. Capture, preview, and send feedback from the browser. The extension remembers the explicit destination.

Never choose a destination from whichever window is foreground. A conversation change, archived destination, project mismatch, or expired binding requires user attention. Do not reroute silently.

## Ownership

| Component | Owns |
| --- | --- |
| Browser extension | Capture consent, preview, selected destination, request submission |
| Local daemon | Existing authentication, encrypted artifacts, immutable approved requests, durable delivery records |
| Desktop plugin | Host capability detection, explicit conversation binding, host-mediated message submission |
| Codex | Conversation history, workspace, permissions, active execution, approvals, progress and final response |

Reuse the existing daemon services and adapter boundaries. Add connected-session routing explicitly; keep direct_exec and mcp_worker available. A connected-session request must never launch a CLI fallback.

## Host Interface Evidence

Official plugin extensions document conversation panel entrypoints and extended message sending. The component bridge reference documents sendFollowUpMessage({prompt, scrollToBottom}). New UI should prefer the MCP Apps bridge when equivalent capabilities exist.

This establishes a candidate current-conversation path, not arbitrary thread discovery, project enumeration, automatic background wake-up, Codex-specific support, or reliable result subscriptions. Each must be tested in the installed Codex surface. Environment-provided codex-app-tools are not a public dependency for Knit.

Sources checked October 15, 2026:

- https://developers.openai.com/plugins/build/extensions
- https://developers.openai.com/plugins/reference#windowopenai-component-bridge
- https://developers.openai.com/plugins/build/plugins
- https://developers.openai.com/plugins/deploy/connect-chatgpt

## Delivery Contract

An immutable approved request contains request_id, destination binding revision, canonical feedback, media manifest, and creation time. A binding contains connector, daemon instance, opaque host conversation handle, user-visible title, workspace/host identity, capability set, expiry, and explicit consent time. Never accept an arbitrary browser-supplied conversation ID as sufficient authorization.

The plugin obtains queued feedback through authenticated daemon operations, verifies the binding, and submits through the host-supported bridge. Model tools alone cannot wake an idle conversation. Automatic delivery requires a verified host capability and user opt-in; otherwise show that the desktop panel must be opened and Send selected. That limited mode does not pass the automatic-delivery release gate.

Persist a delivery ID before calling the host. Use host idempotency where supported. If a connection drops after sending but before acknowledgement, mark delivery uncertain; reconcile against host evidence before retrying. Do not promise exactly-once execution without host support.

States: Waiting for Codex -> Sending -> Delivered -> Working -> Completed, with Needs attention available at every stage. Host bridge resolution proves only host acceptance. Working and Completed require correlated host execution events/results. Closing the panel or losing the connection retains queued requests. Store summaries and results with request and host turn IDs; never infer success from time elapsed.

Busy policy defaults to queue-after-current when supported. Steering or interruption requires an explicit user choice and verified host support. Preserve desktop permissions; Knit must not override sandbox, model, workspace, or approval policy during delivery.

## Security and Lifecycle

Reuse daemon-owned extension pairing and scoped credentials. No raw tokens in state/status, plugin tools, extension session payloads, or ordinary logs. Secrets remain encrypted at rest. No broad CORS expansion, public listener, or browser access to desktop control APIs. Plugin-to-daemon transport and host sandbox reachability must be proven before selecting distribution architecture.

Attach/start ownership follows existing MCP rules: stop only a daemon the integration started. Multiple plugin instances need shared ownership or a supervisor before any one instance can stop a shared daemon. Uninstall/revoke invalidates bindings; requests remain visible and require explicit retargeting.

## Release Acceptance Matrix

| ID | Acceptance | Implementation | Validation |
| --- | --- | --- | --- |
| K1-01 | Install, pair, connect without source builds/manual config edits | Prebuilt private-pilot package, macOS installer, short-code pairing | Fresh isolated Codex install and actual MCP tool call pass; CLI/Python/FFmpeg prerequisites explicit; unassisted fresh-machine UI trial pending |
| K1-02 | Explicit conversation/project binding visible in browser | User-named panel and intended workspace | API and browser tests; host identity verification pending |
| K1-03 | Browser feedback reaches selected existing Codex conversation | Approved feedback with scoped capture retrieval | Native screenshot/replay demo cycles observed; automated daemon/browser/MCP tests pass; native audio/video interpretation pending |
| K1-04 | Idle delivery, busy policy, permissions preserved | Opt-in serial delivery with persistent conversation panel visible; background wake-up deferred | Repeated-use UX accepted by user; automated serial/no-reconnect contract passes; native busy signal remains unverified |
| K1-05 | Durable queue, reconnect reconciliation, duplicate prevention | Encrypted queue, atomic preparation, no uncertain-send replay | Persistence, concurrency, storage failure and browser tests; host reconciliation pending |
| K1-06 | Distinct acceptance/execution/completion states | Explicit delivered/working/completed/failed reports | Late-acknowledgement race tested; independent host event correlation pending |
| K1-07 | Pairing/auth/revocation and token non-disclosure | Existing pairing reused; extension cannot control panel | Scoped credential browser test and MCP protocol checks; uninstall revocation pending |
| K1-08 | Direct-exec and MCP regression safety | Additive endpoints/tools; explicit desktop profile | Full Go, reliability/race, performance, patched five-target build and macOS smoke pass October 16; native Linux/Windows runtime trials pending |
| K1-09 | Primary desktop/browser workflows automated end to end | Actual extension assets + HTTP daemon + panel harness | Playwright contract covered; actual desktop host is still injected |

Do not tag 1.0 until every row has executable evidence. Required real-host scenarios: install, pair, bind, capture text and media, approve, deliver, observe acknowledgement/response, send while busy, reconnect, duplicate submission, revoke, wrong workspace, host unavailable, and existing-mode regressions. Record host version, plugin version, request/turn IDs, timestamps, and redacted results.

## Delivery Spike and Sequence

The isolated prototype in prototypes/codex-desktop exercises the documented component bridge shape with an injected host. A user-assisted installed-host test also passed: the panel sent a unique marker and Codex returned it in the same conversation. Neither proves automatic queue pickup.

The current slice is available as [Knit Browser Feedback Preview](../plugins/knit-desktop/README.md). It uses the main product binary, a separate development daemon instance, authenticated panel APIs, durable encrypted delivery, and the actual extension's destination selection. It does not discover a real host conversation ID: the user explicitly names the connection and workspace. Results are reported by the agent, not independently subscribed host events. Snapshots use MCP image content; audio/video use embedded media resources. An app-only consent tool enables serial automatic delivery while the connection is visible/open, never a hidden model action or CLI fallback. Capture access is restricted to the active binding's delivered/working request and indexed approved capture; arbitrary paths are not accepted. Eight captures, 10 MB each, 24 MB total; capture reads are audited without logging content.

Next: verify this complete text workflow in the installed desktop app, then establish host identity, reconnect reconciliation, automatic pickup and busy behavior, multi-bridge daemon lifetime, media delivery, and production onboarding. Claude Desktop uses a separate capability-verified connector after the Codex path works.

## Design Gate

Target at least 4/5 for hierarchy, whitespace, simplicity, warmth, comprehension, copy, accessibility, consistency, responsiveness, and polish. The preview panel has been rendered at 360px and reviewed; production onboarding and actual host accessibility review remain release gates. See the verification record for provisional scores.

Ease of use is a separate mandatory 4/5 minimum, with bonus points for a fresh install needing no terminal, no config edits, no raw-token handling, no repeated pairing, and no repeated destination selection. Bonus points cannot compensate for failed correctness, security, accessibility, or host-delivery gates. Measure setup steps, time to first delivered feedback, recovery steps, and an unassisted first-use trial.

Public local-MCP plugin distribution requires approved-partner eligibility according to https://developers.openai.com/plugins/deploy/desktop-only. Local marketplace testing does not establish public release eligibility.
