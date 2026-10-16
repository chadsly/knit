# Knit 1.0 Verification Record

Date: October 15, 2026. Overall release status: NOT READY.

## Executed

### Recovery, Private Pilot And Security (October 16)

User reports successful native video review with the new frame package and
accepts persistent-panel operation. This is user-supplied host evidence, not
an independent recording review. Closed/background delivery remains deferred.

Recovery checks now include encrypted SQLite close/reopen with saved binding,
automatic-delivery consent, queued input and idempotent request IDs; uncertain
send no-replay; revoked-binding rejection. Playwright verifies delayed widget
restoration, daemon outage/recovery without changing identity, duplicate submit,
uncertain-send handling and visible wrong-workspace/read-only/no-input/media
failure explanations across surfaces. Native read-only/workspace failure reports
are not synthesized by an actual model in these automated tests.

Security: golang.org/x/sys upgraded to v0.44.0 for GO-2026-5024. A Windows-target
scan then exposed Go 1.25.0 standard-library advisories. go.mod selects toolchain
go1.26.9 and GitHub workflows explicitly select 1.26.9. Afterward, actual
govulncheck scans for darwin_arm64 and windows_amd64 both report No vulnerabilities
found. Full Go, reliability/race and revised Playwright checks pass. Five-target
binaries rebuilt with the patched toolchain; prior rc.1/rc.2 artifacts are obsolete.

Private pilot was explicitly chosen instead of public-directory distribution.
Package generator includes only native binary, manifests, license, instructions,
platform metadata and installer. No source/runtime database/token is copied.
An isolated CODEX_HOME test installs through the actual Codex CLI, resolves a
legacy placeholder incompatibility automatically, invokes knit_status through
Codex's MCP client, discovers desktop tools, and verifies owned-daemon shutdown.
Separate stdio protocol verification passes from the relocated packaged binary.
The pilot requires official Codex CLI/Python; FFmpeg is required for video.
The macOS double-click installer still opens a command window, but needs no
typed config edits or source build. Linux/Windows runtime installation and
unassisted fresh-machine desktop UI onboarding remain unverified.

Current local pilot packages are under
/tmp/knit-private-pilot-verified-20261016 (validation label 1.0.0-rc.3).
No tag, commit, push, public upload, signing or publishing was performed.
See PRIVATE_PILOT.md for prerequisites and the explicit development-profile
separation. Public directory approval is not required for this local pilot but
remains necessary before public local-MCP distribution.
### Video Review Package (October 16)

Native failure confirmed: request 552b2987-c323-4508-82b5-16a1947ceb76 reached
Codex, but its embedded WebM could not be interpreted. Plugin 0.1.10 requests
daemon-derived video samples instead: up to six JPEG images, approximate offsets
0 through 25 seconds at five-second intervals, with the existing approved request
transcript as narration. The original encrypted artifact remains unchanged.
Derived reads use the existing binding/delivery/capture-index authorization;
no browser-supplied paths or URLs, no plaintext temporary recordings, no new
endpoint or auth scheme. Legacy raw capture transport remains available.
FFmpeg is a new explicit daemon-host prerequisite for video review, not installed
automatically. Protocols are restricted to pipes; extraction deadline 30 seconds,
six images at most, dimensions bounded to 960x540, 2 MB per frame/12 MB total.
Failures are explicit and the agent must not claim unsampled footage was viewed.

Tests cover real WebM decoding/timestamps/JPEG validity, invalid recordings,
missing FFmpeg, cancellation, scoped HTTP review reads with original bytes retained,
and MCP native-image output. Browser delivery regression remains covered.
Actual installed Codex interpretation of the new frame package remains pending
restart and a fresh video request; transport tests do not certify that native step.
### Local Transcription Compatibility Repair (October 16)

