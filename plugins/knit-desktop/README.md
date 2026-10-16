# Knit Browser Feedback Preview

For compiled private-pilot installation, see [Private Pilot](../../docs/PRIVATE_PILOT.md).
The development installer below still builds from source; it is not the pilot
user installation path.

Video feedback is delivered as up to six timestamped JPEG samples near 0, 5,
10, 15, 20 and 25 seconds, plus the existing approved narration transcript.
The original video stays in the encrypted artifact store. The daemon requires
FFmpeg on PATH (available on this development Mac). Missing decoder, corrupt
recordings and extraction failures produce a visible error, not silent success.
Processing uses in-memory pipes, no plaintext temporary video files, network
protocols disabled, a 30-second deadline and bounded images up to 960x540.
Samples can miss short interactions or later footage; use a snapshot or precise
text when needed. This is not native Codex video playback or audio interpretation.

This development plugin uses the normal Knit binary and daemon-owned queue,
not a separate agent or Codex CLI execution adapter. It is not the 1.0 release.

If microphone recording reports denied access, select **Enable microphone** in
the extension. In the setup tab, select **Enable microphone** and allow Chrome's
prompt. The setup check stops immediately without recording. Return to the target
page's Knit side panel to start recording. If blocked, check Chrome microphone
settings and macOS Privacy & Security > Microphone > Google Chrome. Granting
browser access does not establish native Codex audio interpretation support.

Managed local speech-to-text currently uses PyAV >=11 and <19 for compatibility
with faster-whisper 1.2.x. Startup checks this constraint and repairs it only
when managed auto-install is enabled; otherwise the health error remains visible.

## Install Locally

Register this repository's Knit Development marketplace if not already registered:

```sh
codex plugin marketplace add /absolute/path/to/knit
node plugins/knit-desktop/setup.cjs
```

The setup builds the main Knit binary and installs the plugin. Generated launch
paths are machine-local. Preview data stays under this plugin's ignored `runtime`
directory. It uses port 7778, separate from the normal Knit daemon on 7777. An
incompatible process using that port causes a clear failure; it is never replaced.
Removing this source checkout also removes the executable path; this is not a
portable production distribution.

## Use

1. Enable **Knit Browser Feedback Preview** in the intended Codex conversation.
2. Ask it to **open Knit browser feedback**. Name the conversation and enter its
   absolute workspace folder. Knit cannot independently discover or verify the
   desktop conversation's project yet.
3. Connecting the panel automatically generates a code. Select **Copy connection code**. In the extension
   side panel expand **Connect to Codex**, paste the code, and select **Connect
   and start review**. Confirm the displayed conversation name before
   connecting. One short-lived code carries the local address and destination;
   no port entry, popup pairing, or separate Main UI setup is needed.
4. The extension remembers the connection and selected conversation. The panel
   configures an empty daemon workspace from your explicit folder choice; it
   never replaces a different existing workspace. Browser confirmation starts
   a review only when no review exists or the old review was stopped. It does
   not reset an active review or turn on microphone/camera capture.
   Once connected, setup is hidden. Use **End review** to stop capture while
   keeping the pairing, then **Start review** when you want to resume.
   **Connect browser** in the desktop panel is only needed to generate another
   code for a new browser or recover from an expired code.
5. In your Codex review connection, optionally enable **Automatically deliver
   my approved browser submissions while this connection is open**. This is
   off until you explicitly choose it. Capture, preview, and submit in the
   browser. Approved requests deliver one at a time without another click while
   the connection remains visible and open. Otherwise they queue for manual
   delivery using **Send to this conversation**. No CLI agent starts.
   Connecting is not a per-request step. The panel restores its own saved
   conversation connection when Codex remounts it and resumes polling when
   reopened. Keep the existing panel visible for consecutive requests; do not
   create another connection for each correction. Inline cards may pause when
   scrolled off screen, even in an open conversation. Select **Keep connection
   visible** to request Codex picture-in-picture presentation, with an expanded
   fullscreen fallback if PiP is unavailable. Delayed host confirmation is allowed.
   Keep that floating connection open while scrolling. If the host rejects the
   request or lacks this API, keep the inline card on screen. This is not
   closed-conversation background delivery. If Codex does not restore
   widget state, delivery stays queued rather than guessing a destination.
6. Codex receives the feedback in this conversation with instructions to check
   the workspace and report progress and a factual result summary. Native Codex
   permissions still apply. Delivery alone is never marked completed.

Snapshots now reach Codex as native MCP images through `desktop_read_capture`.
Raw audio travels as an embedded media resource alongside its approved
transcription; video is converted into native images. Native audio interpretation must still be
verified in the installed Codex host. The agent must report a failure if it
cannot review a capture; it must not silently omit media or claim success.
Up to eight captures, 10 MB each, and 24 MB combined are allowed per request.
Missing or oversized captures block submission without consuming approval.
Captures remain encrypted at rest; no decrypted temporary files are written.

No automatic closed-conversation/background wake-up is promised. Keep the originating
conversation open: the MCP bridge instance currently owns its child daemon's
lifetime. Multiple simultaneous desktop conversations remain a release gate.

The automatic-delivery choice is daemon-owned and saved through an app-only
tool. The model-facing action tool cannot enable it. An unresolved send or run
blocks the next automatic delivery; uncertain sends are not retried. This is
serial delivery based on Knit results, not a verified native Codex busy signal.

Disconnected destinations never silently fall back to CLI execution. If a send
has an uncertain acknowledgement, inspect the conversation; Knit will not resend
automatically. Structured feedback follows configured structured-data retention.

If the extension cannot reach Knit, **Connect to Codex** opens with recovery
instructions. Reopen the Codex panel and copy a fresh connection code. Invalid,
expired, or rejected codes do not replace the saved browser connection. The
handoff accepts only HTTP loopback addresses and refuses redirects. The code
contains a short-lived pairing secret, not the raw extension token; do not share
it. Existing standard popup pairing remains available for direct-exec users.

These controls require the updated development extension and plugin, not an
older Chrome Web Store build. An already-running plugin/daemon must be refreshed
before testing changed backend behavior; installation alone does not replace
the running process.

Use **Knit Browser Feedback Preview**, the actual integration. The obsolete
**Knit Desktop Probe** was only a diagnostic and has been removed from the
development marketplace. Its source remains under `prototypes` for reference.