Actual installed environment had faster-whisper 1.2.1 with PyAV 19.0.1.
The decoder passes metadata_errors to av.open, which PyAV 19 removed.
Managed bootstrap/repair now constrains av>=11,<19 and checks compatibility,
not just whether faster_whisper imports. Auto-install disabled still fails
without changing packages. Automated bootstrap and existing-runtime repair tests
pass. Local runtime repaired to PyAV 18.1.0; actual managed script transcribed
a generated AIFF phrase exactly: The quick brown fox jumps over the lazy dog.
No private user recording was inspected. This verifies local speech-to-text,
not native Codex audio/video interpretation. Extension 0.1.11 places the Enable
microphone recovery button directly after its status message. Plugin 0.1.9
includes the managed-runtime repair for future starts.
### Microphone Permission Recovery (October 16)

User could not record audio: getUserMedia denial was incorrectly described as
a dismissed prompt. Extension 0.1.10 adds an Enable microphone recovery action
that opens an extension tab with a user-clicked audio-only permission check.
Tracks are stopped immediately and no recording is made by the setup page.
Chrome and macOS settings instructions cover blocked access. No permissions
are silently granted or added to the manifest. Node tests cover no access on
load, success/track cleanup and denial/retry; Playwright covers a denied recorder
request exposing the setup action and opening the scoped extension URL.
Native permission granting and an actual recorded note still require verification.

### Release Hardening Pass (October 16)

User accepts repeated-use behavior with the persistent conversation panel open;
background delivery is deferred. Extension 0.1.9 and plugin 0.1.8 now show failed
and needs-attention explanations directly, not only in hover tooltips. Panel
errors use alert semantics. TestDesktopBrowserWorkflow reports wrong-workspace,
read-only, no-input and unsupported-media failure fixtures through real daemon
state and asserts visible explanations in the extension, panel and Main UI.
These are failure propagation tests with an injected host, not proof that native
Codex emitted each failure. Existing API tests enforce workspace and media
rejection without consuming approved feedback; pairing tests cover network loss.

Executed locally: go test ./..., reliability gate (including race detector),
performance gate, revised Playwright workflow, desktop-connect Node tests,
main-binary MCP verification, five-target cross-platform builds, npm/Python
packaging, unsigned release-readiness checks, isolated npm and Python wheel
installs and version commands, and darwin_arm64 runtime smoke. All passed.
Local validation artifacts: /tmp/knit-1.0-validation-20261016, labelled
1.0.0-rc.1 solely for packaging tests; Python normalization produces 1.0.0.dev0.
Nothing published or tagged. SBOM generator used its module-inventory fallback
because syft is unavailable. Production signing/provenance and OS-native runtime
checks outside macOS are not certified by these results.

govulncheck latest passed symbol/package reachability on darwin_arm64 using its
automatically selected Go 1.26.9 toolchain. Module advisory GO-2026-5024 affects
golang.org/x/sys v0.40.0 Windows NewNTUnicodeString; fixed in v0.44.0. Windows
reachability is not established by this macOS scan and needs review before the
Windows release. No dependency upgrade was silently bundled into this UI pass.

Media: real native screenshots/replay contexts worked in the recorded demos.
Automated API/MCP tests cover audio/video bytes, scope and capture bounds.
Actual native audio/video comprehension is still unverified, not a passed gate.
Public terminal-free plugin installation is blocked on local-MCP distribution
eligibility: the official desktop-only submission documentation requires
approved-partner organization allowlisting. Development plugin installation and
clean npm/Python installation do not establish public directory eligibility.

### Off-Screen Card Delivery (October 16)

Plugin 0.1.7 follow-up: PiP confirmation now waits up to two seconds for a host
display-mode notification instead of assuming the API acknowledgement is the
mode change. An explicit Keep connection visible click tries fullscreen if PiP
is rejected or unconfirmed. Both attempts are bounded; neither changes binding,
consent, or routing. Playwright covers delayed PiP, rejected modes, absent API,
and fullscreen fallback. Installed-host acceptance and scrolling remain pending.

Native user observation: opening the conversation is insufficient; scrolling
the connection card into view resumes delivery. Treat off-screen card suspension
as a host limitation, not a disconnected daemon. Plugin 0.1.6 advertises inline,
PiP, and fullscreen presentation and adds an explicit Keep connection visible
action using the documented requestDisplayMode API. Missing/rejected APIs retain
the inline path with honest on-screen guidance. Follow-up sends no longer request
automatic scrolling to the bottom, which previously displaced their own card.

Automated coverage verifies resource display-mode metadata, optional API absence,
host rejection, accepted PiP requests, and scrollToBottom=false, alongside the
existing consecutive-delivery workflow. These injected-host checks do not prove
that the installed Codex build supports persistent PiP or avoids suspension.
Native scrolling verification remains pending. No background execution, new
permissions, CLI fallback, or automatic-delivery consent changes are introduced.

### Consecutive Request Connection Recovery (October 16)

The recorded native demo completed headline, paragraph, button-color, and label
changes through real extension submissions, scoped capture reads, native Codex
edits, tests, and live browser updates. However, the user had to press Connect
between requests. This is not a pass for unattended consecutive delivery.

Plugin 0.1.5 repairs two panel lifecycle gaps: saved widget state may arrive after
initial bridge setup, and cached page restoration must restart polling after
pagehide. Connection state is also saved before sending a follow-up. Restoration
uses only this widget's saved binding, never a guessed daemon destination; pairing
and daemon-owned automatic-delivery consent are unchanged.

TestDesktopBrowserWorkflow passes with a real daemon and injected desktop host:
remount with delayed widget state, restore the same binding, pagehide/pageshow,
and two further extension submissions without Connect, Refresh, or Send clicks.
The test asserts one binding, no duplicate sends, completed results, and retains
uncertain-send/no-replay and explicit-disconnect coverage. Native Codex retesting
after plugin reload remains required; the injected-host test cannot certify host
widget-state persistence. No release requirement is marked complete by this fix.

### Desktop Feedback Preview

Live recording attempt (October 16): the actual Chrome extension captured and
previewed the Field Notes screenshot, but submission was rejected with
`desktop_submission_blocked` / `This capture type is not supported.` No desktop
delivery or source edit occurred. Recording should be stopped for repair, not
represented as a successful cycle.

Root cause: the real browser pointer context produces `replay_json` and
`playwright_script` exports during HTTP approval. The original desktop media
validator rejected those generated artifacts. Plugin 0.1.4 accepts validated
JSON/UTF-8 replay context as read-only text resources alongside media. The
regression test now uses the actual HTTP approval path with replay steps and
checks all three attachments. Playwright now posts a real companion pointer
event before note capture, retrieves image/JSON/script through daemon APIs,
and verifies encrypted-file paths are absent from the outgoing message.
MCP protocol tests cover text resources as well as image/audio/video transport.
This repair still needs a new real desktop take; no successful live cycle is
claimed. The demo button remains unchanged until an actual Knit request arrives.

Latest October 16 development slice: automatic delivery is explicitly opt-in
through an app-only tool and persists in daemon binding state. Playwright enables
it before browser submission, then verifies delivery, capture retrieval, and
completion without clicking Refresh or Send in the Codex connection. The host
remains injected. Screenshot upload uses the real note endpoint and encrypted
artifact store; approved capture retrieval uses authenticated daemon APIs.
Separate real MCP client/server tests verify native image content and embedded
audio/video resource bytes, plus denial of model-facing automatic-delivery
settings. API/service tests cover scope, pre-delivery reads, revocation, missing
captures, per-capture size and count limits, and retained approval on rejection.
The reliability gate passes. This does not establish actual Codex audio/video
comprehension or closed-conversation wake-up. Those remain release gates.

Source versions: extension 0.1.7; installed development plugin 0.1.3. Earlier
text-only statements below describe earlier tested revisions, not current code.

Latest review-connection render inspected at 360px: hierarchy 4, whitespace 4,
simplicity 4, warmth 4, comprehension 4, copy 4, accessibility 4, consistency 4,
responsiveness 4, polish 4. Automatic sending requires an unchecked-by-default
choice; the connection-open limitation is explicit. Production ease-of-use and
native-host media checks are still pending.

October 16 follow-up: the user's screenshot confirms real installed extension
pairing and an active session with the selected conversation. It also exposed
an unnecessary second code-generation click and setup controls still visible
after connection. The revised panel auto-generates the code; the extension hides
setup after a verified binding and provides End review/Start text review while
retaining pairing. Playwright covers automatic code generation (without clicking
Connect browser), hidden setup, end/restart state, and the complete prior delivery
contract. Plugin 0.1.2 and extension source 0.1.6 distinguish these revisions.
At the user's explicit request the obsolete diagnostic Probe was uninstalled;
its marketplace listing was removed, leaving only the real Preview integration.
Live feedback execution in Codex remains a separate pending gate.

Connected-state visual review at 360px: hierarchy 4, whitespace 4, simplicity 4,
warmth 4, comprehension 4, copy 4, accessibility 4, consistency 4,
responsiveness 4, polish 4. Setup and the duplicate icon-only stop control are
hidden for verified desktop connections; End review remains labeled. These are
provisional rendered-review scores, not an installed-host accessibility audit.

October 16 onboarding follow-up: the user-assisted screenshots proved installed
panel rendering, connection, and short-code generation, but exposed a failed
onboarding flow (manual port, repeated workspace setup, separate popup pairing,
and an unreachable extension). This did not meet the ease-of-use gate.

The revised implementation supplies one expiring connection code from the
Codex panel, inline extension confirmation/recovery, remembered destination,
and review startup without visiting the Main UI. It preserves active reviews
and rejects replacement of a different existing workspace. Four Node tests
cover parsing/security, pairing, active-review preservation, and failure paths.
`TestDesktopWorkspaceOnboardingPreservesExistingSettings` covers workspace
setup, persistence, configuration locks, and permission preservation. The
Playwright workflow now begins with no review or configured workspace and uses
the actual new extension confirmation controls. Passed locally. The host bridge
remains injected; the revised onboarding has not yet passed an unassisted live
desktop/installed-extension trial. No ease-of-use bonus is awarded yet.

Rendered revised setup at 360px and inspected it locally: hierarchy 4,
whitespace 4, simplicity 4, warmth 4, comprehension 4, copy 4, accessibility 4,
consistency 4, responsiveness 4, polish 4. These provisional scores do not
replace an installed-host accessibility or unassisted onboarding trial.

- Implemented daemon-owned encrypted text-delivery queue, explicit named panel/workspace binding, disconnect handling, structured retention, and atomic preparation before host delivery.
- Session approval is consumed only after the durable queue write succeeds. Workspace mismatch, missing input, media rejection, write failure, restart, duplicate request IDs, concurrent preparation, and late host acknowledgement have automated coverage.
- Actual extension recorder assets run in Playwright against an isolated real HTTP daemon. Chrome platform APIs are shimmed, but scoped pairing, approval, queue submission, polling, and main-UI result retrieval use real backend endpoints. The desktop host message bridge is injected. Passed.
- Browser tests cover absent bridge, completed result in panel/extension/main UI, deep-linked request details, host rejection, no automatic replay, disconnect, no page errors, and 360px layout. An initial main-UI polling omission was caught by this test and fixed before the passing run.
- Main product binary MCP stdio test passes: tools/resource discovery, daemon-backed panel connect/heartbeat, short-code pairing, token absent from stdout, and owned-daemon shutdown.
- Installed `knit-desktop@knit-development` version 0.1.0 locally. A fresh read-only diagnostic runtime using the actual bundled desktop CLI exposes its tools/resource with `toolsError=null`. No agent turn was started by this diagnostic. This is loader evidence, not a live desktop feedback test.
- `go test ./...` passed. `bash scripts/reliability-gate.sh` passed, including its server race detector. Re-run after later changes before release.
- Preview-only rendered design review at 360px: hierarchy 4, whitespace 4, simplicity 4, warmth 4, comprehension 4, copy 4, accessibility 4, consistency 4, responsiveness 4, polish 4. This is a manual visual/DOM review, not a screen-reader audit. Installed-host review and production onboarding ease-of-use scoring remain pending; no ease-of-use bonus awarded yet.

Reproduce (Playwright must be installed and available on NODE_PATH; Chrome path can be overridden with CHROME_PATH):

```sh
KNIT_DESKTOP_PLAYWRIGHT=1 go test ./internal/server -run TestDesktop -v -count=1
node --test test/desktop-connect.test.cjs
node plugins/knit-desktop/setup.cjs
node plugins/knit-desktop/verify.cjs
KNIT_DIAGNOSTIC_SERVER=knit_desktop python3 prototypes/codex-desktop/check-runtime.py
```

The installer uses isolated preview data and port 7778. See [preview setup and limits](../plugins/knit-desktop/README.md). Still pending: actual installed-host feedback/result round trip, verified host identity, background/busy policy, media, shared daemon lifetime across multiple bridge instances, uninstall revocation, terminal-free distribution, and unassisted onboarding.

### Initial Marker Probe

- Built the local conversation-panel probe using the existing Go MCP SDK.
- Added the repo's Knit Development marketplace and installed knit-desktop-probe@knit-development through the installed Codex CLI's plugin management command. Installation and user-assisted desktop activation are verified.
- Exercised the installed binary over real MCP stdio: initialize, tools/list, tools/call, resources/read. Passed.
- Rendered the returned panel resource in isolated Chrome through Playwright at 360px width. Passed with no page errors.
- Tested absent host bridge (sending disabled), injected host acceptance (single message, no completion claim), and rejected delivery (visible error). Passed. This bridge was injected by the test, not supplied by Codex.
- Ran three Node contract tests. Passed.
- Compiled the Go probe package. Passed.
- Inspected the rendered small-screen panel. Probe-only review: hierarchy 4, whitespace 4, simplicity 5, warmth 4, comprehension 4, copy 4, accessibility 4, consistency 4, responsiveness 4, polish 4, ease of use 4. Production onboarding and installed-host accessibility review remain unscored.

## Commands

```sh
go build -o prototypes/codex-desktop/bin/knit-desktop-probe ./prototypes/codex-desktop/server
node --test prototypes/codex-desktop/delivery.test.mjs
node prototypes/codex-desktop/verify.cjs
go test ./prototypes/codex-desktop/server
```

The browser test needs Playwright on NODE_PATH and a Chrome executable; see the prototype README.

## Live Desktop Verification

User-supplied transcript confirms the installed panel opened and sent a test message at 1:47 PM America/Chicago on October 15, 2026. The Codex conversation received `KNIT_DESKTOP_PROBE_2fda5c55-e7f3-4027-bbaf-5014407d14dc` and the agent returned exactly that marker in the same conversation. This passes the user-assisted current-conversation panel/message/response gate. The prompt requested no tools or file edits; the supplied transcript shows no tool calls during the marker response. Independent permission snapshots were not captured, so broader permission preservation is not certified by this test.

This is actual installed-host evidence supplied by the user, separate from the injected-host Playwright tests. It proves a user-clicked current-conversation send, not autonomous delivery from the browser or arbitrary-conversation routing.

## Earlier Diagnostics

Plugin-loading repair: the initial portable layout did not expose an MCP server in this installation. The subsequent compatibility configuration had an unterminated cwd string and then unresolved executable placeholders. Version 0.0.4 uses an explicit compatibility manifest and generated absolute paths. Four Node tests and the installed-binary Playwright verification pass. A fresh diagnostic app-server using the exact bundled desktop CLI now reports knit_probe with open_connection_check, its resource, and toolsError=null. This verifies actual runtime loading, but not panel rendering in the running desktop app or message delivery to its conversation.

The installed desktop app is running, but exposes no app-server proxy control socket. No CLI execution substitute is used. Native automation of the desktop app was explicitly denied by the computer-use environment; it cannot perform the panel activation/click. A local plugin install does not establish that a running conversation has reloaded the plugin or exposes the documented component bridge.

The formerly pending user-assisted marker test is now passed as described above.

No end-to-end browser-feedback-to-Codex success has been claimed. Pairing, durable destination binding, automatic delivery while idle/busy, response correlation, media delivery, and public distribution approval remain open gates. Ease-of-use bonus points are pending an unassisted production onboarding trial.
