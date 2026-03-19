package server

const floatingComposerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Knit Floating Composer</title>
  <style>
    :root {
      --bg: #f7f4ee;
      --panel: rgba(255,255,255,0.94);
      --text: #192434;
      --muted: #6a7383;
      --accent: #1c7c74;
      --accent-strong: #155b56;
      --accent-soft: #e4f4f1;
      --warn: #b97c1f;
      --danger: #c34f4f;
      --danger-soft: #fbe7e5;
      --good: #1f8f63;
      --line: #e7dfd4;
      --line-strong: #d8cdbd;
      --shadow: 0 18px 44px rgba(27, 39, 59, 0.10);
    }
    html[data-theme="dark"] {
      --bg: #0f1722;
      --panel: rgba(18, 28, 42, 0.9);
      --text: #eef4fb;
      --muted: #aab7ca;
      --accent: #62c6bc;
      --accent-strong: #8de0d7;
      --accent-soft: rgba(98, 198, 188, 0.16);
      --warn: #f0b35d;
      --danger: #ff8f8f;
      --danger-soft: rgba(195,79,79,.14);
      --good: #72d8a9;
      --line: rgba(157, 178, 205, 0.18);
      --line-strong: rgba(157, 178, 205, 0.28);
      --shadow: 0 22px 50px rgba(3, 8, 15, 0.42);
    }
    *, *::before, *::after { box-sizing: border-box; }
    html, body {
      margin: 0;
      padding: 0;
      background:
        radial-gradient(900px 400px at 0% 0%, rgba(28, 124, 116, 0.12), transparent 55%),
        linear-gradient(180deg, var(--panel) 0%, var(--bg) 100%);
      color: var(--text);
      font-family: "Inter", "SF Pro Text", "Segoe UI", sans-serif;
    }
    .wrap { padding: .45rem; }
    .theme-toggle {
      width: 2.65rem;
      height: 2.65rem;
      padding: 0;
      border-radius: 999px;
      display:inline-flex;
      align-items:center;
      justify-content:center;
      font-size: 1.05rem;
      flex: 0 0 auto;
    }
    .header-icon-plain {
      width: 2.4rem;
      height: 2.4rem;
      padding: 0;
      border: 0;
      background: transparent;
      border-radius: 999px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      box-shadow: none;
      color: var(--muted);
      font-size: 1.05rem;
      flex: 0 0 auto;
    }
    .header-icon-plain:hover {
      border-color: transparent;
      background: var(--accent-soft);
      color: var(--text);
      transform: none;
      box-shadow: none;
    }
    .header-icon-plain.danger {
      background: transparent;
      border: 0;
      color: var(--muted);
    }
    .header-icon-plain.danger:hover {
      background: var(--danger-soft);
      color: var(--danger);
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: .65rem;
      box-shadow: var(--shadow);
      backdrop-filter: blur(14px);
    }
    .row { display:flex; gap:.5rem; align-items:center; flex-wrap:wrap; }
    input, textarea, select { max-width: 100%; width: 100%; }
    textarea {
      width:100%;
      min-height:110px;
      background: var(--panel);
      color:var(--text);
      border:1px solid var(--line-strong);
      border-radius:16px;
      padding:.8rem .9rem;
      margin:.35rem 0;
      font: inherit;
    }
    button {
      background: var(--panel);
      color:var(--text);
      border:1px solid var(--line-strong);
      border-radius:14px;
      padding:.62rem .9rem;
      cursor:pointer;
      font-weight: 650;
      transition: transform .14s ease, border-color .14s ease, box-shadow .14s ease, background .14s ease;
    }
    button:hover {
      border-color: var(--accent);
      transform: translateY(-1px);
      box-shadow: 0 10px 22px rgba(27, 39, 59, 0.08);
    }
    button:focus-visible,
    input:focus-visible,
    textarea:focus-visible,
    select:focus-visible {
      outline: 3px solid rgba(28, 124, 116, 0.22);
      outline-offset: 2px;
    }
    button:disabled { opacity: .55; cursor: not-allowed; border-color: var(--line-strong); box-shadow:none; transform:none; }
    .ok { background: var(--accent); color: #fff; border-color: var(--accent); }
    .ok:hover { background: var(--accent-strong); border-color: var(--accent-strong); }
    .danger { color: var(--danger); border-color: rgba(195,79,79,.35); background: var(--danger-soft); }
    .small { font-size:.86rem; color: var(--muted); }
    .recording { color: var(--warn); }
    .meter {
      width: 100%;
      height: 12px;
      border-radius: 8px;
      border: 1px solid var(--line);
      background: var(--bg);
      overflow: hidden;
      margin-top: .3rem;
    }
    .meter-fill {
      width: 0%;
      height: 100%;
      background: linear-gradient(90deg, #4fd1c5 0%, #48bb78 50%, #f6ad55 80%, #f56565 100%);
      transition: width .08s linear;
    }
    .hidden { display: none !important; }
    .video-preview {
      width: 100%;
      max-height: 220px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: var(--bg);
      object-fit: contain;
      margin-top: .4rem;
    }
    .modal-overlay {
      position: fixed;
      inset: 0;
      background: rgba(19, 27, 39, 0.32);
      display: none;
      align-items: center;
      justify-content: center;
      z-index: 9999;
      padding: .6rem;
    }
    .modal-overlay.open { display: flex; }
    .modal-card {
      width: min(640px, 100%);
      max-height: 90vh;
      overflow: auto;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 1rem;
      box-shadow: 0 24px 60px rgba(27,39,59,.14);
    }
    .hero-line {
      display:flex;
      justify-content:space-between;
      gap:.8rem;
      align-items:flex-start;
    }
    .eyebrow {
      display:inline-flex;
      align-items:center;
      gap:.4rem;
      border-radius:999px;
      background: var(--accent-soft);
      color: var(--accent-strong);
      font-size:.76rem;
      font-weight:700;
      letter-spacing:.08em;
      text-transform:uppercase;
      padding:.35rem .65rem;
      margin-bottom:.75rem;
    }
    .title {
      font-size: 1.28rem;
      font-weight: 760;
      letter-spacing: -.02em;
      margin-bottom:.2rem;
    }
    .toolbar-grid {
      display:grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap:.55rem;
      margin-top:.9rem;
    }
    .toolbar-grid button {
      width:100%;
      justify-content:center;
    }
    .status-stack {
      display:grid;
      gap:.55rem;
      margin-top:.85rem;
    }
    .section-card {
      margin-top: 1rem;
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 1rem;
      background: rgba(255,255,255,0.46);
    }
    html[data-theme="dark"] .section-card {
      background: rgba(13, 21, 33, 0.42);
    }
    .section-title {
      font-size: 1rem;
      font-weight: 720;
      margin-bottom: .2rem;
    }
    .section-copy {
      color: var(--muted);
      font-size: .9rem;
      line-height: 1.5;
      margin-bottom: .8rem;
    }
    .field-grid {
      display: grid;
      gap: .8rem;
      grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
      align-items: start;
    }
    .field {
      display: grid;
      gap: .4rem;
      align-content: start;
    }
    .field-label {
      font-size: .82rem;
      font-weight: 700;
      color: var(--muted);
      letter-spacing: .01em;
    }
    .runtime-section {
      margin-top: .95rem;
      padding-top: .95rem;
      border-top: 1px solid var(--line);
    }
    .runtime-section:first-of-type {
      margin-top: .8rem;
      padding-top: 0;
      border-top: 0;
    }
    .runtime-section h4 {
      margin: 0 0 .55rem 0;
      font-size: .95rem;
    }
    .runtime-inline-note {
      color: var(--muted);
      font-size: .84rem;
      line-height: 1.45;
    }
    .sr-only {
      position: absolute;
      width: 1px;
      height: 1px;
      padding: 0;
      margin: -1px;
      overflow: hidden;
      clip: rect(0, 0, 0, 0);
      white-space: nowrap;
      border: 0;
    }
    .compact-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: .5rem;
    }
    .compact-brand {
      min-width: 0;
      display: grid;
      gap: .08rem;
    }
    .compact-brand strong {
      font-size: .82rem;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent-strong);
    }
    .compact-brand span {
      font-size: .76rem;
      color: var(--muted);
      line-height: 1.35;
    }
    .compact-controls {
      display: flex;
      align-items: center;
      gap: .35rem;
      flex: 0 0 auto;
    }
    .compact-capture-grid,
    .compact-utility-grid,
    .compact-send-grid {
      display: grid;
      gap: .42rem;
      margin-top: .55rem;
    }
    .compact-capture-grid {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
    .compact-utility-grid {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
    .compact-send-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
    .icon-tool {
      min-width: 0;
      min-height: 44px;
      padding: 0;
      border-radius: 14px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-size: 1.1rem;
      line-height: 1;
    }
    .icon-tool.active {
      border-color: var(--accent-strong);
      box-shadow: inset 0 0 0 1px rgba(31,143,99,.18);
      background: rgba(31,143,99,.1);
    }
    .icon-tool.wide {
      font-size: 1rem;
    }
    .compact-textarea {
      min-height: 78px;
      margin-top: .55rem;
    }
    .compact-preview,
    .compact-secondary {
      margin-top: .55rem;
      padding: .65rem;
    }
    .compact-preview summary,
    .compact-secondary summary {
      cursor: pointer;
      font-weight: 700;
      color: var(--text);
      list-style: none;
    }
    .compact-preview summary::-webkit-details-marker,
    .compact-secondary summary::-webkit-details-marker {
      display: none;
    }
    .compact-preview summary::before,
    .compact-secondary summary::before {
      content: "▸ ";
      color: var(--muted);
    }
    .compact-preview[open] summary::before,
    .compact-secondary[open] summary::before {
      content: "▾ ";
    }
    .compact-preview-copy {
      font-size: .78rem;
      color: var(--muted);
      margin-bottom: .45rem;
      line-height: 1.35;
    }
    .settings-mini-grid {
      display: grid;
      gap: .55rem;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      margin-top: .8rem;
    }
    .settings-summary {
      display: grid;
      gap: .4rem;
      margin-top: .75rem;
    }
    .settings-summary .status-pill {
      min-height: 0;
      padding: .55rem .7rem;
    }
    .capture-grid {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: .65rem;
    }
    .action-card {
      text-align: left;
      min-height: 84px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      gap: .35rem;
      padding: .8rem .9rem;
    }
    .action-card strong {
      font-size: .94rem;
    }
    .action-card span {
      color: var(--muted);
      font-size: .83rem;
      line-height: 1.4;
      font-weight: 500;
    }
    .mini-toolbar {
      display:flex;
      gap:.5rem;
      flex-wrap: wrap;
      margin-top: .85rem;
    }
    .mini-toolbar button {
      flex: 0 0 auto;
    }
    .note-actions {
      display:flex;
      gap:.55rem;
      flex-wrap:wrap;
      margin-top:.8rem;
      align-items:center;
    }
    .note-actions .grow {
      flex: 1 1 220px;
    }
    .preview-card {
      margin-top: .75rem;
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: .8rem;
      background: rgba(255,255,255,0.34);
    }
    html[data-theme="dark"] .preview-card {
      background: rgba(10, 17, 28, 0.36);
    }
    .request-preview {
      display: grid;
      gap: .75rem;
      margin-top: .2rem;
    }
    .preview-summary-card,
    .preview-note-card,
    .preview-warning-card {
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: .85rem;
      background: rgba(255,255,255,0.78);
    }
    html[data-theme="dark"] .preview-summary-card,
    html[data-theme="dark"] .preview-note-card,
    html[data-theme="dark"] .preview-warning-card {
      background: rgba(10, 17, 28, 0.62);
    }
    .preview-warning-card {
      background: var(--danger-soft);
    }
    .preview-kicker {
      font-size: .76rem;
      font-weight: 700;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent-strong);
      margin-bottom: .3rem;
    }
    .preview-summary-line,
    .preview-note-header {
      display: flex;
      justify-content: space-between;
      gap: .65rem;
      flex-wrap: wrap;
      align-items: center;
    }
    .preview-note-meta {
      display: flex;
      flex-wrap: wrap;
      gap: .45rem;
      margin-top: .4rem;
      color: var(--muted);
      font-size: .82rem;
    }
    .preview-note-meta span {
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: .18rem .48rem;
      background: rgba(255,255,255,0.58);
    }
    html[data-theme="dark"] .preview-note-meta span {
      background: rgba(11, 19, 30, 0.45);
    }
    .preview-note-text {
      margin-top: .55rem;
      white-space: pre-wrap;
      line-height: 1.5;
    }
    .preview-media-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: .7rem;
      margin-top: .75rem;
    }
    .preview-media {
      margin: 0;
      display: grid;
      gap: .4rem;
    }
    .preview-media figcaption {
      font-size: .82rem;
      font-weight: 650;
      color: var(--muted);
    }
    .preview-media img,
    .preview-media video,
    .preview-media audio {
      width: 100%;
      border-radius: 14px;
      border: 1px solid var(--line);
      background: var(--bg);
    }
    .preview-media audio {
      min-height: 44px;
    }
    .status-pill {
      border:1px solid var(--line);
      background: var(--panel);
      border-radius:16px;
      padding:.72rem .8rem;
      min-height: 52px;
    }
    .compact-footer {
      margin-top: .7rem;
      border-top: 1px solid var(--line);
      padding-top: .65rem;
    }
    .compact-radiator {
      display: grid;
      gap: .45rem;
    }
    .compact-radiator-grid {
      display: flex;
      flex-wrap: wrap;
      gap: .35rem;
    }
    .status-chip {
      display: inline-flex;
      align-items: center;
      gap: .32rem;
      min-height: 0;
      min-width: 0;
      max-width: 100%;
      padding: .38rem .6rem;
      border-radius: 999px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.72);
      color: var(--muted);
      font-size: .73rem;
      line-height: 1.25;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    html[data-theme="dark"] .status-chip {
      background: rgba(10, 17, 28, 0.62);
    }
    .status-chip.ok {
      color: var(--accent-strong);
      border-color: rgba(28,124,116,.22);
      background: var(--accent-soft);
    }
    .status-chip.warn {
      color: var(--warn);
      border-color: rgba(185,124,31,.24);
    }
    .status-chip.error {
      color: var(--danger);
      border-color: rgba(195,79,79,.24);
      background: var(--danger-soft);
    }
    .status-chip-label {
      font-weight: 700;
      color: var(--text);
      opacity: .84;
    }
    .status-chip-text {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    pre {
      background: var(--bg);
      color: var(--text);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: .95rem;
      max-width: 100%;
      min-width: 0;
      overflow: auto;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
      font-size: .88rem;
      line-height: 1.5;
    }
    .helper {
      margin-top:.75rem;
      color: var(--muted);
      font-size:.9rem;
    }
    .toast {
      position: fixed;
      left: 50%;
      bottom: 1rem;
      transform: translateX(-50%) translateY(12px);
      background: rgba(25, 36, 52, 0.92);
      color: #fff;
      border-radius: 999px;
      padding: .72rem 1rem;
      font-size: .88rem;
      font-weight: 650;
      box-shadow: 0 14px 30px rgba(15, 23, 34, 0.22);
      opacity: 0;
      pointer-events: none;
      transition: opacity .18s ease, transform .18s ease;
      z-index: 10001;
    }
    .toast.visible {
      opacity: 1;
      transform: translateX(-50%) translateY(0);
    }
    .toast.error {
      background: rgba(195, 79, 79, 0.94);
    }
    @media (max-width: 560px) {
      .toolbar-grid { grid-template-columns: 1fr; }
      .capture-grid { grid-template-columns: 1fr; }
      .compact-capture-grid,
      .compact-utility-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
      .settings-mini-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="panel">
      <div class="sr-only">Compact composer</div>
      <div class="compact-head">
        <div class="compact-brand">
          <strong>Compact Composer</strong>
          <span>Capture, preview, and send from a very small window.</span>
        </div>
        <div class="compact-controls">
          <button id="composerSettingsBtn" class="header-icon-plain" onclick="openComposerSettingsModalFC()" title="Open composer settings" aria-label="Open composer settings">⚙️</button>
          <button id="fcThemeToggleBtn" class="theme-toggle" onclick="toggleThemeFC()" title="Switch to dark theme" aria-label="Switch to dark theme">☾</button>
          <button class="danger header-icon-plain" onclick="window.close()" title="Close window" aria-label="Close window">✕</button>
        </div>
      </div>
      <div class="compact-capture-grid">
        <button id="talkOnlyBtn" class="ok icon-tool" onclick="submitAudioNote()" title="Talk only. Record a voice note without a screenshot." aria-label="Talk only. Record a voice note without a screenshot.">🎙️</button>
        <button id="snapshotTalkBtn" class="icon-tool" onclick="submitAudioNoteWithSnapshot()" title="Snapshot plus voice. Capture a snapshot and record a voice note." aria-label="Snapshot plus voice. Capture a snapshot and record a voice note.">📸</button>
        <button id="videoTalkBtn" class="icon-tool" onclick="submitVideoNote()" title="Video plus voice. Record a video clip with your voice." aria-label="Video plus voice. Record a video clip with your voice.">🎬</button>
        <button id="toggleTextEditorBtn" class="icon-tool" title="Type note. Show or hide the typed note field." aria-label="Type note. Show or hide the typed note field." onclick="toggleTextEditorFC()">✏️</button>
      </div>
      <div class="compact-utility-grid">
        <button id="copyCompanionBtn" class="icon-tool" title="Connect browser. Copy the Browser Companion snippet." aria-label="Connect browser. Copy the Browser Companion snippet." onclick="copyCompanionSnippetFC()">🔗</button>
        <button id="submitTextSnapshotBtn" class="icon-tool" title="Snapshot plus typed note. Open the typed note field if needed, then capture a snapshot with the current note." aria-label="Snapshot plus typed note. Open the typed note field if needed, then capture a snapshot with the current note." onclick="submitTextNoteWithSnapshot()">📝</button>
      </div>
      <textarea id="transcript" class="compact-textarea" placeholder="Type feedback note..." hidden></textarea>
      <div class="compact-send-grid">
        <button id="previewPayloadBtn" class="wide" onclick="previewPayloadFC()" title="Preview request">Preview</button>
        <button id="sendComposerBtn" class="ok wide" onclick="submitSessionFC()" title="Submit request">Submit</button>
      </div>
      <details id="fcPreviewDetails" class="preview-card compact-preview">
        <summary>Preview request</summary>
        <div class="compact-preview-copy" id="previewStateLine" style="margin-top:.45rem;">No preview yet. Capture at least one note, then preview before sending.</div>
        <div class="row" style="margin-top:0;margin-bottom:.45rem;justify-content:flex-end;">
          <button id="openLastLogBtn" class="wide" onclick="openLastLogFC()" title="Open last log">Open log</button>
        </div>
        <div id="fcPayloadPreview" class="request-preview small">Preview the request here before sending it to the agent.</div>
      </details>
      <details class="preview-card compact-secondary">
        <summary>Live agent output</summary>
        <div class="flow-stack" style="margin-top:.55rem;">
          <div>
            <div class="small" style="margin-bottom:.3rem;"><strong>Work log</strong></div>
            <pre id="fcLiveSubmitLog" style="max-height:140px;overflow:auto;">No live work log yet. Work activity appears here after the adapter starts writing logs.</pre>
          </div>
          <div>
            <div class="small" style="margin-bottom:.3rem;"><strong>Agent commentary</strong></div>
            <pre id="fcLiveSubmitCommentary" style="max-height:100px;overflow:auto;">No agent commentary yet. Plain-language progress updates appear here when the agent explains what it is doing.</pre>
          </div>
          <div class="small">Raw prompt/payload details stay available in the full execution log via <strong>Open log</strong>.</div>
        </div>
      </details>
      <details class="preview-card compact-secondary" open>
        <summary>Recent runs</summary>
        <div id="fcSubmitHistory" class="small">No runs yet.</div>
      </details>
      <div class="compact-footer">
        <div id="fcStatusRadiator" class="compact-radiator" aria-label="Composer status">
          <div id="fcSensitiveCaptureBadges" class="compact-radiator-grid"></div>
          <div class="compact-radiator-grid">
            <div class="status-chip" id="stateLine" title="Checking session state..."><span class="status-chip-label">Session</span><span class="status-chip-text">Checking session state...</span></div>
            <div class="status-chip" id="queueLine" title="Queue: loading..."><span class="status-chip-label">Queue</span><span class="status-chip-text">Queue: loading...</span></div>
            <div class="status-chip" id="runtimeGuideLine" title="Platform runtime: loading..."><span class="status-chip-label">Runtime</span><span class="status-chip-text">Platform runtime: loading...</span></div>
            <div class="status-chip" id="recordLine" title="Recorder idle."><span class="status-chip-label">Recorder</span><span class="status-chip-text">Recorder idle.</span></div>
            <div class="status-chip" id="screenshotLine" title="No snapshot queued. Snapshot actions can capture one automatically."><span class="status-chip-label">Snapshot</span><span class="status-chip-text">No snapshot queued.</span></div>
            <div class="status-chip ok" id="statusLine" title="Ready."><span class="status-chip-label">Status</span><span class="status-chip-text">Ready.</span></div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div id="composerSettingsModal" class="modal-overlay" onclick="onComposerSettingsModalBackdropFC(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <strong>Composer Settings</strong>
        <button class="danger" onclick="closeComposerSettingsModalFC()" title="Close">Close</button>
      </div>
      <div class="small" style="margin:.2rem 0 .4rem 0;">Workspace and agent configuration stay behind this gear menu so the compact composer remains usable in a very small window.</div>
      <div class="settings-mini-grid">
        <button id="openWorkspaceBtn" onclick="openWorkspaceFromSettingsFC()" title="Open workspace settings">📁 Workspace</button>
        <button id="openVideoCaptureBtn" onclick="openVideoCaptureModalFC()" title="Open video tools">🎥 Video tools</button>
        <button id="openAudioControlsBtn" onclick="openAudioControlsModal()" title="Open audio controls">🎚️ Audio controls</button>
        <button id="openCodexRuntimeBtn" onclick="openCodexRuntimeFromSettingsFC()" title="Open agent runtime settings">🤖 Agent Runtime</button>
        <button id="openDocsLibraryBtnFC" onclick="openDocsBrowserFC()" title="Open docs library in a new tab">📚 Docs</button>
      </div>
      <div class="settings-summary">
        <div class="status-pill small">Workspace: <code id="fcSettingsWorkspaceLabel">(not set)</code></div>
        <div class="status-pill small">Default adapter: <span id="fcSettingsProviderLabel">codex_cli</span></div>
      </div>
      <label class="small" style="display:flex;align-items:center;gap:.4rem;" title="Include typed form values in the replay bundle for this session. Enabled by default for new sessions.">
        <input id="fcCaptureInputValuesToggle" type="checkbox" onchange="toggleReplayValueCaptureFC()" />
        Capture typed values for replay
      </label>
      <label class="small" style="display:flex;align-items:center;gap:.4rem;" title="Allow larger screenshot, audio, or video payloads to be sent inline when needed.">
        <input id="fcAllowLargeInlineMediaToggle" type="checkbox" />
        Allow large inline media when needed
      </label>
      <div class="small" style="margin-top:.35rem;">Typed values are enabled by default for new sessions. Turn this off here when you want replay bundles to redact non-secret form values.</div>
    </div>
  </div>
  <div id="workspaceModal" class="modal-overlay" onclick="onWorkspaceModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <strong>Workspace (Required)</strong>
        <button id="fcWorkspaceCloseBtn" class="danger" onclick="closeWorkspaceModal()" title="Close">Close</button>
      </div>
      <div class="small" style="margin:.2rem 0 .4rem 0;">Select the repository/workspace directory used for coding-agent submissions.</div>
      <div class="row">
        <input id="fcWorkspaceDir" placeholder="/abs/path/repo" style="min-width:350px;" />
        <button onclick="applyWorkspaceFC()" title="Apply workspace">Apply Workspace</button>
        <button class="ok" onclick="pickWorkspaceDirFC()" title="Choose folder">Choose Folder...</button>
      </div>
      <div id="fcWorkspaceStatus" class="small">Workspace selection required.</div>
      <pre id="fcWorkspaceState">workspace not loaded</pre>
    </div>
  </div>

  <div id="audioControlsModal" class="modal-overlay" onclick="onAudioControlsModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <strong>Audio Controls</strong>
        <div class="row">
          <button id="openTranscriptionFromAudioBtnFC" title="Transcription Runtime" aria-label="Transcription Runtime" onclick="openTranscriptionRuntimeFromAudioModalFC()">⚙️</button>
          <button class="danger" onclick="closeAudioControlsModal()" title="Close">Close</button>
        </div>
      </div>
      <div class="row" style="margin-top:.3rem;">
        <label for="fcAudioMode">Mode:</label>
        <select id="fcAudioMode">
          <option value="always_on">Always on</option>
          <option value="push_to_talk">Push to talk</option>
        </select>
        <label for="fcAudioInputDevice">Input:</label>
        <select id="fcAudioInputDevice" style="min-width:220px;">
          <option value="default">default</option>
        </select>
        <button onclick="refreshAudioDevicesFC()" title="Refresh devices">Refresh Devices</button>
      </div>
      <div class="row">
        <label><input id="fcAudioMuted" type="checkbox" /> muted</label>
        <label><input id="fcAudioPaused" type="checkbox" /> paused</label>
        <button id="fcTestMicBtn" onclick="testMicrophoneFC()" title="Test microphone for 10 seconds">Test Mic (10s)</button>
      </div>
      <div id="fcMicTestState" class="small">Mic test idle.</div>
      <div class="meter"><div id="fcMicTestMeterFill" class="meter-fill"></div></div>
      <div id="fcAudioLevelState" class="small hidden" style="margin-top:.3rem;">Audio level: unknown</div>
    </div>
  </div>
  <div id="transcriptionRuntimeModal" class="modal-overlay" onclick="onTranscriptionRuntimeModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <strong>Transcription Runtime</strong>
        <button class="danger" onclick="closeTranscriptionRuntimeModal()" title="Close">Close</button>
      </div>
      <div class="row" style="margin-top:.3rem;">
        <label for="fcSttMode">Mode:</label>
        <select id="fcSttMode">
          <option value="remote">OpenAI</option>
          <option value="lmstudio">LM Studio</option>
          <option value="faster_whisper">Faster Whisper (local)</option>
          <option value="local">Custom local command</option>
        </select>
      </div>
      <div id="fcSttModeHelp" class="small">Remote OpenAI transcription uses a base URL and model.</div>
      <div id="fcSttConnectionRow" class="row">
        <span id="fcSttBaseURLWrap"><input id="fcSttBaseURL" placeholder="OpenAI base URL" style="min-width:260px;" /></span>
        <span id="fcSttModelWrap"><input id="fcSttModel" placeholder="OpenAI STT model" style="min-width:200px;" /></span>
      </div>
      <div id="fcSttFasterWhisperRow" class="row hidden">
        <span id="fcSttFasterWhisperModelWrap">
          <select id="fcSttFasterWhisperModel" style="min-width:200px;">
            <option value="tiny.en">tiny.en</option>
            <option value="tiny">tiny</option>
            <option value="base.en">base.en</option>
            <option value="base">base</option>
            <option value="small.en">small.en</option>
            <option value="small">small</option>
            <option value="medium.en">medium.en</option>
            <option value="medium">medium</option>
            <option value="large-v1">large-v1</option>
            <option value="large-v2">large-v2</option>
            <option value="large-v3">large-v3</option>
            <option value="large">large</option>
            <option value="distil-large-v2">distil-large-v2</option>
            <option value="distil-medium.en">distil-medium.en</option>
            <option value="distil-small.en">distil-small.en</option>
            <option value="distil-large-v3">distil-large-v3</option>
            <option value="distil-large-v3.5">distil-large-v3.5</option>
            <option value="large-v3-turbo">large-v3-turbo</option>
            <option value="turbo">turbo</option>
          </select>
        </span>
        <span id="fcSttDeviceWrap">
          <select id="fcSttDevice" style="min-width:200px;">
            <option value="cpu">cpu</option>
            <option value="cuda">cuda</option>
            <option value="metal">metal</option>
          </select>
        </span>
        <span id="fcSttComputeTypeWrap">
          <select id="fcSttComputeType" style="min-width:200px;">
            <option value="int8">int8</option>
            <option value="float16">float16</option>
            <option value="int8_float16">int8_float16</option>
            <option value="float32">float32</option>
          </select>
        </span>
        <span id="fcSttLanguageWrap"><input id="fcSttLanguage" placeholder="language (optional, e.g. en)" style="min-width:200px;" maxlength="24" pattern="[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}" title="Use a short language tag such as en or en-US" /></span>
      </div>
      <div id="fcSttCommandRow" class="row hidden">
        <span id="fcSttLocalCommandWrap"><input id="fcSttLocalCommand" placeholder="local command (KNIT_LOCAL_STT_CMD)" style="min-width:390px;" maxlength="2048" spellcheck="false" title="Single-line command only" /></span>
        <span id="fcSttTimeoutWrap"><input id="fcSttTimeoutSeconds" type="number" min="1" max="600" step="1" placeholder="timeout seconds (1-600)" style="min-width:190px;" /></span>
      </div>
      <div class="row">
        <button onclick="checkTranscriptionHealthFC()" title="Check transcription connection">Check connection</button>
      </div>
      <div id="fcSttHealthState" class="small">Transcription health: unknown</div>
      <pre id="fcSttRuntimeState">transcription runtime settings will appear here</pre>
    </div>
  </div>
  <div id="codexRuntimeModal" class="modal-overlay" onclick="onCodexRuntimeModalBackdropFC(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <strong>Agent Runtime (Codex/Claude/OpenCode)</strong>
        <button class="danger" onclick="closeCodexRuntimeModalFC()" title="Close">Close</button>
      </div>
      <div class="small" style="margin:.2rem 0 .4rem 0;">Select the default submit adapter and Knit will save runtime changes automatically after you stop typing.</div>
      <div class="field-grid">
        <label class="field">
          <span class="field-label">Default submit adapter</span>
          <select id="fcAgentDefaultProvider" style="min-width:220px;">
            <option value="codex_cli">codex_cli</option>
            <option value="claude_cli">claude_cli</option>
            <option value="codex_api">codex_api</option>
            <option value="claude_api">claude_api</option>
            <option value="opencode_cli">opencode_cli</option>
          </select>
        </label>
      </div>
      <div id="fcRuntimeProviderHelp" class="runtime-inline-note">Knit will show only the fields used by the selected adapter.</div>

      <section id="fcCodexCliSection" class="runtime-section">
        <h4>Codex CLI</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">CLI command</span>
            <input id="fcCodexCliCmd" placeholder="codex_cli command (KNIT_CLI_ADAPTER_CMD)" style="min-width:340px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Timeout seconds</span>
            <input id="fcCliTimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 600)" style="min-width:180px;" />
          </label>
        </div>
      </section>

      <section id="fcClaudeCliSection" class="runtime-section hidden">
        <h4>Claude CLI</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">CLI command</span>
            <input id="fcClaudeCliCmd" placeholder="claude_cli command (KNIT_CLAUDE_CLI_ADAPTER_CMD)" style="min-width:340px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Timeout seconds</span>
            <input id="fcClaudeCliTimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 600)" style="min-width:180px;" />
          </label>
        </div>
      </section>

      <section id="fcOpenCodeCliSection" class="runtime-section hidden">
        <h4>OpenCode CLI</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">CLI command</span>
            <input id="fcOpenCodeCliCmd" placeholder="opencode_cli command (KNIT_OPENCODE_CLI_ADAPTER_CMD)" style="min-width:340px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Timeout seconds</span>
            <input id="fcOpenCodeCliTimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 600)" style="min-width:180px;" />
          </label>
        </div>
      </section>

      <section id="fcCodexAPISection" class="runtime-section hidden">
        <h4>Codex API</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Base URL</span>
            <input id="fcCodexAPIBaseURL" type="url" placeholder="https://api.openai.com" style="min-width:340px;" maxlength="1024" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">API timeout seconds</span>
            <input id="fcCodexAPITimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 60)" style="min-width:180px;" />
          </label>
          <label class="field">
            <span class="field-label">OpenAI org ID</span>
            <input id="fcCodexAPIOrg" placeholder="OPENAI_ORG_ID (optional)" style="min-width:260px;" maxlength="256" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">OpenAI project ID</span>
            <input id="fcCodexAPIProject" placeholder="OPENAI_PROJECT_ID (optional)" style="min-width:260px;" maxlength="256" spellcheck="false" />
          </label>
        </div>
      </section>

      <section id="fcClaudeAPISection" class="runtime-section hidden">
        <h4>Claude API</h4>
        <div id="fcClaudeAPIKeyStatus" class="runtime-inline-note">Set ANTHROPIC_API_KEY in the environment before using claude_api.</div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Base URL</span>
            <input id="fcClaudeAPIBaseURL" type="url" placeholder="https://api.anthropic.com" style="min-width:340px;" maxlength="1024" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">API timeout seconds</span>
            <input id="fcClaudeAPITimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 60)" style="min-width:180px;" />
          </label>
          <label class="field">
            <span class="field-label">Model</span>
            <input id="fcClaudeAPIModel" placeholder="KNIT_CLAUDE_API_MODEL" style="min-width:260px;" maxlength="128" spellcheck="false" />
          </label>
        </div>
      </section>

      <section id="fcCodexSharedSection" class="runtime-section">
        <h4>Shared Submission Settings</h4>
        <div class="runtime-inline-note">Workspace is managed from the Workspace modal and is reused across adapters.</div>
        <div class="row" style="margin-top:.5rem;">
          <div><strong>Workspace:</strong> <code id="fcCodexWorkdirLabel">(not set)</code></div>
          <button onclick="openWorkspaceModal()" title="Set workspace">Set Workspace</button>
        </div>
        <div class="field-grid" style="margin-top:.6rem;">
          <label class="field">
            <span class="field-label">Output directory</span>
            <input id="fcCodexOutputDir" placeholder="/tmp" style="min-width:170px;" maxlength="1024" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">Submit mode</span>
            <select id="fcSubmitExecutionMode">
              <option value="series">series (default)</option>
              <option value="parallel">parallel</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Post-submit rebuild command</span>
            <input id="fcPostSubmitRebuildCmd" placeholder="post-submit rebuild command (optional)" style="min-width:380px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Post-submit verify/test command</span>
            <input id="fcPostSubmitVerifyCmd" placeholder="post-submit verify/test command (optional)" style="min-width:380px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Post-submit timeout seconds</span>
            <input id="fcPostSubmitTimeoutSec" type="number" min="1" max="7200" step="1" inputmode="numeric" placeholder="1-7200 (default 600)" style="min-width:210px;" />
          </label>
        </div>
      </section>

      <section id="fcCodexCommonSection" class="runtime-section">
        <h4>Codex Model Settings</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Profile</span>
            <input id="fcCodexProfile" placeholder="profile (optional)" style="min-width:180px;" maxlength="128" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">Model</span>
            <select id="fcCodexModel" style="min-width:220px;">
              <option value="">Use Codex default model</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Reasoning effort</span>
            <select id="fcCodexReasoning" style="min-width:220px;">
              <option value="">Use Codex default reasoning</option>
            </select>
          </label>
        </div>
        <div class="runtime-inline-note">Profile maps to your local Codex config.toml profile. Use a separate profile for Knit if you want different MCP servers or auth behavior. Knit only loads Codex model options when you click Refresh.</div>
        <div class="row" style="margin-top:.6rem;">
          <button onclick="refreshCodexOptionsFC()" title="Refresh Codex options">Refresh Codex Options</button>
        </div>
      </section>

      <section id="fcCodexCLIDefaultsSection" class="runtime-section">
        <h4>Codex CLI Defaults</h4>
        <div class="runtime-inline-note" id="fcCodexDefaultBehavior">Knit defaults local coding-agent runs to <code>workspace-write</code> sandbox and <code>never</code> approval so implementation requests can complete without falling back to read-only behavior.</div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Sandbox</span>
            <select id="fcCodexSandbox">
              <option value="read-only">read-only</option>
              <option value="workspace-write">workspace-write</option>
              <option value="danger-full-access">danger-full-access</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Approval policy</span>
            <select id="fcCodexApproval">
              <option value="untrusted">untrusted</option>
              <option value="on-request">on-request</option>
              <option value="never">never</option>
            </select>
          </label>
          <div class="field">
            <span class="field-label">Repository safety</span>
            <label style="display:flex;align-items:center;gap:.55rem;padding-top:.75rem;">
              <input id="fcCodexSkipRepoCheck" type="checkbox" checked />
              <span>Skip Git repo check</span>
            </label>
          </div>
        </div>
      </section>

      <section id="fcDeliveryPromptSection" class="runtime-section">
        <h4>Delivery Prompt</h4>
        <div class="runtime-inline-note">Choose what the agent should do with the approved Knit feedback, then edit the resolved prompt directly if you want to override the default handoff.</div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Prompt template</span>
            <select id="fcDeliveryIntentProfile" onchange="syncDeliveryIntentPromptTextFC(true)">
              <option value="implement_changes">Implement changes</option>
              <option value="draft_plan">Draft plan</option>
              <option value="create_jira_tickets">Create Jira tickets</option>
            </select>
          </label>
          <label class="field" style="grid-column:1 / -1;">
            <span class="field-label">Prompt text</span>
            <textarea id="fcDeliveryInstructionText" class="compact-textarea" rows="9" placeholder="The selected prompt template will appear here."></textarea>
          </label>
        </div>
      </section>

      <div id="fcCodexOptionsState" class="small">Codex options not loaded yet.</div>
      <pre id="fcCodexRuntimeState">runtime codex settings will appear here</pre>
    </div>
  </div>
  <div id="videoCaptureModal" class="modal-overlay" onclick="onVideoCaptureModalBackdropFC(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <strong>Video Capture</strong>
        <button class="danger" onclick="closeVideoCaptureModalFC()" title="Close">Close</button>
      </div>
      <div class="small" style="margin:.2rem 0 .4rem 0;">
        Companion (🔗) is required. Use this to queue a manual snapshot for the next note or keep a live visual open while you review.
      </div>
      <div class="row">
        <button id="startLiveVideoBtn" onclick="startLiveVideoFC()" title="Start live video">Start Live Video</button>
        <button id="stopLiveVideoBtn" onclick="stopLiveVideoFC()" title="Stop live video">Stop Live Video</button>
        <button id="captureScreenshotBtn" onclick="captureScreenshotForNextNote()" title="Capture snapshot">Capture Snapshot</button>
        <button id="clearScreenshotBtn" onclick="clearQueuedScreenshot()" title="Clear snapshot">Clear Snapshot</button>
      </div>
      <div class="small" style="margin-top:.35rem;">Live preview is available here. For event clips/advanced recording modes, use the main Knit page video modal.</div>
      <div class="small" id="fcLiveVideoState" style="margin-top:.25rem;">Live video: off</div>
      <video id="fcLivePreview" class="video-preview hidden" autoplay muted playsinline></video>
      <div class="small" id="fcVideoCaptureState" style="margin-top:.25rem;">No snapshot queued.</div>
    </div>
  </div>
  <div id="fcToast" class="toast" role="status" aria-live="polite"></div>
<script>
const controlToken = '__KNIT_TOKEN__';
const stateLineEl = document.getElementById('stateLine');
const queueLineEl = document.getElementById('queueLine');
const fcSensitiveCaptureBadgesEl = document.getElementById('fcSensitiveCaptureBadges');
const statusLineEl = document.getElementById('statusLine');
const previewStateLineEl = document.getElementById('previewStateLine');
const fcPreviewDetailsEl = document.getElementById('fcPreviewDetails');
const runtimeGuideLineEl = document.getElementById('runtimeGuideLine');
const fcPayloadPreviewEl = document.getElementById('fcPayloadPreview');
const fcSubmitHistoryEl = document.getElementById('fcSubmitHistory');
const fcCaptureInputValuesToggleEl = document.getElementById('fcCaptureInputValuesToggle');
const fcAllowLargeInlineMediaToggleEl = document.getElementById('fcAllowLargeInlineMediaToggle');
const transcriptEl = document.getElementById('transcript');
const fcDeliveryIntentProfileEl = document.getElementById('fcDeliveryIntentProfile');
const fcDeliveryInstructionTextEl = document.getElementById('fcDeliveryInstructionText');
const talkOnlyBtnEl = document.getElementById('talkOnlyBtn');
const snapshotTalkBtnEl = document.getElementById('snapshotTalkBtn');
const videoTalkBtnEl = document.getElementById('videoTalkBtn');
const toggleTextEditorBtnEl = document.getElementById('toggleTextEditorBtn');
const composerSettingsBtnEl = document.getElementById('composerSettingsBtn');
const openWorkspaceBtnEl = document.getElementById('openWorkspaceBtn');
const copyCompanionBtnEl = document.getElementById('copyCompanionBtn');
const openVideoCaptureBtnEl = document.getElementById('openVideoCaptureBtn');
const submitTextSnapshotBtnEl = document.getElementById('submitTextSnapshotBtn');
const openAudioControlsBtnEl = document.getElementById('openAudioControlsBtn');
const openCodexRuntimeBtnEl = document.getElementById('openCodexRuntimeBtn');
const previewPayloadBtnEl = document.getElementById('previewPayloadBtn');
const sendComposerBtnEl = document.getElementById('sendComposerBtn');
const openLastLogBtnEl = document.getElementById('openLastLogBtn');
const captureScreenshotBtnEl = document.getElementById('captureScreenshotBtn');
const clearScreenshotBtnEl = document.getElementById('clearScreenshotBtn');
const recordLineEl = document.getElementById('recordLine');
const screenshotLineEl = document.getElementById('screenshotLine');
const composerSettingsModalEl = document.getElementById('composerSettingsModal');
const fcSettingsWorkspaceLabelEl = document.getElementById('fcSettingsWorkspaceLabel');
const fcSettingsProviderLabelEl = document.getElementById('fcSettingsProviderLabel');
const workspaceModalEl = document.getElementById('workspaceModal');
const fcWorkspaceCloseBtnEl = document.getElementById('fcWorkspaceCloseBtn');
const fcWorkspaceDirEl = document.getElementById('fcWorkspaceDir');
const fcWorkspaceStatusEl = document.getElementById('fcWorkspaceStatus');
const fcWorkspaceStateEl = document.getElementById('fcWorkspaceState');
const audioControlsModalEl = document.getElementById('audioControlsModal');
const transcriptionRuntimeModalEl = document.getElementById('transcriptionRuntimeModal');
const codexRuntimeModalEl = document.getElementById('codexRuntimeModal');
const videoCaptureModalEl = document.getElementById('videoCaptureModal');
const fcVideoCaptureStateEl = document.getElementById('fcVideoCaptureState');
const fcLiveVideoStateEl = document.getElementById('fcLiveVideoState');
const fcLivePreviewEl = document.getElementById('fcLivePreview');
const startLiveVideoBtnEl = document.getElementById('startLiveVideoBtn');
const stopLiveVideoBtnEl = document.getElementById('stopLiveVideoBtn');
const fcAudioModeEl = document.getElementById('fcAudioMode');
const fcAudioInputDeviceEl = document.getElementById('fcAudioInputDevice');
const fcAudioMutedEl = document.getElementById('fcAudioMuted');
const fcAudioPausedEl = document.getElementById('fcAudioPaused');
const fcTestMicBtnEl = document.getElementById('fcTestMicBtn');
const fcMicTestStateEl = document.getElementById('fcMicTestState');
const fcMicTestMeterFillEl = document.getElementById('fcMicTestMeterFill');
const fcAudioLevelStateEl = document.getElementById('fcAudioLevelState');
const fcSttModeEl = document.getElementById('fcSttMode');
const fcSttBaseURLEl = document.getElementById('fcSttBaseURL');
const fcSttModelEl = document.getElementById('fcSttModel');
const fcSttFasterWhisperModelEl = document.getElementById('fcSttFasterWhisperModel');
const fcSttDeviceEl = document.getElementById('fcSttDevice');
const fcSttComputeTypeEl = document.getElementById('fcSttComputeType');
const fcSttLanguageEl = document.getElementById('fcSttLanguage');
const fcSttLocalCommandEl = document.getElementById('fcSttLocalCommand');
const fcSttTimeoutSecondsEl = document.getElementById('fcSttTimeoutSeconds');
const fcSttModeHelpEl = document.getElementById('fcSttModeHelp');
const fcSttConnectionRowEl = document.getElementById('fcSttConnectionRow');
const fcSttFasterWhisperRowEl = document.getElementById('fcSttFasterWhisperRow');
const fcSttCommandRowEl = document.getElementById('fcSttCommandRow');
const fcSttBaseURLWrapEl = document.getElementById('fcSttBaseURLWrap');
const fcSttModelWrapEl = document.getElementById('fcSttModelWrap');
const fcSttFasterWhisperModelWrapEl = document.getElementById('fcSttFasterWhisperModelWrap');
const fcSttDeviceWrapEl = document.getElementById('fcSttDeviceWrap');
const fcSttComputeTypeWrapEl = document.getElementById('fcSttComputeTypeWrap');
const fcSttLanguageWrapEl = document.getElementById('fcSttLanguageWrap');
const fcSttLocalCommandWrapEl = document.getElementById('fcSttLocalCommandWrap');
const fcSttTimeoutWrapEl = document.getElementById('fcSttTimeoutWrap');
const fcSttHealthStateEl = document.getElementById('fcSttHealthState');
const fcSttRuntimeStateEl = document.getElementById('fcSttRuntimeState');
const fcAgentDefaultProviderEl = document.getElementById('fcAgentDefaultProvider');
const fcRuntimeProviderHelpEl = document.getElementById('fcRuntimeProviderHelp');
const fcCodexCliSectionEl = document.getElementById('fcCodexCliSection');
const fcClaudeCliSectionEl = document.getElementById('fcClaudeCliSection');
const fcOpenCodeCliSectionEl = document.getElementById('fcOpenCodeCliSection');
const fcCodexAPISectionEl = document.getElementById('fcCodexAPISection');
const fcClaudeAPISectionEl = document.getElementById('fcClaudeAPISection');
const fcCodexSharedSectionEl = document.getElementById('fcCodexSharedSection');
const fcCodexCommonSectionEl = document.getElementById('fcCodexCommonSection');
const fcCodexCLIDefaultsSectionEl = document.getElementById('fcCodexCLIDefaultsSection');
const fcCodexCliCmdEl = document.getElementById('fcCodexCliCmd');
const fcClaudeCliCmdEl = document.getElementById('fcClaudeCliCmd');
const fcOpenCodeCliCmdEl = document.getElementById('fcOpenCodeCliCmd');
const fcCodexWorkdirLabelEl = document.getElementById('fcCodexWorkdirLabel');
const fcCodexOutputDirEl = document.getElementById('fcCodexOutputDir');
const fcCliTimeoutSecondsEl = document.getElementById('fcCliTimeoutSeconds');
const fcClaudeCliTimeoutSecondsEl = document.getElementById('fcClaudeCliTimeoutSeconds');
const fcOpenCodeCliTimeoutSecondsEl = document.getElementById('fcOpenCodeCliTimeoutSeconds');
const fcSubmitExecutionModeEl = document.getElementById('fcSubmitExecutionMode');
const fcCodexSandboxEl = document.getElementById('fcCodexSandbox');
const fcCodexApprovalEl = document.getElementById('fcCodexApproval');
const fcCodexSkipRepoCheckEl = document.getElementById('fcCodexSkipRepoCheck');
const fcCodexProfileEl = document.getElementById('fcCodexProfile');
const fcCodexModelEl = document.getElementById('fcCodexModel');
const fcCodexReasoningEl = document.getElementById('fcCodexReasoning');
const fcCodexAPIBaseURLEl = document.getElementById('fcCodexAPIBaseURL');
const fcCodexAPITimeoutSecondsEl = document.getElementById('fcCodexAPITimeoutSeconds');
const fcCodexAPIOrgEl = document.getElementById('fcCodexAPIOrg');
const fcCodexAPIProjectEl = document.getElementById('fcCodexAPIProject');
const fcClaudeAPIBaseURLEl = document.getElementById('fcClaudeAPIBaseURL');
const fcClaudeAPITimeoutSecondsEl = document.getElementById('fcClaudeAPITimeoutSeconds');
const fcClaudeAPIModelEl = document.getElementById('fcClaudeAPIModel');
const fcClaudeAPIKeyStatusEl = document.getElementById('fcClaudeAPIKeyStatus');
const fcPostSubmitRebuildCmdEl = document.getElementById('fcPostSubmitRebuildCmd');
const fcPostSubmitVerifyCmdEl = document.getElementById('fcPostSubmitVerifyCmd');
const fcPostSubmitTimeoutSecEl = document.getElementById('fcPostSubmitTimeoutSec');
const fcCodexOptionsStateEl = document.getElementById('fcCodexOptionsState');
const fcCodexRuntimeStateEl = document.getElementById('fcCodexRuntimeState');
const fcThemeToggleBtnEl = document.getElementById('fcThemeToggleBtn');
const fcToastEl = document.getElementById('fcToast');
const fcLiveSubmitLogEl = document.getElementById('fcLiveSubmitLog');
const fcLiveSubmitCommentaryEl = document.getElementById('fcLiveSubmitCommentary');
const fcSubmitAttemptOutputPreviewByID = new Map();
const fcSubmitAttemptOutputPreviewInflight = new Set();
const fcSeenSubmitRecoveryNotices = new Set();

let queuedScreenshotBlob = null;
let inFlight = false;
let recording = false;
let lastState = null;
let fcLatestPayloadPreviewData = null;
let fcPreviewDeliveryOptions = { redactReplayValues: false, omitVideoClips: false, omitVideoEventIDs: [] };
let fcClipBlobCacheByEventID = new Map();
let fcClipResizeInFlight = new Set();
let fcAudioDirty = false;
let fcAudioApplying = false;
let fcAudioApplyTimer = 0;
let fcSTTRuntimeDirty = false;
let fcSTTRuntimeApplying = false;
let fcSTTRuntimeApplyTimer = 0;
let fcPTTHeld = false;
let fcMicTestRunning = false;
let fcWorkspacePrompted = false;
let fcWorkspaceRequired = false;
let fcCodexOptionsLoaded = false;
let fcCodexOptionsAttempted = false;
let fcCodexRuntimeDirty = false;
let fcCodexRuntimeApplying = false;
let fcCodexRuntimeApplyTimer = 0;
let fcRecordingKind = '';
let fcAudioNoteMode = '';
let fcActiveAudioNote = null;
let fcAudioNoteRecorder = null;
let fcAudioNoteStream = null;
let fcAudioNoteChunks = [];
let fcAudioNoteStopPromise = null;
let fcActiveVideoNote = null;
let fcVideoNoteAudioRecorder = null;
let fcVideoNoteClipRecorder = null;
let fcVideoNoteMicStream = null;
let fcVideoNoteDisplayStream = null;
let fcVideoNoteCombinedStream = null;
let fcVideoNoteAudioChunks = [];
let fcVideoNoteClipChunks = [];
let fcVideoNoteAudioStopPromise = null;
let fcVideoNoteClipStopPromise = null;
let fcVideoNoteFinalizing = false;
let fcTextEditorOpen = false;
let fcLiveDisplayStream = null;
const FC_SETTINGS_KEY = 'knit_ui_settings_v1';
let fcSettings = {};
let fcCurrentTheme = 'light';
let fcToastTimer = 0;
let fcLiveLogAttemptId = '';
let fcLiveLogOffset = 0;
let fcLiveLogRawText = '';
let fcLiveLogCompletedForAttempt = '';
let fcLiveLogUnavailableForAttempt = '';
let fcWatchedSubmitAttemptIDs = new Set();
let fcSubmitAttemptStatusByID = new Map();
let fcSubmitAttemptNotificationsReady = false;

function loadFCSettings() {
  try {
    const raw = window.localStorage ? window.localStorage.getItem(FC_SETTINGS_KEY) : '';
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
    return parsed;
  } catch (_) {
    return {};
  }
}

function saveFCSettings() {
  try {
    if (!window.localStorage) return;
    window.localStorage.setItem(FC_SETTINGS_KEY, JSON.stringify(fcSettings));
  } catch (_) {}
}

function hasFCSetting(key) {
  return Object.prototype.hasOwnProperty.call(fcSettings, key);
}

function setFCSetting(key, value) {
  if (!key) return;
  fcSettings[key] = value;
  saveFCSettings();
}

function normalizeThemeFC(theme) {
  return String(theme || '').trim().toLowerCase() === 'dark' ? 'dark' : 'light';
}

function applyThemeFC(theme) {
  fcCurrentTheme = normalizeThemeFC(theme);
  document.documentElement.setAttribute('data-theme', fcCurrentTheme);
  if (!fcThemeToggleBtnEl) return;
  const nextTheme = fcCurrentTheme === 'dark' ? 'light' : 'dark';
  fcThemeToggleBtnEl.textContent = fcCurrentTheme === 'dark' ? '☀' : '☾';
  fcThemeToggleBtnEl.title = nextTheme === 'dark' ? 'Switch to dark theme' : 'Switch to light theme';
  fcThemeToggleBtnEl.setAttribute('aria-label', fcThemeToggleBtnEl.title);
}

function toggleThemeFC() {
  const nextTheme = fcCurrentTheme === 'dark' ? 'light' : 'dark';
  applyThemeFC(nextTheme);
  setFCSetting('theme', nextTheme);
}

function bindFCPersistedField(el, key, opts = {}) {
  if (!el || !key) return;
  const isCheckbox = !!opts.checkbox;
  if (hasFCSetting(key)) {
    if (isCheckbox) {
      el.checked = !!fcSettings[key];
    } else {
      el.value = String(fcSettings[key] ?? '');
    }
  }
  const save = () => {
    if (isCheckbox) {
      setFCSetting(key, !!el.checked);
      if (typeof opts.afterSave === 'function') opts.afterSave();
      return;
    }
    setFCSetting(key, String(el.value ?? ''));
    if (typeof opts.afterSave === 'function') opts.afterSave();
  };
  el.addEventListener(isCheckbox ? 'change' : 'input', save);
  if (!isCheckbox && (el.tagName === 'SELECT' || el.type === 'number')) {
    el.addEventListener('change', save);
  }
}

function initFCPersistentSettings() {
  fcSettings = loadFCSettings();
  applyThemeFC(normalizeThemeFC(fcSettings.theme || 'light'));
  bindFCPersistedField(fcWorkspaceDirEl, 'workspace_dir');
  bindFCPersistedField(fcAgentDefaultProviderEl, 'default_provider');
  bindFCPersistedField(fcCodexCliCmdEl, 'cli_adapter_cmd');
  bindFCPersistedField(fcClaudeCliCmdEl, 'claude_cli_adapter_cmd');
  bindFCPersistedField(fcOpenCodeCliCmdEl, 'opencode_cli_adapter_cmd');
  bindFCPersistedField(fcCliTimeoutSecondsEl, 'cli_timeout_seconds');
  bindFCPersistedField(fcClaudeCliTimeoutSecondsEl, 'claude_cli_timeout_seconds');
  bindFCPersistedField(fcOpenCodeCliTimeoutSecondsEl, 'opencode_cli_timeout_seconds');
  bindFCPersistedField(fcSubmitExecutionModeEl, 'submit_execution_mode');
  bindFCPersistedField(fcCodexOutputDirEl, 'codex_output_dir');
  bindFCPersistedField(fcCodexSandboxEl, 'codex_sandbox');
  bindFCPersistedField(fcCodexApprovalEl, 'codex_approval_policy');
  bindFCPersistedField(fcCodexSkipRepoCheckEl, 'codex_skip_git_repo_check', { checkbox: true });
  bindFCPersistedField(fcCodexProfileEl, 'codex_profile');
  bindFCPersistedField(fcCodexModelEl, 'codex_model');
  bindFCPersistedField(fcCodexReasoningEl, 'codex_reasoning_effort');
  bindFCPersistedField(fcCodexAPIBaseURLEl, 'openai_base_url');
  bindFCPersistedField(fcCodexAPITimeoutSecondsEl, 'codex_api_timeout_seconds');
  bindFCPersistedField(fcCodexAPIOrgEl, 'openai_org_id');
  bindFCPersistedField(fcCodexAPIProjectEl, 'openai_project_id');
  bindFCPersistedField(fcClaudeAPIBaseURLEl, 'anthropic_base_url');
  bindFCPersistedField(fcClaudeAPITimeoutSecondsEl, 'claude_api_timeout_seconds');
  bindFCPersistedField(fcClaudeAPIModelEl, 'claude_api_model');
  bindFCPersistedField(fcDeliveryIntentProfileEl, 'delivery_intent_profile');
  bindFCPersistedField(fcDeliveryInstructionTextEl, 'delivery_instruction_text');
  bindFCPersistedField(fcPostSubmitRebuildCmdEl, 'post_submit_rebuild_cmd');
  bindFCPersistedField(fcPostSubmitVerifyCmdEl, 'post_submit_verify_cmd');
  bindFCPersistedField(fcPostSubmitTimeoutSecEl, 'post_submit_timeout_seconds');
  bindFCPersistedField(fcAudioModeEl, 'audio_mode', { afterSave: renderSensitiveCaptureBadgesFC });
  bindFCPersistedField(fcAllowLargeInlineMediaToggleEl, 'allow_large_inline_media', { checkbox: true, afterSave: renderSensitiveCaptureBadgesFC });
  bindFCPersistedField(fcSttModeEl, 'stt_mode');
  bindFCPersistedField(fcSttBaseURLEl, 'stt_base_url');
  bindFCPersistedField(fcSttModelEl, 'stt_model');
  bindFCPersistedField(fcSttFasterWhisperModelEl, 'stt_model');
  bindFCPersistedField(fcSttDeviceEl, 'stt_device');
  bindFCPersistedField(fcSttComputeTypeEl, 'stt_compute_type');
  bindFCPersistedField(fcSttLanguageEl, 'stt_language');
  bindFCPersistedField(fcSttLocalCommandEl, 'stt_local_command');
  bindFCPersistedField(fcSttTimeoutSecondsEl, 'stt_timeout_seconds');
}

function authHeaders(isMutation = false) {
  const headers = { 'X-Knit-Token': controlToken };
  if (isMutation) {
    const nonce = (self.crypto && self.crypto.randomUUID) ? self.crypto.randomUUID() : (String(Date.now()) + Math.random());
    headers['X-Knit-Nonce'] = nonce;
    headers['X-Knit-Timestamp'] = String(Date.now());
  }
  return headers;
}

function companionSnippetFC() {
  return "(() => { const s = document.createElement('script'); s.src = '" + location.origin + "/companion.js?token=" + encodeURIComponent(controlToken) + "'; document.head.appendChild(s); })();";
}

function showToastFC(msg, isError = false) {
  if (!fcToastEl) return;
  fcToastEl.textContent = msg;
  fcToastEl.classList.toggle('error', !!isError);
  fcToastEl.classList.add('visible');
  if (fcToastTimer) {
    clearTimeout(fcToastTimer);
  }
  fcToastTimer = window.setTimeout(() => {
    fcToastEl.classList.remove('visible');
  }, 2200);
}

function isTerminalSubmitStatusFC(status) {
  const value = String(status || '').trim();
  return value === 'submitted' || value === 'failed';
}

function truncateSubmitToastTextFC(value, limit = 88) {
  const text = String(value || '').trim().replace(/\s+/g, ' ');
  if (!text) return '';
  if (text.length <= limit) return text;
  return text.slice(0, Math.max(0, limit - 1)).trimEnd() + '...';
}

function submitAttemptToastMessageFC(attempt) {
  const status = String(attempt?.status || '').trim();
  const request = truncateSubmitToastTextFC(String(attempt?.request_preview || ''));
  const attemptID = String(attempt?.attempt_id || '').trim();
  if (status === 'failed') {
    if (request) return 'Request failed: ' + request;
    return 'Request failed' + (attemptID ? ' (' + attemptID + ')' : '.');
  }
  if (request) return 'Request completed: ' + request;
  return 'Request completed' + (attemptID ? ' (' + attemptID + ')' : '.');
}

function notifySubmitAttemptTransitionsFC(attempts) {
  const nextStatuses = new Map();
  const list = Array.isArray(attempts) ? attempts : [];
  if (!fcSubmitAttemptNotificationsReady) {
    list.forEach(attempt => {
      const attemptID = String(attempt?.attempt_id || '').trim();
      if (!attemptID) return;
      nextStatuses.set(attemptID, String(attempt?.status || '').trim());
    });
    fcSubmitAttemptStatusByID = nextStatuses;
    fcSubmitAttemptNotificationsReady = true;
    return;
  }
  list.forEach(attempt => {
    const attemptID = String(attempt?.attempt_id || '').trim();
    if (!attemptID) return;
    const status = String(attempt?.status || '').trim();
    const prevStatus = String(fcSubmitAttemptStatusByID.get(attemptID) || '').trim();
    const watched = fcWatchedSubmitAttemptIDs.has(attemptID);
    const transitioned = prevStatus !== status;
    if (isTerminalSubmitStatusFC(status) && transitioned && (watched || prevStatus)) {
      showToastFC(submitAttemptToastMessageFC(attempt), status === 'failed');
      fcWatchedSubmitAttemptIDs.delete(attemptID);
    }
    nextStatuses.set(attemptID, status);
  });
  fcSubmitAttemptStatusByID = nextStatuses;
}

function isCompanionAttachedFC() {
  const status = String(lastState?.capture_sources?.companion?.status || '').toLowerCase();
  return status === 'available';
}

function hasWorkspaceSelectedFC() {
  return String(fcWorkspaceDirEl?.value || lastState?.runtime_codex?.codex_workdir || '').trim().length > 0;
}

function captureBlockedReasonFC(kind = 'audio') {
  if (!hasWorkspaceSelectedFC()) {
    return 'Choose a workspace first.';
  }
  if (!lastState?.session?.id) {
    return 'Start a session on the main Knit page first.';
  }
  if ((kind === 'snapshot' || kind === 'video') && !isCompanionAttachedFC()) {
    return 'Connect browser first.';
  }
  return '';
}

function ensureCaptureReadyFC(kind = 'audio') {
  const reason = captureBlockedReasonFC(kind);
  if (!reason) {
    return true;
  }
  setStatus(reason, true);
  return false;
}

function requireCompanionFC(action) {
  if (isCompanionAttachedFC()) {
    return true;
  }
  setStatus('Browser companion is required to ' + action + '. Click 🔗, run snippet in target-tab DevTools, then retry.', true);
  return false;
}

async function copyCompanionSnippetFC() {
  const snippet = companionSnippetFC();
  try {
    await navigator.clipboard.writeText(snippet);
    setStatus('Companion snippet copied. Run it in target-tab DevTools.');
    showToastFC('Connect browser link copied');
  } catch {
    try {
      window.prompt('Copy Browser Companion snippet:', snippet);
      setStatus('Copy snippet from prompt and run in target-tab DevTools.');
      showToastFC('Connect browser link ready to copy');
    } catch {
      setStatus('Clipboard copy failed. Browser blocked prompt fallback.', true);
      showToastFC('Could not copy the browser link', true);
    }
  }
}

function setStatus(msg, isError=false) {
  setStatusChipFC(statusLineEl, 'Status', msg, isError ? 'error' : 'ok');
}

function setPreviewState(msg, isError = false) {
  if (!previewStateLineEl) return;
  previewStateLineEl.textContent = msg;
  previewStateLineEl.style.color = isError ? '#c34f4f' : '#6a7383';
}

function markPreviewStale(reason = 'Preview needs refresh after your latest note.') {
  resetPreviewDeliveryOptionsFC();
  setPreviewState(reason, false);
}

function clearSubmittedPreviewFC() {
  resetPreviewDeliveryOptionsFC();
  fcLatestPayloadPreviewData = null;
  if (!fcPayloadPreviewEl) return;
  fcPayloadPreviewEl.className = 'request-preview small';
  fcPayloadPreviewEl.textContent = 'Request queued. Capture another note, then preview the next request here.';
  if (fcPreviewDetailsEl) fcPreviewDetailsEl.open = false;
}

function resetPreviewDeliveryOptionsFC() {
  fcPreviewDeliveryOptions = { redactReplayValues: false, omitVideoClips: false, omitVideoEventIDs: [] };
}

function renderRuntimeGuideFC(data) {
  if (!runtimeGuideLineEl) return;
  const runtimePlatform = data?.runtime_platform || {};
  const profile = data?.platform_profile || {};
  const summary = String(runtimePlatform.runtime_summary || '').trim();
  const hostTarget = String(runtimePlatform.host_target || '').trim();
  const displayName = String(profile.display_name || 'Current OS').trim();
  let text = summary || (displayName + ' runtime ready.');
  if (hostTarget) text += ' Host target: ' + hostTarget + '.';
  setStatusChipFC(runtimeGuideLineEl, 'Runtime', text, '');
}

function sensitiveBadgeFC(label, value, tone = '') {
  const toneClass = tone === 'ok' ? ' ok' : '';
  return '<span class="status-chip' + toneClass + '" title="' + escapePreviewHTML(label + ': ' + value) + '"><span class="status-chip-label">' + escapePreviewHTML(label) + '</span><span class="status-chip-text">' + escapePreviewHTML(value) + '</span></span>';
}

function setStatusChipFC(el, label, text, tone = '') {
  if (!el) return;
  const safeLabel = escapePreviewHTML(label);
  const rawText = String(text || '').trim() || 'unknown';
  el.className = 'status-chip' + (tone ? ' ' + tone : '');
  el.title = label + ': ' + rawText;
  el.innerHTML = '<span class="status-chip-label">' + safeLabel + '</span><span class="status-chip-text">' + escapePreviewHTML(rawText) + '</span>';
}

function replayTypedValueStatusLabelFC() {
  return !!(lastState?.session?.capture_input_values) ? 'on' : 'redacted';
}

function largeMediaStatusLabelFC() {
  return !!fcSettings.allow_large_inline_media ? 'allowed' : 'ask first';
}

function currentVideoModeLabelFC() {
  return String(lastState?.video_mode || 'event_triggered').trim() || 'event_triggered';
}

function currentAudioModeLabelFC() {
  return String(lastState?.audio?.state?.mode || fcAudioModeEl?.value || 'always_on').trim() || 'always_on';
}

function renderSensitiveCaptureBadgesFC() {
  if (!fcSensitiveCaptureBadgesEl) return;
  fcSensitiveCaptureBadgesEl.innerHTML = [
    sensitiveBadgeFC('Replay typed values', replayTypedValueStatusLabelFC(), lastState?.session?.capture_input_values ? 'ok' : ''),
    sensitiveBadgeFC('Large media', largeMediaStatusLabelFC(), fcSettings.allow_large_inline_media ? 'ok' : ''),
    sensitiveBadgeFC('Video mode', currentVideoModeLabelFC()),
    sensitiveBadgeFC('Audio mode', currentAudioModeLabelFC())
  ].join('');
}

function disclosureStatusLabelFC(value) {
  switch (String(value || '').trim()) {
    case 'included':
      return 'Included';
    case 'redacted':
      return 'Redacted';
    case 'mixed':
      return 'Mixed';
    default:
      return 'Not used';
  }
}

function renderDisclosureSummaryFC(preview) {
  const disclosure = preview?.disclosure;
  if (!disclosure || typeof disclosure !== 'object') return '';
  const requestCount = Number(disclosure.request_text_count || 0);
  const actions = [];
  const typedStatus = String(disclosure.typed_values_status || '').trim();
  if (typedStatus === 'included' || typedStatus === 'mixed' || fcPreviewDeliveryOptions.redactReplayValues) {
    const label = fcPreviewDeliveryOptions.redactReplayValues ? 'Send typed values again' : 'Redact typed values for this preview';
    actions.push('<button type="button" onclick="togglePreviewReplayRedactionFC()" title="' + escapePreviewHTML(label) + '">' + escapePreviewHTML(label) + '</button>');
  }
  const videoCount = Number(disclosure.videos_sent || 0) + Number(disclosure.videos_omitted || 0);
  if (videoCount > 0 || fcPreviewDeliveryOptions.omitVideoClips) {
    const label = fcPreviewDeliveryOptions.omitVideoClips
      ? 'Send clips again'
      : (Number(disclosure.screenshots_sent || 0) > 0 ? 'Use snapshot instead of clip' : 'Omit clip for this request');
    actions.push('<button type="button" onclick="togglePreviewVideoDeliveryFC()" title="' + escapePreviewHTML(label) + '">' + escapePreviewHTML(label) + '</button>');
  }
  return '<div class="preview-warning-card">' +
    '<strong>What will be sent</strong>' +
    '<div class="preview-note-text" style="margin-top:.35rem;">' +
      '<div>Destination: ' + escapePreviewHTML(String(disclosure.destination || 'local adapter')) + '</div>' +
      '<div>Request text: ' + escapePreviewHTML(String(requestCount)) + ' change request' + (requestCount === 1 ? '' : 's') + '</div>' +
      '<div>Typed values: ' + escapePreviewHTML(disclosureStatusLabelFC(disclosure.typed_values_status)) + '</div>' +
      '<div>Screenshots: ' + escapePreviewHTML(String(disclosure.screenshots_sent || 0)) + ' sent' + (Number(disclosure.screenshots_omitted || 0) > 0 ? ', ' + escapePreviewHTML(String(disclosure.screenshots_omitted || 0)) + ' omitted' : '') + '</div>' +
      '<div>Video clips: ' + escapePreviewHTML(String(disclosure.videos_sent || 0)) + ' sent' + (Number(disclosure.videos_omitted || 0) > 0 ? ', ' + escapePreviewHTML(String(disclosure.videos_omitted || 0)) + ' omitted' : '') + '</div>' +
      '<div>Audio clips: ' + escapePreviewHTML(String(disclosure.audio_sent || 0)) + ' sent' + (Number(disclosure.audio_omitted || 0) > 0 ? ', ' + escapePreviewHTML(String(disclosure.audio_omitted || 0)) + ' omitted' : '') + '</div>' +
    '</div>' +
    (actions.length ? '<div class="mini-toolbar" style="margin-top:.55rem;">' + actions.join('') + '</div>' : '') +
  '</div>';
}

async function togglePreviewReplayRedactionFC() {
  fcPreviewDeliveryOptions.redactReplayValues = !fcPreviewDeliveryOptions.redactReplayValues;
  await previewPayloadFC();
}

async function togglePreviewVideoDeliveryFC() {
  fcPreviewDeliveryOptions.omitVideoClips = !fcPreviewDeliveryOptions.omitVideoClips;
  await previewPayloadFC();
}

function previewVideoEventOmittedFC(eventID) {
  const id = String(eventID || '').trim();
  return !!id && Array.isArray(fcPreviewDeliveryOptions.omitVideoEventIDs) && fcPreviewDeliveryOptions.omitVideoEventIDs.includes(id);
}

function setPreviewVideoEventOmittedFC(eventID, omitted) {
  const id = String(eventID || '').trim();
  const current = Array.isArray(fcPreviewDeliveryOptions.omitVideoEventIDs) ? fcPreviewDeliveryOptions.omitVideoEventIDs.slice() : [];
  const next = current.filter(item => item !== id);
  if (omitted && id) {
    next.push(id);
  }
  fcPreviewDeliveryOptions.omitVideoEventIDs = next;
}

async function togglePreviewVideoEventOmissionFC(eventID) {
  const id = String(eventID || '').trim();
  if (!id) return;
  setPreviewVideoEventOmittedFC(id, !previewVideoEventOmittedFC(id));
  await previewPayloadFC();
}

function oversizedVideoPreviewNotesFC(preview) {
  const notes = Array.isArray(preview?.notes) ? preview.notes : [];
  return notes.filter(note => previewNoteNeedsClipResizeFC(note));
}

function renderOversizedVideoWarningActionsFC(preview) {
  const notes = oversizedVideoPreviewNotesFC(preview);
  if (!notes.length) return '';
  const items = notes.map((note) => {
    const eventID = String(note?.event_id || '').trim();
    const useSnapshotLabel = previewVideoEventOmittedFC(eventID)
      ? 'Send clip again'
      : (note?.has_screenshot ? 'Use snapshot instead' : 'Omit clip for this request');
    return '<div class="preview-note-card" style="margin-top:.6rem;">' +
      '<div class="preview-note-header"><strong>' + escapePreviewHTML(eventID || 'change request') + '</strong><span class="small">' + escapePreviewHTML(formatMediaSizeFC(note?.video_size_bytes || 0) + ' over ' + formatMediaSizeFC(note?.video_send_limit_bytes || 0)) + '</span></div>' +
      '<div class="preview-note-text">' + escapePreviewHTML(String(note?.video_transmission_note || 'This clip is too large to send with the current inline media setting.')) + '</div>' +
      '<div class="mini-toolbar" style="margin-top:.55rem;">' +
      '<button type="button" ' + (fcClipResizeInFlight.has(eventID) ? 'disabled ' : '') + 'onclick="fitPreviewClipToSendLimitFC(\'' + escapePreviewHTML(eventID) + '\')" title="Make clip smaller to send">' + escapePreviewHTML(fcClipResizeInFlight.has(eventID) ? 'Making clip smaller…' : 'Make clip smaller to send') + '</button>' +
      '<button type="button" onclick="togglePreviewVideoEventOmissionFC(\'' + escapePreviewHTML(eventID) + '\')" title="' + escapePreviewHTML(useSnapshotLabel) + '">' + escapePreviewHTML(useSnapshotLabel) + '</button>' +
      '</div>' +
      '</div>';
  }).join('');
  return '<div class="preview-warning-card">' +
    '<strong>Large clip needs a decision</strong>' +
    '<div class="preview-note-text" style="margin-top:.35rem;">Choose how to handle the affected request before you submit.</div>' +
    items +
  '</div>';
}

function hasTypedNoteDraftFC() {
  return String(transcriptEl?.value || '').trim().length > 0;
}

async function flushTypedNoteDraftFC(reason) {
  if (!hasTypedNoteDraftFC()) {
    return false;
  }
  await submitTextNote();
  if (hasTypedNoteDraftFC()) {
    throw new Error('The typed note could not be added before ' + reason + '.');
  }
  return true;
}

function setButtonsDisabled(disabled) {
  [composerSettingsBtnEl, openWorkspaceBtnEl, copyCompanionBtnEl, openVideoCaptureBtnEl, toggleTextEditorBtnEl, submitTextSnapshotBtnEl, talkOnlyBtnEl, snapshotTalkBtnEl, videoTalkBtnEl, openAudioControlsBtnEl, openCodexRuntimeBtnEl, previewPayloadBtnEl, sendComposerBtnEl, openLastLogBtnEl, startLiveVideoBtnEl, stopLiveVideoBtnEl, captureScreenshotBtnEl].forEach(btn => {
    if (!btn) return;
    btn.disabled = !!disabled;
  });
  if (clearScreenshotBtnEl) {
    clearScreenshotBtnEl.disabled = !!disabled || !queuedScreenshotBlob;
  }
}

function updatePreviewSubmitButtonsFC() {
  const blocked = !!inFlight || !!recording;
  if (previewPayloadBtnEl) previewPayloadBtnEl.disabled = blocked;
  if (sendComposerBtnEl) sendComposerBtnEl.disabled = blocked;
}

function setBusy(active) {
  inFlight = !!active;
  setButtonsDisabled(inFlight);
  updatePreviewSubmitButtonsFC();
}

function setIconToolStateFC(btn, icon, title, options = {}) {
  if (!btn) return;
  btn.disabled = !!options.disabled;
  btn.textContent = icon;
  btn.title = title;
  btn.setAttribute('aria-label', title);
  btn.classList.toggle('active', !!options.active);
}

function resetCaptureActionButtons() {
  const audioBlocked = captureBlockedReasonFC('audio');
  const snapshotBlocked = captureBlockedReasonFC('snapshot');
  const videoBlocked = captureBlockedReasonFC('video');
  setIconToolStateFC(talkOnlyBtnEl, '🎙️', audioBlocked ? ('Talk only unavailable. ' + audioBlocked) : 'Talk only. Record a voice note without a screenshot.', { disabled: inFlight || !!audioBlocked });
  setIconToolStateFC(snapshotTalkBtnEl, '📸', snapshotBlocked ? ('Snapshot plus voice unavailable. ' + snapshotBlocked) : 'Snapshot plus voice. Capture a snapshot and record a voice note.', { disabled: inFlight || !!snapshotBlocked });
  setIconToolStateFC(videoTalkBtnEl, '🎬', videoBlocked ? ('Video plus voice unavailable. ' + videoBlocked) : 'Video plus voice. Record a video clip with your voice.', { disabled: inFlight || !!videoBlocked });
}

function setRecordingState(active, secondsRemaining = 0) {
  recording = !!active;
  updatePreviewSubmitButtonsFC();
  if (!recordLineEl) return;
  if (!recording) {
    fcRecordingKind = '';
    fcAudioNoteMode = '';
    setStatusChipFC(recordLineEl, 'Recorder', 'Idle', '');
    resetCaptureActionButtons();
    return;
  }
  if (fcRecordingKind === 'audio') {
    setStatusChipFC(recordLineEl, 'Recorder', 'Recording voice note', 'warn');
    if (fcAudioNoteMode === 'snapshot') {
      setIconToolStateFC(talkOnlyBtnEl, '🎙️', 'Talk only. Record a voice note without a screenshot.', { disabled: true });
      setIconToolStateFC(snapshotTalkBtnEl, '■', 'Stop recording snapshot plus voice note.', { active: true });
    } else {
      setIconToolStateFC(talkOnlyBtnEl, '■', 'Stop recording voice note.', { active: true });
      setIconToolStateFC(snapshotTalkBtnEl, '📸', 'Snapshot plus voice. Capture a snapshot and record a voice note.', { disabled: true });
    }
    setIconToolStateFC(videoTalkBtnEl, '🎬', 'Video plus voice. Record a video clip with your voice.', { disabled: true });
    return;
  }
  if (fcRecordingKind === 'video') {
    setStatusChipFC(recordLineEl, 'Recorder', 'Recording video note', 'warn');
    setIconToolStateFC(talkOnlyBtnEl, '🎙️', 'Talk only. Record a voice note without a screenshot.', { disabled: true });
    setIconToolStateFC(snapshotTalkBtnEl, '📸', 'Snapshot plus voice. Capture a snapshot and record a voice note.', { disabled: true });
    setIconToolStateFC(videoTalkBtnEl, '■', 'Stop recording video note.', { active: true });
    return;
  }
  setStatusChipFC(recordLineEl, 'Recorder', 'Recording... ' + secondsRemaining + 's left', 'warn');
  setIconToolStateFC(talkOnlyBtnEl, '🎙️', 'Talk only. Record a voice note without a screenshot.', { disabled: true });
  setIconToolStateFC(snapshotTalkBtnEl, '📸', 'Snapshot plus voice. Capture a snapshot and record a voice note.', { disabled: true });
  setIconToolStateFC(videoTalkBtnEl, '🎬', 'Video plus voice. Record a video clip with your voice.', { disabled: true });
}

function setPTTFC(active) {
  fcPTTHeld = !!active;
}

function renderTextEditorStateFC() {
  if (!transcriptEl || !toggleTextEditorBtnEl) return;
  transcriptEl.hidden = !fcTextEditorOpen;
  setIconToolStateFC(
    toggleTextEditorBtnEl,
    fcTextEditorOpen ? '⌨️' : '✏️',
    fcTextEditorOpen ? 'Hide typed note field.' : 'Type note. Show the typed note field.',
    { active: fcTextEditorOpen, disabled: inFlight }
  );
}

function toggleTextEditorFC() {
  fcTextEditorOpen = !fcTextEditorOpen;
  renderTextEditorStateFC();
  if (fcTextEditorOpen) {
    window.setTimeout(() => {
      try { transcriptEl?.focus(); } catch (_) {}
    }, 0);
  }
}

function ensureTextEditorOpenFC() {
  if (fcTextEditorOpen) return;
  fcTextEditorOpen = true;
  renderTextEditorStateFC();
  window.setTimeout(() => {
    try { transcriptEl?.focus(); } catch (_) {}
  }, 0);
}

function updateWorkspaceModalStateFC() {
  const selected = (fcWorkspaceDirEl?.value || '').trim();
  const locked = !!lastState?.config_locked;
  fcWorkspaceRequired = !locked && !selected;
  if (fcWorkspaceCloseBtnEl) {
    fcWorkspaceCloseBtnEl.disabled = fcWorkspaceRequired;
    fcWorkspaceCloseBtnEl.title = fcWorkspaceRequired ? 'Select a workspace first.' : 'Close';
  }
  if (fcWorkspaceStatusEl) {
    if (locked) {
      fcWorkspaceStatusEl.textContent = 'Workspace is managed by policy in this environment.';
      fcWorkspaceStatusEl.style.color = '#f6ad55';
    } else if (fcWorkspaceRequired) {
      fcWorkspaceStatusEl.textContent = 'Workspace selection required before continuing.';
      fcWorkspaceStatusEl.style.color = '#f6ad55';
    } else {
      fcWorkspaceStatusEl.textContent = 'Workspace selected: ' + selected;
      fcWorkspaceStatusEl.style.color = '#48bb78';
    }
  }
  if (fcWorkspaceStateEl) {
    fcWorkspaceStateEl.textContent = JSON.stringify({
      selected_workspace: selected,
      picker: 'native folder dialog'
    }, null, 2);
  }
  syncComposerSettingsSummaryFC();
}

function syncComposerSettingsSummaryFC() {
  const workspace = String(fcWorkspaceDirEl?.value || lastState?.runtime_codex?.codex_workdir || '').trim();
  const provider = String(fcAgentDefaultProviderEl?.value || lastState?.runtime_codex?.default_provider || 'codex_cli').trim() || 'codex_cli';
  if (fcSettingsWorkspaceLabelEl) {
    fcSettingsWorkspaceLabelEl.textContent = workspace || '(not set)';
  }
  if (fcSettingsProviderLabelEl) {
    fcSettingsProviderLabelEl.textContent = provider;
  }
}

function openComposerSettingsModalFC() {
  if (!composerSettingsModalEl) return;
  syncComposerSettingsSummaryFC();
  composerSettingsModalEl.classList.add('open');
}

function closeComposerSettingsModalFC() {
  if (!composerSettingsModalEl) return;
  composerSettingsModalEl.classList.remove('open');
}

function onComposerSettingsModalBackdropFC(event) {
  if (event && event.target === composerSettingsModalEl) {
    closeComposerSettingsModalFC();
  }
}

function openDocsBrowserFC(name) {
  closeComposerSettingsModalFC();
  const url = new URL(window.location.origin + '/docs');
  if (controlToken) url.searchParams.set('token', controlToken);
  if (name) url.searchParams.set('name', String(name).trim());
  const tab = window.open(url.pathname + url.search, 'knitDocs', 'noopener');
  if (!tab) {
    showToastFC('Docs tab blocked. Allow popups for 127.0.0.1 and try again.', true);
    return;
  }
  try { tab.focus(); } catch (_) {}
}

function openWorkspaceFromSettingsFC() {
  closeComposerSettingsModalFC();
  openWorkspaceModal();
}

function openCodexRuntimeFromSettingsFC() {
  closeComposerSettingsModalFC();
  openCodexRuntimeModalFC();
}

function openWorkspaceModal() {
  if (!workspaceModalEl) return;
  closeComposerSettingsModalFC();
  workspaceModalEl.classList.add('open');
  updateWorkspaceModalStateFC();
}

function closeWorkspaceModal() {
  if (!workspaceModalEl) return;
  updateWorkspaceModalStateFC();
  if (fcWorkspaceRequired) {
    setStatus('Select a workspace folder before closing this modal.', true);
    return;
  }
  workspaceModalEl.classList.remove('open');
}

function onWorkspaceModalBackdrop(event) {
  if (event && event.target === workspaceModalEl) {
    closeWorkspaceModal();
  }
}

async function applyWorkspaceFC() {
  try {
    if (!lastState) {
      await refreshState();
    }
    if (lastState?.config_locked) throw new Error('Config is locked by policy.');
    const path = (fcWorkspaceDirEl?.value || '').trim();
    if (!path) throw new Error('Select a workspace first.');
    setFCSetting('workspace_dir', path);
    const rc = lastState?.runtime_codex || {};
    const payload = {
      default_provider: String(fcAgentDefaultProviderEl?.value || rc.default_provider || 'codex_cli'),
      cli_adapter_cmd: String((fcCodexCliCmdEl?.value || rc.cli_adapter_cmd || '')).trim(),
      claude_cli_adapter_cmd: String((fcClaudeCliCmdEl?.value || rc.claude_cli_adapter_cmd || '')).trim(),
      opencode_cli_adapter_cmd: String((fcOpenCodeCliCmdEl?.value || rc.opencode_cli_adapter_cmd || '')).trim(),
      submit_execution_mode: String(fcSubmitExecutionModeEl?.value || rc.submit_execution_mode || 'series'),
      codex_workdir: path,
      codex_output_dir: String((fcCodexOutputDirEl?.value || rc.codex_output_dir || '')).trim(),
      codex_sandbox: String(fcCodexSandboxEl?.value || rc.codex_sandbox || ''),
      codex_approval_policy: String(fcCodexApprovalEl?.value || rc.codex_approval_policy || ''),
      codex_profile: String((fcCodexProfileEl?.value || rc.codex_profile || '')).trim(),
      codex_model: String((fcCodexModelEl?.value || rc.codex_model || '')).trim(),
      codex_reasoning_effort: String((fcCodexReasoningEl?.value || rc.codex_reasoning_effort || '')).trim(),
      openai_base_url: String((fcCodexAPIBaseURLEl?.value || rc.openai_base_url || '')).trim(),
      openai_org_id: String((fcCodexAPIOrgEl?.value || rc.openai_org_id || '')).trim(),
      openai_project_id: String((fcCodexAPIProjectEl?.value || rc.openai_project_id || '')).trim(),
      post_submit_rebuild_cmd: String((fcPostSubmitRebuildCmdEl?.value || rc.post_submit_rebuild_cmd || '')).trim(),
      post_submit_verify_cmd: String((fcPostSubmitVerifyCmdEl?.value || rc.post_submit_verify_cmd || '')).trim(),
      codex_skip_git_repo_check: !!(fcCodexSkipRepoCheckEl?.checked ?? (String(rc.codex_skip_git_repo_check || '1') !== '0'))
    };
    const timeoutSeconds = Number.parseInt(String(fcCliTimeoutSecondsEl?.value || rc.cli_timeout_seconds || '0'), 10);
    const claudeTimeoutSeconds = Number.parseInt(String(fcClaudeCliTimeoutSecondsEl?.value || rc.claude_cli_timeout_seconds || '0'), 10);
    const opencodeTimeoutSeconds = Number.parseInt(String(fcOpenCodeCliTimeoutSecondsEl?.value || rc.opencode_cli_timeout_seconds || '0'), 10);
    const codexAPITimeoutSeconds = Number.parseInt(String(fcCodexAPITimeoutSecondsEl?.value || rc.codex_api_timeout_seconds || '0'), 10);
    const postTimeoutSeconds = Number.parseInt(String(fcPostSubmitTimeoutSecEl?.value || rc.post_submit_timeout_seconds || '0'), 10);
    if (Number.isFinite(timeoutSeconds) && timeoutSeconds > 0) {
      payload.cli_timeout_seconds = timeoutSeconds;
    }
    if (Number.isFinite(claudeTimeoutSeconds) && claudeTimeoutSeconds > 0) {
      payload.claude_cli_timeout_seconds = claudeTimeoutSeconds;
    }
    if (Number.isFinite(opencodeTimeoutSeconds) && opencodeTimeoutSeconds > 0) {
      payload.opencode_cli_timeout_seconds = opencodeTimeoutSeconds;
    }
    if (Number.isFinite(codexAPITimeoutSeconds) && codexAPITimeoutSeconds > 0) {
      payload.codex_api_timeout_seconds = codexAPITimeoutSeconds;
    }
    if (Number.isFinite(postTimeoutSeconds) && postTimeoutSeconds > 0) {
      payload.post_submit_timeout_seconds = postTimeoutSeconds;
    }
    await postJSON('/api/runtime/codex', payload);
    await refreshState();
    updateWorkspaceModalStateFC();
    closeWorkspaceModal();
    setStatus('Workspace selected: ' + path);
  } catch (err) {
    setStatus('Apply workspace failed: ' + err.message, true);
  }
}

async function pickWorkspaceDirFC() {
  try {
    if (lastState?.config_locked) throw new Error('Config is locked by policy.');
    const picked = await postJSON('/api/fs/pickdir', {});
    const path = (picked && picked.path) ? String(picked.path) : '';
    if (!path) throw new Error('No folder selected.');
    fcWorkspaceDirEl.value = path;
    setFCSetting('workspace_dir', path);
    updateWorkspaceModalStateFC();
    await applyWorkspaceFC();
  } catch (err) {
    setStatus('Choose folder failed: ' + err.message, true);
  }
}

function openAudioControlsModal() {
  if (!audioControlsModalEl) return;
  audioControlsModalEl.classList.add('open');
  refreshState();
  refreshAudioDevicesFC();
}

function closeAudioControlsModal() {
  if (!audioControlsModalEl) return;
  audioControlsModalEl.classList.remove('open');
  setPTTFC(false);
}

function onAudioControlsModalBackdrop(event) {
  if (event && event.target === audioControlsModalEl) {
    closeAudioControlsModal();
  }
}

function openTranscriptionRuntimeFromAudioModalFC() {
  closeAudioControlsModal();
  openTranscriptionRuntimeModal();
}

function openTranscriptionRuntimeModal() {
  if (!transcriptionRuntimeModalEl) return;
  transcriptionRuntimeModalEl.classList.add('open');
  refreshState();
  syncFCTranscriptionUI(lastState || {});
}

function closeTranscriptionRuntimeModal() {
  if (!transcriptionRuntimeModalEl) return;
  transcriptionRuntimeModalEl.classList.remove('open');
}

function onTranscriptionRuntimeModalBackdrop(event) {
  if (event && event.target === transcriptionRuntimeModalEl) {
    closeTranscriptionRuntimeModal();
  }
}

function openCodexRuntimeModalFC() {
  if (!codexRuntimeModalEl) return;
  closeComposerSettingsModalFC();
  codexRuntimeModalEl.classList.add('open');
  refreshState();
  syncDeliveryIntentPromptTextFC(false);
  syncFCProviderOptions(lastState || {});
  syncFCCodexRuntimeUI(lastState || {});
  syncFCCodexRuntimeModeUI();
}

function closeCodexRuntimeModalFC() {
  if (!codexRuntimeModalEl) return;
  codexRuntimeModalEl.classList.remove('open');
}

function onCodexRuntimeModalBackdropFC(event) {
  if (event && event.target === codexRuntimeModalEl) {
    closeCodexRuntimeModalFC();
  }
}

function setCodexRuntimeStatusFC(message, isError = false) {
  if (!fcCodexOptionsStateEl) return;
  fcCodexOptionsStateEl.textContent = message;
  fcCodexOptionsStateEl.style.color = isError ? '#c34f4f' : '#1c7c74';
}

function syncFCCodexRuntimeModeUI() {
  const provider = selectedProviderFC();
  const toggle = (el, visible) => {
    if (el) el.classList.toggle('hidden', !visible);
  };
  toggle(fcCodexCliSectionEl, provider === 'codex_cli');
  toggle(fcClaudeCliSectionEl, provider === 'claude_cli');
  toggle(fcOpenCodeCliSectionEl, provider === 'opencode_cli');
  toggle(fcCodexAPISectionEl, provider === 'codex_api');
  toggle(fcClaudeAPISectionEl, provider === 'claude_api');
  toggle(fcCodexCommonSectionEl, provider === 'codex_cli' || provider === 'codex_api');
  toggle(fcCodexCLIDefaultsSectionEl, provider === 'codex_cli');
  toggle(fcCodexSharedSectionEl, true);
  if (fcRuntimeProviderHelpEl) {
    switch (provider) {
    case 'codex_api':
      fcRuntimeProviderHelpEl.textContent = 'Codex API uses a base URL, API timeout, and optional OpenAI org/project IDs. Shared submission settings still apply.';
      break;
    case 'claude_api':
      fcRuntimeProviderHelpEl.textContent = 'Claude API uses Anthropic base URL, model, and timeout settings. Codex-only sandbox, approval, reasoning, and org/project fields stay hidden.';
      break;
    case 'claude_cli':
      fcRuntimeProviderHelpEl.textContent = 'Claude CLI only needs its command and timeout here. Codex-only sandbox, approval, model, and reasoning settings stay hidden.';
      break;
    case 'opencode_cli':
      fcRuntimeProviderHelpEl.textContent = 'OpenCode CLI only needs its command and timeout here. Codex-only sandbox, approval, model, and reasoning settings stay hidden.';
      break;
    default:
      fcRuntimeProviderHelpEl.textContent = 'Codex CLI uses Knit defaults of workspace-write sandbox and never approval unless you explicitly choose different values here.';
      break;
    }
  }
  syncComposerSettingsSummaryFC();
}

function resetRuntimeFieldValidityFC(el) {
  if (el && typeof el.setCustomValidity === 'function') {
    el.setCustomValidity('');
  }
}

function invalidRuntimeFieldFC(el, message) {
  if (el && typeof el.setCustomValidity === 'function') {
    el.setCustomValidity(message);
    try { el.reportValidity(); } catch (_) {}
  }
  throw new Error(message);
}

function readRuntimeSingleLineFC(el, label, maxLen = 2048) {
  resetRuntimeFieldValidityFC(el);
  const value = String(el?.value || '').trim();
  if (!value) return '';
  if (/[\r\n]/.test(value)) {
    invalidRuntimeFieldFC(el, label + ' must stay on one line.');
  }
  if (value.length > maxLen) {
    invalidRuntimeFieldFC(el, label + ' must be ' + maxLen + ' characters or fewer.');
  }
  return value;
}

function readRuntimeURLFC(el, label, maxLen = 1024) {
  const value = readRuntimeSingleLineFC(el, label, maxLen);
  if (!value) return '';
  let parsed;
  try {
    parsed = new URL(value);
  } catch (_) {
    invalidRuntimeFieldFC(el, label + ' must be a valid http or https URL.');
  }
  if (!parsed || !/^https?:$/.test(parsed.protocol)) {
    invalidRuntimeFieldFC(el, label + ' must use http or https.');
  }
  return value;
}

function readRuntimeSecondsFC(el, label, maxSeconds = 3600) {
  resetRuntimeFieldValidityFC(el);
  const raw = String(el?.value || '').trim();
  if (!raw) return 0;
  const value = Number.parseInt(raw, 10);
  if (!Number.isFinite(value) || value < 1 || value > maxSeconds) {
    invalidRuntimeFieldFC(el, label + ' must be between 1 and ' + maxSeconds + '.');
  }
  return value;
}

function scheduleCodexRuntimeApplyFC() {
  fcCodexRuntimeDirty = true;
  syncFCCodexRuntimeModeUI();
  setCodexRuntimeStatusFC('Saving runtime settings...');
  if (fcCodexRuntimeApplyTimer) {
    clearTimeout(fcCodexRuntimeApplyTimer);
  }
  fcCodexRuntimeApplyTimer = window.setTimeout(() => {
    applyCodexRuntimeFC();
  }, 350);
}

function openVideoCaptureModalFC() {
  if (!videoCaptureModalEl) return;
  videoCaptureModalEl.classList.add('open');
  renderScreenshotQueueState();
}

function closeVideoCaptureModalFC() {
  if (!videoCaptureModalEl) return;
  stopLiveVideoFC(true);
  videoCaptureModalEl.classList.remove('open');
}

function onVideoCaptureModalBackdropFC(event) {
  if (event && event.target === videoCaptureModalEl) {
    closeVideoCaptureModalFC();
  }
}

function syncFCAudioUI(data) {
  const audioState = data?.audio?.state || {};
  const devices = Array.isArray(data?.audio?.devices) ? data.audio.devices : [];
  const preservingDraft = fcAudioDirty || fcAudioApplying;
  const selectedMode = String((preservingDraft ? fcAudioModeEl?.value : (audioState.mode || fcAudioModeEl?.value)) || 'always_on');
  if (fcAudioModeEl && !preservingDraft) fcAudioModeEl.value = audioState.mode || fcAudioModeEl.value || 'always_on';
  if (fcAudioMutedEl && !preservingDraft) fcAudioMutedEl.checked = !!audioState.muted;
  if (fcAudioPausedEl && !preservingDraft) fcAudioPausedEl.checked = !!audioState.paused;
  if (selectedMode !== 'push_to_talk' && fcPTTHeld) {
    setPTTFC(false);
  }
  if (fcAudioInputDeviceEl && !preservingDraft) {
    const curr = String(audioState.input_device_id || fcAudioInputDeviceEl.value || 'default');
    fcAudioInputDeviceEl.innerHTML = '';
    if (!devices.length) {
      const fallback = document.createElement('option');
      fallback.value = 'default';
      fallback.textContent = 'default';
      fcAudioInputDeviceEl.appendChild(fallback);
    } else {
      devices.forEach(d => {
        const opt = document.createElement('option');
        const id = String((d && d.id) || '').trim() || 'default';
        const label = String((d && d.label) || id).trim();
        opt.value = id;
        opt.textContent = label;
        fcAudioInputDeviceEl.appendChild(opt);
      });
    }
    fcAudioInputDeviceEl.value = curr;
  }
  setAudioLevelStateVisibleFC(fcMicTestRunning);
  if (fcAudioLevelStateEl) {
    const lvl = Number(audioState.last_level || 0);
    const valid = !!audioState.level_valid;
    const mode = String(audioState.mode || 'unknown');
    fcAudioLevelStateEl.textContent = 'Audio level: ' + lvl.toFixed(3) + ' | valid=' + valid + ' | mode=' + mode;
    fcAudioLevelStateEl.style.color = valid ? '#48bb78' : '#f6ad55';
  }
}

function setAudioLevelStateVisibleFC(visible) {
  if (!fcAudioLevelStateEl) return;
  fcAudioLevelStateEl.classList.toggle('hidden', !visible);
}

function syncFCTranscriptionUI(data) {
  const rt = data?.runtime_transcription || {};
  if (fcSttModeEl) fcSttModeEl.value = rt.mode || data?.transcription_mode || fcSttModeEl.value || 'faster_whisper';
  if (fcSttBaseURLEl) fcSttBaseURLEl.value = rt.endpoint || fcSttBaseURLEl.value || '';
  if (fcSttModelEl) fcSttModelEl.value = rt.model || fcSttModelEl.value || '';
  if (fcSttFasterWhisperModelEl) fcSttFasterWhisperModelEl.value = normalizeFasterWhisperModelFC(rt.model || fcSttModelEl?.value || fcSttFasterWhisperModelEl.value || '');
  if (fcSttDeviceEl) fcSttDeviceEl.value = rt.device || fcSttDeviceEl.value || '';
  if (fcSttComputeTypeEl) fcSttComputeTypeEl.value = rt.compute_type || fcSttComputeTypeEl.value || '';
  if (fcSttLanguageEl) fcSttLanguageEl.value = rt.language || fcSttLanguageEl.value || '';
  if (fcSttLocalCommandEl) fcSttLocalCommandEl.value = rt.local_command || fcSttLocalCommandEl.value || '';
  if (fcSttTimeoutSecondsEl) fcSttTimeoutSecondsEl.value = rt.timeout_seconds || fcSttTimeoutSecondsEl.value || '';
  syncFCSTTRuntimeModeUI();
  if (fcSttRuntimeStateEl) fcSttRuntimeStateEl.textContent = JSON.stringify(rt, null, 2);
}

const fasterWhisperModelOptionsFC = [
  'tiny.en', 'tiny', 'base.en', 'base', 'small.en', 'small', 'medium.en', 'medium',
  'large-v1', 'large-v2', 'large-v3', 'large', 'distil-large-v2', 'distil-medium.en',
  'distil-small.en', 'distil-large-v3', 'distil-large-v3.5', 'large-v3-turbo', 'turbo'
];
const defaultFasterWhisperModelFC = 'small';

function normalizeFasterWhisperModelFC(value) {
  const model = String(value || '').trim();
  return fasterWhisperModelOptionsFC.includes(model) ? model : defaultFasterWhisperModelFC;
}

function currentSTTModelValueFC() {
  const mode = String(fcSttModeEl?.value || 'faster_whisper').trim().toLowerCase();
  if (mode === 'faster_whisper') {
    return normalizeFasterWhisperModelFC(fcSttFasterWhisperModelEl?.value || fcSttModelEl?.value || '');
  }
  return String(fcSttModelEl?.value || '').trim();
}

function syncFCSTTRuntimeModeUI() {
  const mode = String(fcSttModeEl?.value || 'faster_whisper').trim().toLowerCase();
  const isRemote = mode === 'remote';
  const isLMStudio = mode === 'lmstudio';
  const isFasterWhisper = mode === 'faster_whisper';
  const isLocal = mode === 'local';

  const toggle = (el, hidden) => {
    if (el) el.classList.toggle('hidden', !!hidden);
  };

  toggle(fcSttBaseURLWrapEl, !(isRemote || isLMStudio));
  toggle(fcSttModelWrapEl, !(isRemote || isLMStudio));
  toggle(fcSttFasterWhisperModelWrapEl, !isFasterWhisper);
  toggle(fcSttDeviceWrapEl, !isFasterWhisper);
  toggle(fcSttComputeTypeWrapEl, !isFasterWhisper);
  toggle(fcSttLanguageWrapEl, !isFasterWhisper);
  toggle(fcSttLocalCommandWrapEl, !isLocal);
  toggle(fcSttTimeoutWrapEl, !(isLMStudio || isFasterWhisper || isLocal));

  toggle(fcSttConnectionRowEl, !(isRemote || isLMStudio || isFasterWhisper));
  toggle(fcSttFasterWhisperRowEl, !isFasterWhisper);
  toggle(fcSttCommandRowEl, !(isLMStudio || isFasterWhisper || isLocal));

  if (fcSttModeHelpEl) {
    if (isRemote) {
      fcSttModeHelpEl.textContent = 'Remote OpenAI transcription uses a base URL and model.';
    } else if (isLMStudio) {
      fcSttModeHelpEl.textContent = 'LM Studio uses a local OpenAI-compatible endpoint, model, and optional timeout.';
    } else if (isFasterWhisper) {
      fcSttModeHelpEl.textContent = 'Managed faster-whisper runs locally inside Knit and uses model/device/compute settings.';
    } else {
      fcSttModeHelpEl.textContent = 'Local command mode shells out to your configured command with an optional timeout.';
    }
  }
  if (fcSttBaseURLEl) {
    fcSttBaseURLEl.placeholder = isLMStudio ? 'LM Studio base URL' : 'OpenAI base URL';
  }
  if (fcSttModelEl) {
    if (isRemote) {
      fcSttModelEl.placeholder = 'OpenAI STT model';
    } else if (isLMStudio) {
      fcSttModelEl.placeholder = 'LM Studio model';
    } else if (isFasterWhisper) {
      fcSttModelEl.placeholder = 'faster-whisper model';
    }
  }
  if (isFasterWhisper) {
    const normalized = normalizeFasterWhisperModelFC(fcSttFasterWhisperModelEl?.value || fcSttModelEl?.value || '');
    if (fcSttFasterWhisperModelEl) fcSttFasterWhisperModelEl.value = normalized;
    if (fcSttModelEl) fcSttModelEl.value = normalized;
  }
}

function setSelectOptionsFC(selectEl, values, currentValue, defaultLabel) {
  if (!selectEl) return;
  const seen = new Set();
  selectEl.innerHTML = '';
  const base = document.createElement('option');
  base.value = '';
  base.textContent = defaultLabel;
  selectEl.appendChild(base);
  values.forEach(v => {
    const value = String(v || '').trim();
    if (!value || seen.has(value)) return;
    seen.add(value);
    const opt = document.createElement('option');
    opt.value = value;
    opt.textContent = value;
    selectEl.appendChild(opt);
  });
  const curr = String(currentValue || '').trim();
  if (curr && !seen.has(curr)) {
    const custom = document.createElement('option');
    custom.value = curr;
    custom.textContent = curr + ' (current)';
    selectEl.appendChild(custom);
  }
  selectEl.value = curr;
}

function syncFCProviderOptions(data) {
  if (!fcAgentDefaultProviderEl) return;
  const adapters = Array.isArray(data?.adapters) ? data.adapters : [];
  if (!adapters.length) return;
  const preferred = String(data?.runtime_codex?.default_provider || '');
  const selected = String(fcAgentDefaultProviderEl.value || preferred || '');
  fcAgentDefaultProviderEl.innerHTML = '';
  adapters.forEach(name => {
    const value = String(name || '').trim();
    if (!value) return;
    const opt = document.createElement('option');
    opt.value = value;
    opt.textContent = value;
    fcAgentDefaultProviderEl.appendChild(opt);
  });
  if (selected) {
    fcAgentDefaultProviderEl.value = selected;
  }
  if (!fcAgentDefaultProviderEl.value && adapters.length) {
    fcAgentDefaultProviderEl.value = String(adapters[0]);
  }
  syncFCCodexRuntimeModeUI();
}

function deliveryPromptForProfileFC(runtimeCodex, profile) {
  const rc = runtimeCodex || {};
  switch (String(profile || '').trim()) {
    case 'draft_plan':
      return String(rc.draft_plan_prompt || '').trim() || defaultDeliveryIntentPromptFC('draft_plan');
    case 'create_jira_tickets':
      return String(rc.create_jira_tickets_prompt || '').trim() || defaultDeliveryIntentPromptFC('create_jira_tickets');
    default:
      return String(rc.implement_changes_prompt || '').trim() || defaultDeliveryIntentPromptFC('implement_changes');
  }
}

function syncDeliveryPromptUIFromStateFC(runtimeCodex, preservingRuntimeDraft = false) {
  if (!fcDeliveryIntentProfileEl || !fcDeliveryInstructionTextEl || preservingRuntimeDraft) return;
  const rc = runtimeCodex || {};
  const profile = String(rc.delivery_intent_profile || fcDeliveryIntentProfileEl.value || 'implement_changes').trim();
  fcDeliveryIntentProfileEl.value = profile === 'draft_plan' || profile === 'create_jira_tickets' ? profile : 'implement_changes';
  fcDeliveryInstructionTextEl.value = deliveryPromptForProfileFC(rc, fcDeliveryIntentProfileEl.value);
}

function currentDeliveryPromptPayloadFC() {
  const selected = selectedDeliveryIntentProfileFC();
  const rc = lastState?.runtime_codex || {};
  const selectedText = selectedDeliveryInstructionTextFC();
  return {
    delivery_intent_profile: selected,
    implement_changes_prompt: selected === 'implement_changes' ? selectedText : deliveryPromptForProfileFC(rc, 'implement_changes'),
    draft_plan_prompt: selected === 'draft_plan' ? selectedText : deliveryPromptForProfileFC(rc, 'draft_plan'),
    create_jira_tickets_prompt: selected === 'create_jira_tickets' ? selectedText : deliveryPromptForProfileFC(rc, 'create_jira_tickets'),
  };
}

function syncFCCodexRuntimeUI(data) {
  const rc = data?.runtime_codex || {};
  const preservingRuntimeDraft = fcCodexRuntimeDirty || fcCodexRuntimeApplying;
  if (fcCodexRuntimeStateEl) fcCodexRuntimeStateEl.textContent = JSON.stringify(rc, null, 2);
  if (fcAgentDefaultProviderEl && !preservingRuntimeDraft) fcAgentDefaultProviderEl.value = rc.default_provider || fcAgentDefaultProviderEl.value || 'codex_cli';
  if (fcCodexCliCmdEl && !preservingRuntimeDraft) fcCodexCliCmdEl.value = rc.cli_adapter_cmd || fcCodexCliCmdEl.value || '';
  if (fcClaudeCliCmdEl && !preservingRuntimeDraft) fcClaudeCliCmdEl.value = rc.claude_cli_adapter_cmd || fcClaudeCliCmdEl.value || '';
  if (fcOpenCodeCliCmdEl && !preservingRuntimeDraft) fcOpenCodeCliCmdEl.value = rc.opencode_cli_adapter_cmd || fcOpenCodeCliCmdEl.value || '';
  if (fcCodexOutputDirEl && !preservingRuntimeDraft) fcCodexOutputDirEl.value = rc.codex_output_dir || fcCodexOutputDirEl.value || '';
  if (fcCliTimeoutSecondsEl && !preservingRuntimeDraft) fcCliTimeoutSecondsEl.value = rc.cli_timeout_seconds || fcCliTimeoutSecondsEl.value || '600';
  if (fcClaudeCliTimeoutSecondsEl && !preservingRuntimeDraft) fcClaudeCliTimeoutSecondsEl.value = rc.claude_cli_timeout_seconds || fcClaudeCliTimeoutSecondsEl.value || '600';
  if (fcOpenCodeCliTimeoutSecondsEl && !preservingRuntimeDraft) fcOpenCodeCliTimeoutSecondsEl.value = rc.opencode_cli_timeout_seconds || fcOpenCodeCliTimeoutSecondsEl.value || '600';
  if (fcSubmitExecutionModeEl && !preservingRuntimeDraft) fcSubmitExecutionModeEl.value = rc.submit_execution_mode || fcSubmitExecutionModeEl.value || 'series';
  if (fcCodexSandboxEl && !preservingRuntimeDraft) fcCodexSandboxEl.value = rc.codex_sandbox || '';
  if (fcCodexApprovalEl && !preservingRuntimeDraft) fcCodexApprovalEl.value = rc.codex_approval_policy || '';
  if (fcCodexSkipRepoCheckEl && !preservingRuntimeDraft && rc.codex_skip_git_repo_check !== undefined && rc.codex_skip_git_repo_check !== null && String(rc.codex_skip_git_repo_check) !== '') {
    fcCodexSkipRepoCheckEl.checked = String(rc.codex_skip_git_repo_check) !== '0';
  }
  if (fcCodexProfileEl && !preservingRuntimeDraft) fcCodexProfileEl.value = rc.codex_profile || fcCodexProfileEl.value || '';
  if (fcCodexModelEl && !preservingRuntimeDraft) fcCodexModelEl.value = rc.codex_model || fcCodexModelEl.value || '';
  if (fcCodexReasoningEl && !preservingRuntimeDraft) fcCodexReasoningEl.value = rc.codex_reasoning_effort || fcCodexReasoningEl.value || '';
  if (fcCodexAPIBaseURLEl && !preservingRuntimeDraft) fcCodexAPIBaseURLEl.value = rc.openai_base_url || fcCodexAPIBaseURLEl.value || '';
  if (fcCodexAPITimeoutSecondsEl && !preservingRuntimeDraft) fcCodexAPITimeoutSecondsEl.value = rc.codex_api_timeout_seconds || fcCodexAPITimeoutSecondsEl.value || '60';
  if (fcCodexAPIOrgEl && !preservingRuntimeDraft) fcCodexAPIOrgEl.value = rc.openai_org_id || fcCodexAPIOrgEl.value || '';
  if (fcCodexAPIProjectEl && !preservingRuntimeDraft) fcCodexAPIProjectEl.value = rc.openai_project_id || fcCodexAPIProjectEl.value || '';
  if (fcClaudeAPIBaseURLEl && !preservingRuntimeDraft) fcClaudeAPIBaseURLEl.value = rc.anthropic_base_url || fcClaudeAPIBaseURLEl.value || '';
  if (fcClaudeAPITimeoutSecondsEl && !preservingRuntimeDraft) fcClaudeAPITimeoutSecondsEl.value = rc.claude_api_timeout_seconds || fcClaudeAPITimeoutSecondsEl.value || '60';
  if (fcClaudeAPIModelEl && !preservingRuntimeDraft) fcClaudeAPIModelEl.value = rc.claude_api_model || fcClaudeAPIModelEl.value || '';
  syncDeliveryPromptUIFromStateFC(rc, preservingRuntimeDraft);
  if (fcClaudeAPIKeyStatusEl) {
    fcClaudeAPIKeyStatusEl.textContent = rc.anthropic_api_key_configured ? 'ANTHROPIC_API_KEY detected for claude_api.' : 'Set ANTHROPIC_API_KEY in the environment before using claude_api.';
  }
  if (fcPostSubmitRebuildCmdEl && !preservingRuntimeDraft) fcPostSubmitRebuildCmdEl.value = rc.post_submit_rebuild_cmd || fcPostSubmitRebuildCmdEl.value || '';
  if (fcPostSubmitVerifyCmdEl && !preservingRuntimeDraft) fcPostSubmitVerifyCmdEl.value = rc.post_submit_verify_cmd || fcPostSubmitVerifyCmdEl.value || '';
  if (fcPostSubmitTimeoutSecEl && !preservingRuntimeDraft) fcPostSubmitTimeoutSecEl.value = rc.post_submit_timeout_seconds || fcPostSubmitTimeoutSecEl.value || '600';
  const workdir = rc.codex_workdir || fcWorkspaceDirEl?.value || '';
  if (fcCodexWorkdirLabelEl) fcCodexWorkdirLabelEl.textContent = workdir || '(not set)';
  syncFCCodexRuntimeModeUI();
  syncComposerSettingsSummaryFC();
}

async function refreshCodexOptionsFC() {
  try {
    fcCodexOptionsAttempted = true;
    setCodexRuntimeStatusFC('Loading Codex model/reasoning options...');
    const res = await fetch('/api/runtime/codex/options', { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    const models = Array.isArray(data.models) ? data.models : [];
    const modelValues = models.map(m => (m && m.model) ? String(m.model) : '').filter(Boolean);
    const reasoningValues = Array.isArray(data.reasoning_efforts) ? data.reasoning_efforts : [];
    const currentModel = (lastState?.runtime_codex?.codex_model || fcCodexModelEl?.value || data.default_model || '').trim();
    const currentReasoning = (lastState?.runtime_codex?.codex_reasoning_effort || fcCodexReasoningEl?.value || data.default_reasoning || '').trim();
    setSelectOptionsFC(fcCodexModelEl, modelValues, currentModel, 'Use Codex default model');
    setSelectOptionsFC(fcCodexReasoningEl, reasoningValues, currentReasoning, 'Use Codex default reasoning');
    setCodexRuntimeStatusFC('Loaded ' + modelValues.length + ' models from Codex CLI.');
    fcCodexOptionsLoaded = true;
  } catch (err) {
    setCodexRuntimeStatusFC('Codex options load failed: ' + err.message, true);
  }
}

async function applyCodexRuntimeFC() {
  if (fcCodexRuntimeApplyTimer) {
    clearTimeout(fcCodexRuntimeApplyTimer);
    fcCodexRuntimeApplyTimer = 0;
  }
  fcCodexRuntimeApplying = true;
  try {
    if (lastState?.config_locked) throw new Error('Config is locked by policy.');
    const timeoutSeconds = readRuntimeSecondsFC(fcCliTimeoutSecondsEl, 'Codex CLI timeout');
    const claudeTimeoutSeconds = readRuntimeSecondsFC(fcClaudeCliTimeoutSecondsEl, 'Claude CLI timeout');
    const opencodeTimeoutSeconds = readRuntimeSecondsFC(fcOpenCodeCliTimeoutSecondsEl, 'OpenCode CLI timeout');
    const codexAPITimeoutSeconds = readRuntimeSecondsFC(fcCodexAPITimeoutSecondsEl, 'Codex API timeout');
    const claudeAPITimeoutSeconds = readRuntimeSecondsFC(fcClaudeAPITimeoutSecondsEl, 'Claude API timeout');
    const postTimeoutSeconds = readRuntimeSecondsFC(fcPostSubmitTimeoutSecEl, 'Post-submit timeout', 7200);
    const deliveryPromptPayload = currentDeliveryPromptPayloadFC();
    const payload = {
      default_provider: selectedProviderFC(),
      cli_adapter_cmd: readRuntimeSingleLineFC(fcCodexCliCmdEl, 'Codex CLI command'),
      cli_timeout_seconds: timeoutSeconds,
      claude_cli_adapter_cmd: readRuntimeSingleLineFC(fcClaudeCliCmdEl, 'Claude CLI command'),
      claude_cli_timeout_seconds: claudeTimeoutSeconds,
      opencode_cli_adapter_cmd: readRuntimeSingleLineFC(fcOpenCodeCliCmdEl, 'OpenCode CLI command'),
      opencode_cli_timeout_seconds: opencodeTimeoutSeconds,
      submit_execution_mode: (fcSubmitExecutionModeEl?.value || 'series'),
      codex_workdir: readRuntimeSingleLineFC(fcWorkspaceDirEl, 'Workspace path'),
      codex_output_dir: readRuntimeSingleLineFC(fcCodexOutputDirEl, 'Output directory', 1024),
      codex_sandbox: (fcCodexSandboxEl?.value || ''),
      codex_approval_policy: (fcCodexApprovalEl?.value || ''),
      codex_profile: readRuntimeSingleLineFC(fcCodexProfileEl, 'Codex profile', 128),
      codex_model: readRuntimeSingleLineFC(fcCodexModelEl, 'Codex model', 128),
      codex_reasoning_effort: readRuntimeSingleLineFC(fcCodexReasoningEl, 'Codex reasoning effort', 64),
      openai_base_url: readRuntimeURLFC(fcCodexAPIBaseURLEl, 'Codex API base URL'),
      codex_api_timeout_seconds: codexAPITimeoutSeconds,
      openai_org_id: readRuntimeSingleLineFC(fcCodexAPIOrgEl, 'OpenAI org ID', 256),
      openai_project_id: readRuntimeSingleLineFC(fcCodexAPIProjectEl, 'OpenAI project ID', 256),
      anthropic_base_url: readRuntimeURLFC(fcClaudeAPIBaseURLEl, 'Claude API base URL'),
      claude_api_timeout_seconds: claudeAPITimeoutSeconds,
      claude_api_model: readRuntimeSingleLineFC(fcClaudeAPIModelEl, 'Claude API model', 128),
      delivery_intent_profile: deliveryPromptPayload.delivery_intent_profile,
      implement_changes_prompt: deliveryPromptPayload.implement_changes_prompt,
      draft_plan_prompt: deliveryPromptPayload.draft_plan_prompt,
      create_jira_tickets_prompt: deliveryPromptPayload.create_jira_tickets_prompt,
      post_submit_rebuild_cmd: readRuntimeSingleLineFC(fcPostSubmitRebuildCmdEl, 'Post-submit rebuild command'),
      post_submit_verify_cmd: readRuntimeSingleLineFC(fcPostSubmitVerifyCmdEl, 'Post-submit verify command'),
      post_submit_timeout_seconds: postTimeoutSeconds,
      codex_skip_git_repo_check: !!fcCodexSkipRepoCheckEl?.checked
    };
    const res = await postJSON('/api/runtime/codex', payload);
    if (fcCodexRuntimeStateEl) fcCodexRuntimeStateEl.textContent = JSON.stringify(res.runtime_codex || payload, null, 2);
    lastState = lastState || {};
    lastState.runtime_codex = res.runtime_codex || payload;
    fcCodexRuntimeDirty = false;
    syncFCCodexRuntimeUI({ runtime_codex: res.runtime_codex || payload });
    if (fcCodexWorkdirLabelEl) fcCodexWorkdirLabelEl.textContent = payload.codex_workdir || '(not set)';
    setFCSetting('default_provider', payload.default_provider || '');
    setFCSetting('cli_adapter_cmd', payload.cli_adapter_cmd || '');
    setFCSetting('claude_cli_adapter_cmd', payload.claude_cli_adapter_cmd || '');
    setFCSetting('opencode_cli_adapter_cmd', payload.opencode_cli_adapter_cmd || '');
    setFCSetting('cli_timeout_seconds', Number.isFinite(timeoutSeconds) && timeoutSeconds > 0 ? String(timeoutSeconds) : '');
    setFCSetting('claude_cli_timeout_seconds', Number.isFinite(claudeTimeoutSeconds) && claudeTimeoutSeconds > 0 ? String(claudeTimeoutSeconds) : '');
    setFCSetting('opencode_cli_timeout_seconds', Number.isFinite(opencodeTimeoutSeconds) && opencodeTimeoutSeconds > 0 ? String(opencodeTimeoutSeconds) : '');
    setFCSetting('submit_execution_mode', payload.submit_execution_mode || 'series');
    setFCSetting('codex_output_dir', payload.codex_output_dir || '');
    setFCSetting('codex_sandbox', payload.codex_sandbox || '');
    setFCSetting('codex_approval_policy', payload.codex_approval_policy || '');
    setFCSetting('codex_skip_git_repo_check', !!payload.codex_skip_git_repo_check);
    setFCSetting('codex_profile', payload.codex_profile || '');
    setFCSetting('codex_model', payload.codex_model || '');
    setFCSetting('codex_reasoning_effort', payload.codex_reasoning_effort || '');
    setFCSetting('openai_base_url', payload.openai_base_url || '');
    setFCSetting('codex_api_timeout_seconds', Number.isFinite(codexAPITimeoutSeconds) && codexAPITimeoutSeconds > 0 ? String(codexAPITimeoutSeconds) : '');
    setFCSetting('openai_org_id', payload.openai_org_id || '');
    setFCSetting('openai_project_id', payload.openai_project_id || '');
    setFCSetting('anthropic_base_url', payload.anthropic_base_url || '');
    setFCSetting('claude_api_timeout_seconds', Number.isFinite(claudeAPITimeoutSeconds) && claudeAPITimeoutSeconds > 0 ? String(claudeAPITimeoutSeconds) : '');
    setFCSetting('claude_api_model', payload.claude_api_model || '');
    setFCSetting('delivery_intent_profile', payload.delivery_intent_profile || 'implement_changes');
    setFCSetting('delivery_instruction_text', selectedDeliveryInstructionTextFC());
    setFCSetting('post_submit_rebuild_cmd', payload.post_submit_rebuild_cmd || '');
    setFCSetting('post_submit_verify_cmd', payload.post_submit_verify_cmd || '');
    setFCSetting('post_submit_timeout_seconds', Number.isFinite(postTimeoutSeconds) && postTimeoutSeconds > 0 ? String(postTimeoutSeconds) : '');
    setCodexRuntimeStatusFC('Runtime settings saved automatically.');
    syncFCCodexRuntimeModeUI();
    setStatus('Agent runtime settings updated.');
  } catch (err) {
    fcCodexRuntimeDirty = false;
    setCodexRuntimeStatusFC('Runtime settings could not be saved: ' + err.message, true);
    setStatus('Runtime update failed: ' + err.message, true);
  } finally {
    fcCodexRuntimeApplying = false;
  }
}

async function applyTranscriptionRuntimeFC() {
  if (fcSTTRuntimeApplyTimer) {
    clearTimeout(fcSTTRuntimeApplyTimer);
    fcSTTRuntimeApplyTimer = 0;
  }
  fcSTTRuntimeApplying = true;
  try {
    if (lastState?.config_locked) throw new Error('Config is locked by policy.');
    const mode = (fcSttModeEl?.value || '').trim();
    const timeoutSeconds = Number.parseInt(fcSttTimeoutSecondsEl?.value || '0', 10);
    const payload = {
      mode,
      base_url: (fcSttBaseURLEl?.value || '').trim(),
      model: currentSTTModelValueFC(),
      device: (fcSttDeviceEl?.value || '').trim(),
      compute_type: (fcSttComputeTypeEl?.value || '').trim(),
      language: (fcSttLanguageEl?.value || '').trim(),
      local_command: (fcSttLocalCommandEl?.value || '').trim(),
      timeout_seconds: Number.isFinite(timeoutSeconds) && timeoutSeconds > 0 ? timeoutSeconds : 0
    };
    await postJSON('/api/runtime/transcription', payload);
    fcSTTRuntimeDirty = false;
    await refreshState();
    setFCSetting('stt_mode', mode);
    setFCSetting('stt_base_url', payload.base_url);
    setFCSetting('stt_model', payload.model);
    setFCSetting('stt_device', payload.device);
    setFCSetting('stt_compute_type', payload.compute_type);
    setFCSetting('stt_language', payload.language);
    setFCSetting('stt_local_command', payload.local_command);
    setFCSetting('stt_timeout_seconds', Number.isFinite(timeoutSeconds) && timeoutSeconds > 0 ? String(timeoutSeconds) : '');
    if (fcSttHealthStateEl) {
      fcSttHealthStateEl.textContent = 'Transcription settings saved.';
      fcSttHealthStateEl.style.color = '#1f8f63';
    }
    setStatus('Transcription runtime updated: ' + mode);
  } catch (err) {
    fcSTTRuntimeDirty = false;
    if (fcSttHealthStateEl) {
      fcSttHealthStateEl.textContent = 'Transcription settings could not be saved: ' + err.message;
      fcSttHealthStateEl.style.color = '#c34f4f';
    }
    setStatus('Transcription runtime update failed: ' + err.message, true);
  } finally {
    fcSTTRuntimeApplying = false;
  }
}

function scheduleTranscriptionRuntimeApplyFC() {
  fcSTTRuntimeDirty = true;
  syncFCSTTRuntimeModeUI();
  if (fcSttHealthStateEl) {
    fcSttHealthStateEl.textContent = 'Saving transcription settings...';
    fcSttHealthStateEl.style.color = '#1c7c74';
  }
  if (fcSTTRuntimeApplyTimer) {
    clearTimeout(fcSTTRuntimeApplyTimer);
  }
  fcSTTRuntimeApplyTimer = window.setTimeout(() => {
    applyTranscriptionRuntimeFC();
  }, 350);
}

async function checkTranscriptionHealthFC() {
  try {
    if (fcSttHealthStateEl) {
      fcSttHealthStateEl.textContent = 'Checking transcription connection...';
      fcSttHealthStateEl.style.color = '#1c7c74';
    }
    const res = await fetch('/api/runtime/transcription/health', { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    const status = String(data.status || 'unknown');
    const msg = String(data.message || '');
    if (fcSttHealthStateEl) {
      fcSttHealthStateEl.textContent = 'Transcription connection: ' + status + (msg ? (' - ' + msg) : '');
      fcSttHealthStateEl.style.color = status === 'ok' ? '#1f8f63' : '#c34f4f';
    }
    if (data.runtime_transcription) {
      lastState = lastState || {};
      lastState.runtime_transcription = data.runtime_transcription;
      lastState.transcription_mode = data.runtime_transcription.mode || lastState.transcription_mode;
      syncFCTranscriptionUI(lastState);
    }
  } catch (err) {
    if (fcSttHealthStateEl) {
      fcSttHealthStateEl.textContent = 'Connection check failed: ' + err.message;
      fcSttHealthStateEl.style.color = '#c34f4f';
    }
    setStatus('Transcription health check failed: ' + err.message, true);
  }
}

function scheduleAudioConfigApplyFC() {
  fcAudioDirty = true;
  if (fcAudioApplyTimer) {
    clearTimeout(fcAudioApplyTimer);
  }
  fcAudioApplyTimer = window.setTimeout(() => {
    applyAudioConfigFC();
  }, 300);
}

async function applyAudioConfigFC() {
  if (fcAudioApplyTimer) {
    clearTimeout(fcAudioApplyTimer);
    fcAudioApplyTimer = 0;
  }
  fcAudioApplying = true;
  try {
    const payload = {
      mode: fcAudioModeEl?.value || 'always_on',
      input_device_id: fcAudioInputDeviceEl?.value || 'default',
      muted: !!fcAudioMutedEl?.checked,
      paused: !!fcAudioPausedEl?.checked,
    };
    await postJSON('/api/audio/config', payload);
    fcAudioDirty = false;
    setFCSetting('audio_mode', payload.mode);
    await refreshState();
  } catch (err) {
    fcAudioDirty = false;
    setStatus('Audio configuration failed: ' + err.message, true);
  } finally {
    fcAudioApplying = false;
  }
}

async function refreshAudioDevicesFC() {
  try {
    let devicePayload = [];
    if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
      const devices = await navigator.mediaDevices.enumerateDevices();
      devicePayload = devices
        .filter(d => d.kind === 'audioinput')
        .map(d => ({ id: d.deviceId || 'default', label: d.label || 'Audio Input' }));
    }
    await postJSON('/api/audio/devices', { devices: devicePayload });
    await refreshState();
    setStatus('Audio devices refreshed.');
  } catch (err) {
    setStatus('Audio device refresh failed: ' + err.message, true);
  }
}

async function testMicrophoneFC(seconds = 10) {
  if (fcMicTestRunning) {
    setStatus('Mic test already in progress.');
    return;
  }
  let stream = null;
  let ctx = null;
  let interval = 0;
  try {
    fcMicTestRunning = true;
    setAudioLevelStateVisibleFC(true);
    if (fcTestMicBtnEl) fcTestMicBtnEl.disabled = true;
    if (fcMicTestMeterFillEl) fcMicTestMeterFillEl.style.width = '0%';
    if (fcMicTestStateEl) {
      fcMicTestStateEl.textContent = 'Starting mic test...';
      fcMicTestStateEl.style.color = '#4fd1c5';
    }
    const deviceId = (fcAudioInputDeviceEl?.value || '').trim();
    stream = await navigator.mediaDevices.getUserMedia({
      audio: deviceId ? { deviceId: { exact: deviceId } } : true,
      video: false
    });
    const Ctx = window.AudioContext || window.webkitAudioContext;
    if (!Ctx) throw new Error('WebAudio is unavailable in this browser.');
    ctx = new Ctx();
    const source = ctx.createMediaStreamSource(stream);
    const analyser = ctx.createAnalyser();
    analyser.fftSize = 1024;
    source.connect(analyser);
    const data = new Uint8Array(analyser.fftSize);
    let peak = 0;
    let sum = 0;
    let samples = 0;
    const startedAt = Date.now();
    const durationMs = Math.max(1000, Number(seconds || 10) * 1000);

    await new Promise((resolve) => {
      interval = window.setInterval(() => {
        analyser.getByteTimeDomainData(data);
        let sq = 0;
        for (let i = 0; i < data.length; i++) {
          const centered = (data[i] - 128) / 128;
          sq += centered * centered;
        }
        const rms = Math.sqrt(sq / data.length);
        peak = Math.max(peak, rms);
        sum += rms;
        samples += 1;
        const elapsed = Date.now() - startedAt;
        const leftSeconds = Math.max(0, Math.ceil((durationMs - elapsed) / 1000));
        const db = 20 * Math.log10(Math.max(rms, 0.0001));
        const normalized = Math.max(0, Math.min(1, (db + 60) / 60));
        if (fcMicTestMeterFillEl) fcMicTestMeterFillEl.style.width = Math.round(normalized * 100) + '%';
        if (fcMicTestStateEl) {
          fcMicTestStateEl.textContent = 'Testing microphone... ' + leftSeconds + 's left | level=' + rms.toFixed(3);
          fcMicTestStateEl.style.color = '#4fd1c5';
        }
        if (elapsed >= durationMs) {
          window.clearInterval(interval);
          resolve();
        }
      }, 100);
    });

    const avg = samples > 0 ? (sum / samples) : 0;
    await postJSON('/api/audio/level', { level: peak });
    await refreshState();
    const min = Number(lastState?.audio?.state?.level_min || 0.02);
    const max = Number(lastState?.audio?.state?.level_max || 0.95);
    const valid = peak >= min && peak <= max;
    if (fcMicTestStateEl) {
      fcMicTestStateEl.textContent = 'Mic test complete. peak=' + peak.toFixed(3) + ' avg=' + avg.toFixed(3) + ' valid=' + valid;
      fcMicTestStateEl.style.color = valid ? '#48bb78' : '#f6ad55';
    }
    setStatus('Mic test complete. Peak level: ' + peak.toFixed(3) + ' (valid=' + valid + ').');
  } catch (err) {
    if (fcMicTestStateEl) {
      fcMicTestStateEl.textContent = 'Mic test failed: ' + err.message;
      fcMicTestStateEl.style.color = '#f56565';
    }
    setStatus('Mic test failed: ' + err.message, true);
  } finally {
    if (interval) window.clearInterval(interval);
    if (ctx) {
      try {
        await ctx.close();
      } catch (_) {}
    }
    if (stream) stream.getTracks().forEach(t => t.stop());
    fcMicTestRunning = false;
    setAudioLevelStateVisibleFC(false);
    if (fcTestMicBtnEl) fcTestMicBtnEl.disabled = false;
  }
}

function renderScreenshotQueueState() {
  if (!screenshotLineEl) return;
  if (!queuedScreenshotBlob) {
    setStatusChipFC(screenshotLineEl, 'Snapshot', 'None queued', '');
    if (fcVideoCaptureStateEl) {
      fcVideoCaptureStateEl.textContent = 'No snapshot queued.';
      fcVideoCaptureStateEl.style.color = '#6a7383';
    }
  } else {
    const kb = Math.max(1, Math.round(queuedScreenshotBlob.size / 1024));
    setStatusChipFC(screenshotLineEl, 'Snapshot', kb + ' KB queued', 'ok');
    if (fcVideoCaptureStateEl) {
      fcVideoCaptureStateEl.textContent = 'Snapshot queued (' + kb + ' KB).';
      fcVideoCaptureStateEl.style.color = '#1f8f63';
    }
  }
  setButtonsDisabled(inFlight);
}

function looksLikeLocalAttemptLogRefFC(ref) {
  const value = String(ref || '').trim();
  if (!value) return false;
  const isAbsolute = value.startsWith('/') || /^[A-Za-z]:[\\/]/.test(value);
  if (!isAbsolute) return false;
  return /(?:^|[\\/])knit-codex-[^\\/]*\.log[^\\/]*$/i.test(value);
}

function attemptLogRefFC(attempt) {
  if (!attempt) return '';
  return String(attempt.execution_ref || attempt.ref || '');
}

function trimLiveOutputBufferFC(text, maxChars = 80000) {
  const value = String(text || '').replace(/\r\n/g, '\n');
  if (value.length <= maxChars) return value;
  const sliced = value.slice(-maxChars);
  const newline = sliced.indexOf('\n');
  return newline >= 0 ? sliced.slice(newline + 1) : sliced;
}

function trimLiveOutputSectionFC(text, maxChars = 32000) {
  const value = String(text || '').replace(/\r\n/g, '\n').trim();
  if (!value) return '';
  if (value.length <= maxChars) return value;
  const sliced = value.slice(-maxChars);
  const newline = sliced.indexOf('\n');
  const body = newline >= 0 ? sliced.slice(newline + 1) : sliced;
  return '…\n' + body;
}

function isLiveOutputWorkMarkerFC(line) {
  const trimmed = String(line || '').trim();
  if (!trimmed) return false;
  if (/^\[\d{4}-\d{2}-\d{2}T/.test(trimmed)) return true;
  if (trimmed === 'exec' || trimmed.startsWith('exec ')) return true;
  if (trimmed.startsWith('mcp:')) return true;
  if (trimmed === 'apply_patch' || trimmed.startsWith('apply_patch ')) return true;
  if (trimmed === 'OpenAI Codex v0.114.0 (research preview)' || trimmed.startsWith('OpenAI Codex v')) return true;
  if (trimmed === '--------') return true;
  if (/^(workdir|model|provider|approval|sandbox|reasoning effort|reasoning summaries|session id):/i.test(trimmed)) return true;
  if (trimmed.startsWith('/bin/') || trimmed.startsWith('bash ') || trimmed.startsWith('sh ') || trimmed.startsWith('git ') || trimmed.startsWith('rg ') || trimmed.startsWith('sed ') || trimmed.startsWith('cat ') || trimmed.startsWith('go ') || trimmed.startsWith('npm ') || trimmed.startsWith('pnpm ')) return true;
  if (/^[0-9]{3,}:/.test(trimmed)) return true;
  if (trimmed.includes(' succeeded in ') || trimmed.includes(' failed in ')) return true;
  return false;
}

function isLikelyLivePayloadLineFC(line) {
  const text = String(line || '');
  const trimmed = text.trim();
  if (!trimmed) return false;
  if (trimmed.startsWith('{"created":') || trimmed.includes('"inline_data_url"')) return true;
  if (trimmed.includes('data:image/') || trimmed.includes('data:video/') || trimmed.includes('data:audio/')) return true;
  if (text.length > 4000) return true;
  return /^[A-Za-z0-9+/]{256,}={0,2}$/.test(trimmed);
}

function isLikelyLiveCommentaryLineFC(line) {
  const trimmed = String(line || '').trim();
  if (!trimmed) return false;
  return /^(I('|’)m|I am|I('|’)ll|I will|I have|I found|I can|Next I|The current |This is |That means|I’m|I’ll)/.test(trimmed);
}

function splitLiveAgentOutputForDisplayFC(raw) {
  const work = [];
  const commentary = [];
  const lines = String(raw || '').replace(/\r\n/g, '\n').split('\n');
  let mode = 'work';
  let omittingPrompt = false;
  let insertedPromptNotice = false;
  let insertedPayloadNotice = false;

  const pushPromptNotice = () => {
    if (!insertedPromptNotice) {
      work.push('[request context omitted from live view; use Open log for the full prompt and payload]');
      insertedPromptNotice = true;
    }
  };
  const pushPayloadNotice = () => {
    if (!insertedPayloadNotice) {
      work.push('[large inline payload omitted from live view]');
      insertedPayloadNotice = true;
    }
  };

  for (const line of lines) {
    const trimmed = String(line || '').trim();
    if (trimmed === 'user') {
      omittingPrompt = true;
      mode = 'work';
      continue;
    }
    if (omittingPrompt) {
      if (trimmed === 'codex' || isLiveOutputWorkMarkerFC(trimmed)) {
        omittingPrompt = false;
        pushPromptNotice();
      } else {
        if (trimmed === 'Canonical payload JSON:' || isLikelyLivePayloadLineFC(line)) {
          pushPayloadNotice();
        }
        continue;
      }
    }
    if (trimmed === 'codex') {
      mode = 'commentary';
      continue;
    }
    if (isLikelyLivePayloadLineFC(line)) {
      pushPayloadNotice();
      continue;
    }
    if (isLiveOutputWorkMarkerFC(trimmed)) {
      mode = 'work';
      if (trimmed === 'exec') continue;
      work.push(line);
      continue;
    }
    if (mode === 'commentary' || isLikelyLiveCommentaryLineFC(line)) {
      commentary.push(line);
      continue;
    }
    work.push(line);
  }
  if (omittingPrompt) pushPromptNotice();
  return {
    work: trimLiveOutputSectionFC(work.join('\n')),
    commentary: trimLiveOutputSectionFC(commentary.join('\n')),
  };
}

function renderLiveAgentOutputFC() {
  if (!fcLiveSubmitLogEl) return;
  const split = splitLiveAgentOutputForDisplayFC(fcLiveLogRawText);
  fcLiveSubmitLogEl.textContent = split.work || 'No live work log yet. Work activity appears here after the adapter starts writing logs.';
  fcLiveSubmitLogEl.scrollTop = fcLiveSubmitLogEl.scrollHeight;
  if (fcLiveSubmitCommentaryEl) {
    fcLiveSubmitCommentaryEl.textContent = split.commentary || 'No agent commentary yet. Plain-language progress updates appear here when the agent explains what it is doing.';
    fcLiveSubmitCommentaryEl.scrollTop = fcLiveSubmitCommentaryEl.scrollHeight;
  }
}

function providerDestinationLabelFC(provider) {
  const value = String(provider || '').trim();
  switch (value) {
    case 'codex_api':
    case 'claude_api':
      return 'Sent to remote provider';
    case 'codex_cli':
    case 'claude_cli':
    case 'opencode_cli':
    case 'cli':
      return 'Sent to local CLI on this machine';
    default:
      return 'Stays on this machine';
  }
}

function findSubmitAttemptByIdFC(attemptID) {
  const id = String(attemptID || '').trim();
  if (!id) return null;
  const attempts = Array.isArray(lastState?.submit_attempts) ? lastState.submit_attempts : [];
  return attempts.find(a => String(a?.attempt_id || '') === id) || null;
}

function activeRunningSubmitAttemptFC() {
  const attempts = Array.isArray(lastState?.submit_attempts) ? lastState.submit_attempts : [];
  return attempts.find(a => String(a?.status || '') === 'in_progress') || null;
}

function activeSubmitAttemptForLogFC() {
  const runningAttempt = activeRunningSubmitAttemptFC();
  if (runningAttempt && looksLikeLocalAttemptLogRefFC(attemptLogRefFC(runningAttempt))) return runningAttempt;
  if (fcLiveLogAttemptId && fcLiveLogCompletedForAttempt !== fcLiveLogAttemptId) {
    const latestAttempt = findSubmitAttemptByIdFC(fcLiveLogAttemptId);
    if (latestAttempt && looksLikeLocalAttemptLogRefFC(attemptLogRefFC(latestAttempt))) return latestAttempt;
  }
  const attempts = Array.isArray(lastState?.submit_attempts) ? lastState.submit_attempts : [];
  return attempts.find(a => {
    if (!a) return false;
    const attemptId = String(a.attempt_id || '').trim();
    if (!attemptId || attemptId === fcLiveLogCompletedForAttempt) return false;
    if (String(a.status || '') === 'queued') return false;
    return looksLikeLocalAttemptLogRefFC(attemptLogRefFC(a));
  }) || null;
}

function hasOpenableSubmitLogFC() {
  return !!activeSubmitAttemptForLogFC();
}

async function refreshActiveSubmitLogFC() {
  if (!fcLiveSubmitLogEl) return;
  const logAttempt = activeSubmitAttemptForLogFC();
  if (!logAttempt) return;

  const attemptId = String(logAttempt.attempt_id || '');
  if (!attemptId) return;
  const executionRef = attemptLogRefFC(logAttempt);
  if (!executionRef) {
    if (fcLiveLogAttemptId !== attemptId) {
      fcLiveLogAttemptId = attemptId;
      fcLiveLogOffset = 0;
      fcLiveLogRawText = '[' + attemptId + '] waiting for adapter log path...';
      fcLiveLogCompletedForAttempt = '';
      fcLiveLogUnavailableForAttempt = '';
      renderLiveAgentOutputFC();
    }
    return;
  }

  if (!looksLikeLocalAttemptLogRefFC(executionRef)) {
    if (fcLiveLogUnavailableForAttempt !== attemptId) {
      fcLiveLogAttemptId = attemptId;
      fcLiveLogOffset = 0;
      fcLiveLogRawText = '[' + attemptId + '] live log unavailable for this adapter.';
      fcLiveLogCompletedForAttempt = attemptId;
      fcLiveLogUnavailableForAttempt = attemptId;
      renderLiveAgentOutputFC();
    }
    return;
  }

  if (fcLiveLogAttemptId !== attemptId) {
    fcLiveLogAttemptId = attemptId;
    fcLiveLogOffset = 0;
    fcLiveLogRawText = '[' + attemptId + '] streaming adapter output...\n';
    fcLiveLogCompletedForAttempt = '';
    fcLiveLogUnavailableForAttempt = '';
    renderLiveAgentOutputFC();
  }

  try {
    const path = '/api/session/attempt/log?attempt_id=' + encodeURIComponent(attemptId) +
      '&offset=' + encodeURIComponent(String(fcLiveLogOffset)) +
      '&limit=12000';
    const res = await fetch(path, { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    const chunk = String(data.chunk || '');
    if (chunk) {
      fcLiveLogRawText = trimLiveOutputBufferFC(fcLiveLogRawText + chunk);
      renderLiveAgentOutputFC();
    }
    const nextOffset = Number(data.next_offset || fcLiveLogOffset);
    if (Number.isFinite(nextOffset) && nextOffset >= 0) {
      fcLiveLogOffset = nextOffset;
    }
    const status = String(data.status || '');
    if (status !== 'in_progress' && data.eof && fcLiveLogCompletedForAttempt !== attemptId) {
      fcLiveLogCompletedForAttempt = attemptId;
      fcLiveLogRawText = trimLiveOutputBufferFC(fcLiveLogRawText + '\n[' + attemptId + '] stream complete.\n');
      renderLiveAgentOutputFC();
    }
  } catch (err) {
    const message = String(err?.message || '');
    if (message.includes('submission reference is not') || message.includes('last submission reference is not')) {
      fcLiveLogCompletedForAttempt = attemptId;
      fcLiveLogUnavailableForAttempt = attemptId;
      fcLiveLogRawText = '[' + attemptId + '] live log unavailable for this adapter.';
      renderLiveAgentOutputFC();
      return;
    }
    fcLiveLogRawText = trimLiveOutputBufferFC(fcLiveLogRawText + '\n[' + attemptId + '] stream error: ' + err.message + '\n');
    renderLiveAgentOutputFC();
  }
}

function submitAttemptOutputTextFC(attempt) {
  if (!attempt || typeof attempt !== 'object') return '';
  const attemptID = String(attempt.attempt_id || '').trim();
  if (attemptID) {
    const cached = fcSubmitAttemptOutputPreviewByID.get(attemptID);
    if (cached?.status === 'ready') return String(cached.text || '').trim();
    if (cached?.status === 'empty') return 'Adapter log is empty.';
    if (cached?.status === 'loading') return 'Loading adapter output...';
  }
  const error = String(attempt.error || '').trim();
  if (error) return 'Error: ' + error;
  const note = String(attempt.note || '').trim();
  if (note) return note;
  if (looksLikeLocalAttemptLogRefFC(attemptLogRefFC(attempt))) return 'Loading adapter output...';
  const ref = String(attempt.ref || '').trim();
  if (ref) return 'Result received.';
  return '';
}

function submitAttemptWorkspaceTextFC(attempt) {
  if (!attempt || typeof attempt !== 'object') return '';
  return String(attempt.workdir_used || '').trim();
}

function submitAttemptOutputHasPreviewFC(attempt) {
  if (!attempt || typeof attempt !== 'object') return false;
  const attemptID = String(attempt.attempt_id || '').trim();
  if (!attemptID) return false;
  const cached = fcSubmitAttemptOutputPreviewByID.get(attemptID);
  return cached?.status === 'ready' || cached?.status === 'empty' || cached?.status === 'loading';
}

function normalizeSubmitAttemptOutputPreviewFC(text, truncated, truncatedHead) {
  const compact = String(text || '').replace(/\r\n/g, '\n').trim();
  if (!compact) return '';
  let preview = compact;
  if (truncatedHead) preview = '…\n' + preview;
  if (truncated) preview = preview + '\n…';
  return preview;
}

async function hydrateSubmitAttemptOutputsFC() {
  const attempts = Array.isArray(lastState?.submit_attempts) ? lastState.submit_attempts : [];
  const visible = attempts.slice(0, 5);
  for (const attempt of visible) {
    if (!attempt || !looksLikeLocalAttemptLogRefFC(attemptLogRefFC(attempt))) continue;
    const attemptID = String(attempt.attempt_id || '').trim();
    if (!attemptID) continue;
    const cached = fcSubmitAttemptOutputPreviewByID.get(attemptID);
    if (cached?.status === 'ready' || cached?.status === 'empty') continue;
    if (fcSubmitAttemptOutputPreviewInflight.has(attemptID)) continue;
    fcSubmitAttemptOutputPreviewInflight.add(attemptID);
    fcSubmitAttemptOutputPreviewByID.set(attemptID, { status: 'loading', text: '' });
    try {
      const status = String(attempt?.status || '').trim();
      const useTail = status !== 'in_progress' && status !== 'queued';
      const path = '/api/session/attempt/log?attempt_id=' + encodeURIComponent(attemptID) + '&offset=0&limit=24000' + (useTail ? '&tail=1' : '');
      const res = await fetch(path, { headers: authHeaders(false) });
      const txt = await res.text();
      if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
      const data = txt ? JSON.parse(txt) : {};
      const preview = normalizeSubmitAttemptOutputPreviewFC(String(data.chunk || ''), !data.eof, !!data.truncated_head);
      fcSubmitAttemptOutputPreviewByID.set(attemptID, { status: preview ? 'ready' : 'empty', text: preview });
    } catch (_) {
      fcSubmitAttemptOutputPreviewByID.delete(attemptID);
    } finally {
      fcSubmitAttemptOutputPreviewInflight.delete(attemptID);
    }
  }
  renderSubmitHistoryFC();
}

function renderSubmitAttemptOutputFC(attempt) {
  const output = submitAttemptOutputTextFC(attempt);
  if (!output) {
    return '<div class="small" style="color:#6a7383;">No output captured yet.</div>';
  }
  if (!submitAttemptOutputHasPreviewFC(attempt)) {
    return '<div class="small">' + escapePreviewHTML(output) + '</div>';
  }
  return '<pre style="margin-top:.35rem;max-height:120px;overflow:auto;white-space:pre-wrap;">' + escapePreviewHTML(output) + '</pre>';
}

function renderSubmitHistoryFC() {
  if (!fcSubmitHistoryEl) return;
  const attempts = Array.isArray(lastState?.submit_attempts) ? lastState.submit_attempts : [];
  if (!attempts.length) {
    fcSubmitHistoryEl.textContent = 'No runs yet.';
    return;
  }
  fcSubmitHistoryEl.innerHTML = attempts.slice(0, 5).map(attempt => {
    const status = String(attempt?.status || 'unknown');
    const request = escapePreviewHTML(String(attempt?.request_preview || 'No request preview captured.'));
    const attemptID = escapePreviewHTML(String(attempt?.attempt_id || 'attempt'));
    const destination = escapePreviewHTML(providerDestinationLabelFC(attempt?.provider || 'agent'));
    return '<div class="status-card" style="margin-top:.45rem;">' +
      '<div class="row" style="justify-content:space-between;align-items:flex-start;gap:.6rem;">' +
        '<strong>' + destination + '</strong>' +
        '<span class="small">' + escapePreviewHTML(status) + '</span>' +
      '</div>' +
      '<div class="small" style="margin-top:.3rem;"><strong>Request:</strong> ' + request + '</div>' +
      (submitAttemptWorkspaceTextFC(attempt) ? '<div class="small" style="margin-top:.3rem;"><strong>Workspace used:</strong> ' + escapePreviewHTML(submitAttemptWorkspaceTextFC(attempt)) + '</div>' : '') +
      '<div class="small" style="margin-top:.3rem;"><strong>Output:</strong></div>' +
      renderSubmitAttemptOutputFC(attempt) +
      '<div class="small" style="margin-top:.3rem;color:#6a7383;">' + attemptID + '</div>' +
    '</div>';
  }).join('');
}

function notifySubmitRecoveryNoticesFC(notices) {
  const list = Array.isArray(notices) ? notices : [];
  list.forEach(note => {
    const message = String(note || '').trim();
    if (!message || fcSeenSubmitRecoveryNotices.has(message)) return;
    fcSeenSubmitRecoveryNotices.add(message);
    setStatus(message, true);
    showToastFC(message, true);
  });
}

async function refreshState() {
  try {
    const res = await fetch('/api/state', { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    lastState = data;
    syncReplayValueCaptureUIFC();
    syncFCProviderOptions(data);
    const rc = data.runtime_codex || {};
    const selectedWorkspace = String(rc.codex_workdir || fcWorkspaceDirEl?.value || '').trim();
    const configLocked = !!data.config_locked;
    if (fcWorkspaceDirEl) {
      fcWorkspaceDirEl.value = rc.codex_workdir || fcWorkspaceDirEl.value || '';
    }
    syncFCCodexRuntimeUI(data);
    updateWorkspaceModalStateFC();
    const sess = data.session || null;
    const sessionID = sess && sess.id ? sess.id : 'none';
    const capture = data.capture_state || 'unknown';
    const approved = !!(sess && sess.approved);
    const feedback = Array.isArray(sess && sess.feedback) ? sess.feedback : [];
    const feedbackCount = feedback.length;
    const queue = data?.submit_queue || {};
    const queueMode = String(queue.mode || 'series');
    const queueRunning = Number(queue.running || 0);
    const queueQueued = Number(queue.queued || 0);
    const postSubmitRunning = !!queue.post_submit_running;
    const sessionLabel = sessionID === 'none' ? 'No session' : ('S ' + String(sessionID).slice(0, 8));
    setStatusChipFC(stateLineEl, 'Session', sessionLabel + ' • ' + feedbackCount + ' notes • ' + capture + (approved ? ' • ready' : ''), sessionID === 'none' ? 'warn' : '');
    stateLineEl.title = 'Session ' + sessionID + ' | capture ' + capture + ' | notes ' + feedbackCount + (approved ? ' | ready to send' : '');
    if (queueLineEl) {
      setStatusChipFC(queueLineEl, 'Queue', queueMode + ' • ' + queueRunning + ' running • ' + queueQueued + ' waiting' + (postSubmitRunning ? ' • rebuild/check running' : ''), (queueRunning > 0 || queueQueued > 0) ? 'ok' : '');
      queueLineEl.title = 'Queue: ' + queueRunning + ' running | ' + queueQueued + ' waiting | mode ' + queueMode + (postSubmitRunning ? ' | rebuild/check running' : '');
    }
    notifySubmitAttemptTransitionsFC(Array.isArray(data?.submit_attempts) ? data.submit_attempts : []);
    notifySubmitRecoveryNoticesFC(data?.submit_recovery_notices);
    renderSubmitHistoryFC();
    void hydrateSubmitAttemptOutputsFC();
    const hasRenderedPreview = !!(fcPayloadPreviewEl && !String(fcPayloadPreviewEl.textContent || '').includes('Preview the request here before sending it to the agent.'));
    if (!feedbackCount) {
      setPreviewState(sessionID === 'none' ? 'Start a session, capture a note, then preview it here.' : 'Capture at least one note, then preview it here.');
      if (fcPayloadPreviewEl) {
        fcPayloadPreviewEl.textContent = 'Preview the request here before sending it to the agent.';
      }
    } else if (approved && !hasRenderedPreview) {
      setPreviewState('Your latest notes are prepared. Preview them here or send them now.');
    }
    syncFCAudioUI(data);
    syncFCTranscriptionUI(data);
    syncComposerSettingsSummaryFC();
    renderRuntimeGuideFC(data);
    renderSensitiveCaptureBadgesFC();
    if (openLastLogBtnEl) {
      openLastLogBtnEl.disabled = !hasOpenableSubmitLogFC();
    }
    refreshActiveSubmitLogFC();
    if (!recording && !inFlight) {
      const readyReason = captureBlockedReasonFC('audio');
      if (readyReason) {
        setStatus(readyReason, true);
      } else if (String(statusLineEl?.textContent || '').trim() === '' || String(statusLineEl?.textContent || '').trim() === 'Ready.') {
        setStatus('Ready to capture.');
      }
    }
    if (!fcWorkspacePrompted) {
      fcWorkspacePrompted = true;
      if (!configLocked && !selectedWorkspace) {
        openWorkspaceModal();
      }
    }
  } catch (err) {
    setStatusChipFC(stateLineEl, 'Session', 'State check failed', 'error');
    stateLineEl.title = 'State check failed: ' + err.message;
    setStatus('State check failed: ' + err.message, true);
  }
}

async function postForm(path, formData) {
  const res = await fetch(path, { method: 'POST', headers: authHeaders(true), body: formData });
  const txt = await res.text();
  if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
  return txt ? JSON.parse(txt) : {};
}

async function postJSON(path, body) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { ...authHeaders(true), 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  const txt = await res.text();
  if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
  return txt ? JSON.parse(txt) : {};
}

function escapePreviewHTML(value) {
  return String(value == null ? '' : value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function formatPreviewDuration(ms) {
  const totalSeconds = Math.max(0, Math.round(Number(ms || 0) / 1000));
  if (!totalSeconds) return '';
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) return minutes + 'm ' + String(seconds).padStart(2, '0') + 's';
  return totalSeconds + 's';
}

function renderPreviewMedia(note) {
  const media = [];
  if (note?.screenshot_data_url) {
    media.push('<figure class="preview-media"><figcaption>Snapshot</figcaption><img src="' + escapePreviewHTML(note.screenshot_data_url) + '" alt="Captured snapshot for request preview" /></figure>');
  }
  if (note?.video_data_url) {
    media.push('<figure class="preview-media"><figcaption>Clip</figcaption><video controls playsinline src="' + escapePreviewHTML(note.video_data_url) + '"></video></figure>');
  }
  if (note?.audio_data_url) {
    media.push('<figure class="preview-media"><figcaption>Audio</figcaption><audio controls src="' + escapePreviewHTML(note.audio_data_url) + '"></audio></figure>');
  }
  if (media.length === 0) return '';
  return '<div class="preview-media-grid">' + media.join('') + '</div>';
}

function renderPreviewContextFC(note) {
  const parts = [];
  if (note?.dom_summary) {
    parts.push('<div><strong>DOM:</strong> ' + escapePreviewHTML(note.dom_summary) + '</div>');
  }
  if (Array.isArray(note?.console) && note.console.length) {
    parts.push('<div><strong>Console</strong><br/>' + note.console.map(item => escapePreviewHTML(item)).join('<br/>') + '</div>');
  }
  if (Array.isArray(note?.network) && note.network.length) {
    parts.push('<div><strong>Network</strong><br/>' + note.network.map(item => escapePreviewHTML(item)).join('<br/>') + '</div>');
  }
  if (parts.length === 0) return '';
  return '<div class="preview-note-text">' + parts.join('<br/>') + '</div>';
}

function previewNoteByIDFC(eventID) {
  const notes = Array.isArray(fcLatestPayloadPreviewData?.preview?.notes) ? fcLatestPayloadPreviewData.preview.notes : [];
  return notes.find(note => String(note?.event_id || '') === String(eventID || '')) || null;
}

function syncPreviewSessionStateFC(nextSession) {
  if (!nextSession) return;
  lastState = lastState ? { ...lastState, session: nextSession } : { session: nextSession };
  syncReplayValueCaptureUIFC();
}

function feedbackCountFC() {
  return Array.isArray(lastState?.session?.feedback) ? lastState.session.feedback.length : 0;
}

function syncReplayValueCaptureUIFC() {
  if (!fcCaptureInputValuesToggleEl) return;
  const enabled = !!(lastState?.session?.capture_input_values);
  fcCaptureInputValuesToggleEl.checked = enabled;
  fcCaptureInputValuesToggleEl.disabled = !lastState?.session?.id;
  fcCaptureInputValuesToggleEl.title = enabled
    ? 'Typed values will be included in replay bundles for this session.'
    : 'Typed values stay redacted unless you opt in for this session.';
  if (fcAllowLargeInlineMediaToggleEl) {
    fcAllowLargeInlineMediaToggleEl.checked = !!fcSettings.allow_large_inline_media;
  }
  renderSensitiveCaptureBadgesFC();
}

async function toggleReplayValueCaptureFC() {
  if (!fcCaptureInputValuesToggleEl) return;
  if (!lastState?.session?.id) {
    fcCaptureInputValuesToggleEl.checked = false;
    setStatus('Start a session before changing replay value capture.', true);
    return;
  }
  try {
    const enabled = !!fcCaptureInputValuesToggleEl.checked;
    const data = await postJSON('/api/session/replay/settings', { capture_input_values: enabled });
    syncPreviewSessionStateFC(data?.session);
    setStatus(enabled ? 'Replay bundles will include typed values for this session.' : 'Replay bundles will redact typed values for this session.');
    renderSensitiveCaptureBadgesFC();
  } catch (err) {
    fcCaptureInputValuesToggleEl.checked = !!(lastState?.session?.capture_input_values);
    setStatus('Replay capture update failed: ' + err.message, true);
  }
}

function renderReplayBundleFC(note) {
  const steps = Array.isArray(note?.replay_steps) ? note.replay_steps : [];
  const script = String(note?.playwright_script || '');
  const mode = String(note?.replay_value_mode || '');
  if (!steps.length && !script) return '';
  const parts = ['<div class="preview-note-text"><strong>Repro bundle</strong></div>'];
  if (mode) {
    parts.push('<div class="preview-note-text"><strong>Value capture:</strong> ' + escapePreviewHTML(mode === 'opt_in' ? 'opted in' : 'redacted') + '</div>');
  }
  if (steps.length) {
    parts.push('<div class="preview-note-text"><strong>Steps</strong><br/>' + steps.map(item => escapePreviewHTML(item)).join('<br/>') + '</div>');
  }
  if (script) {
    parts.push('<details class="preview-card" style="margin-top:.55rem;"><summary>Playwright script</summary><pre style="white-space:pre-wrap;">' + escapePreviewHTML(script) + '</pre></details>');
  }
  return parts.join('');
}

async function exportReplayBundleFC(eventID, format) {
  try {
    const qs = new URLSearchParams({ event_id: String(eventID || ''), format: String(format || '') });
    const res = await fetch('/api/session/replay/export?' + qs.toString(), { headers: authHeaders(false) });
    if (!res.ok) {
      throw new Error(await res.text() || 'export failed');
    }
    const blob = await res.blob();
    const href = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = href;
    a.download = format === 'playwright' ? ('replay-' + eventID + '.spec.ts') : ('replay-' + eventID + '.json');
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(href);
    setStatus('Replay export downloaded: ' + a.download + '.');
  } catch (err) {
    setStatus('Replay export failed: ' + err.message, true);
  }
}

async function ensureFeedbackPresentFC() {
  if (feedbackCountFC() > 0) {
    return;
  }
  await refreshState();
  if (feedbackCountFC() > 0) {
    return;
  }
  throw new Error('Capture at least one note first.');
}

async function editPreviewNoteFC(eventID) {
  try {
    if (recording) throw new Error('Stop the current recording before editing a change request.');
    const note = previewNoteByIDFC(eventID);
    if (!note) throw new Error('Change request not found in the current preview.');
    const nextText = window.prompt('Edit change request text:', String(note.text || ''));
    if (nextText === null) return;
    const trimmed = String(nextText || '').trim();
    if (!trimmed) {
      throw new Error('Change request text cannot be empty. Delete the request instead if you want to remove it.');
    }
    const data = await postJSON('/api/session/feedback/update-text', { event_id: eventID, text: trimmed });
    syncPreviewSessionStateFC(data?.session);
    await previewPayloadFC();
    setStatus('Change request updated.');
  } catch (err) {
    setStatus('Change request update failed: ' + err.message, true);
  }
}

async function deletePreviewNoteFC(eventID) {
  try {
    if (recording) throw new Error('Stop the current recording before deleting a change request.');
    const note = previewNoteByIDFC(eventID);
    if (!note) throw new Error('Change request not found in the current preview.');
    const confirmed = window.confirm('Delete this change request?\n\n' + String(note.text || ''));
    if (!confirmed) return;
    const data = await postJSON('/api/session/feedback/delete', { event_id: eventID });
    syncPreviewSessionStateFC(data?.session);
    const feedbackCount = Array.isArray(lastState?.session?.feedback) ? lastState.session.feedback.length : 0;
    if (feedbackCount > 0) {
      await previewPayloadFC();
    } else {
      fcLatestPayloadPreviewData = null;
      renderPayloadPreview({ preview: { notes: [] } });
    }
    setStatus('Change request deleted.');
  } catch (err) {
    setStatus('Change request delete failed: ' + err.message, true);
  }
}

function renderPayloadPreview(data) {
  if (!fcPayloadPreviewEl) return;
  fcLatestPayloadPreviewData = data || null;
  const preview = data?.preview;
  const notes = Array.isArray(preview?.notes) ? preview.notes : [];
  if (notes.length === 0) {
    resetPreviewDeliveryOptionsFC();
    fcPayloadPreviewEl.className = 'request-preview small';
    fcPayloadPreviewEl.textContent = 'Preview the request here before sending it to the agent.';
    if (fcPreviewDetailsEl) fcPreviewDetailsEl.open = false;
    return;
  }
  const provider = String(data?.provider || '');
  const summary = String(preview?.summary || '');
  const destination = providerDestinationLabelFC(provider);
  const count = Math.max(1, Number(preview?.change_request_count || notes.length));
  const intentLabel = String(preview?.intent_label || 'Implement changes');
  const warnings = Array.isArray(preview?.warnings) ? preview.warnings : [];
  const disclosureBlock = renderDisclosureSummaryFC(preview);
  const oversizedVideoBlock = renderOversizedVideoWarningActionsFC(preview);
  const noteCards = notes.map((note, index) => {
    const meta = [];
    if (note?.target) meta.push('Target: ' + escapePreviewHTML(note.target));
    if (note?.review_mode) meta.push('Mode: ' + escapePreviewHTML(note.review_mode));
    if (note?.video_duration_ms) meta.push('Clip: ' + escapePreviewHTML(formatPreviewDuration(note.video_duration_ms)));
    if (note?.pointer_event_count) meta.push('Events: ' + escapePreviewHTML(String(note.pointer_event_count)));
    if (note?.replay_step_count) meta.push('Replay: ' + escapePreviewHTML(String(note.replay_step_count)));
    if (Array.isArray(note?.console) && note.console.length) meta.push('Console: ' + escapePreviewHTML(String(note.console.length)));
    if (Array.isArray(note?.network) && note.network.length) meta.push('Network: ' + escapePreviewHTML(String(note.network.length)));
    if (note?.has_audio && !note?.audio_data_url) meta.push('Transcript from audio note');
    const eventID = String(note?.event_id || '');
    return '<article class="preview-note-card">' +
      '<div class="preview-note-header"><strong>Change request ' + (index + 1) + '</strong><span class="small">' + escapePreviewHTML(note?.event_id || '') + '</span></div>' +
      (meta.length ? '<div class="preview-note-meta">' + meta.map(item => '<span>' + item + '</span>').join('') + '</div>' : '') +
      '<div class="preview-note-text">' + escapePreviewHTML(note?.text || '') + '</div>' +
      renderPreviewContextFC(note) +
      renderReplayBundleFC(note) +
      '<div class="mini-toolbar">' +
      '<button type="button" onclick="editPreviewNoteFC(\'' + escapePreviewHTML(eventID) + '\')" title="Edit change request text">Edit text</button>' +
      '<button type="button" class="danger" onclick="deletePreviewNoteFC(\'' + escapePreviewHTML(eventID) + '\')" title="Delete change request">Delete</button>' +
      '<button type="button" onclick="exportReplayBundleFC(\'' + escapePreviewHTML(eventID) + '\', \'json\')" title="Export replay JSON">Replay JSON</button>' +
      '<button type="button" onclick="exportReplayBundleFC(\'' + escapePreviewHTML(eventID) + '\', \'playwright\')" title="Export Playwright script">Playwright</button>' +
      '</div>' +
      renderPreviewVideoDecisionActionsFC(note) +
      renderPreviewMedia(note) +
      '</article>';
  }).join('');
  const warningBlock = warnings.length
    ? '<div class="preview-warning-card"><strong>Review media before sending.</strong><div class="preview-note-text">' + warnings.map(item => escapePreviewHTML(item)).join('<br/>') + '</div><div class="small" style="margin-top:.45rem;color:#6a7383;">Choose how to proceed on the affected request: make the clip smaller, rely on a snapshot, or explicitly allow the larger clip.</div></div>'
    : '';
  fcPayloadPreviewEl.className = 'request-preview';
  if (fcPreviewDetailsEl) fcPreviewDetailsEl.open = true;
  fcPayloadPreviewEl.innerHTML = '<div class="preview-summary-card">' +
    '<div class="preview-kicker">Ready to send</div>' +
    '<div class="preview-summary-line"><strong>' + count + ' request' + (count === 1 ? '' : 's') + ' prepared</strong><span class="small">' + escapePreviewHTML(destination) + '</span></div>' +
    '<div class="preview-note-meta"><span>Action: ' + escapePreviewHTML(intentLabel) + '</span></div>' +
    (summary ? '<div class="preview-note-text">' + escapePreviewHTML(summary) + '</div>' : '') +
    '</div>' +
    disclosureBlock +
    oversizedVideoBlock +
    noteCards +
    warningBlock;
}

function fcLiveVideoActive() {
  return !!(fcLiveDisplayStream && fcLiveDisplayStream.getVideoTracks && fcLiveDisplayStream.getVideoTracks().some(t => t.readyState === 'live'));
}

function setFCLiveVideoState(msg, isError = false) {
  if (!fcLiveVideoStateEl) return;
  fcLiveVideoStateEl.textContent = msg;
  fcLiveVideoStateEl.style.color = isError ? '#c34f4f' : '#6a7383';
}

async function startLiveVideoFC() {
  if (inFlight || recording) return;
  if (fcLiveVideoActive()) {
    setFCLiveVideoState('Live video: on');
    return;
  }
  try {
    if (!requireCompanionFC('capture live video')) return;
    setStatus('Select the target window/tab for live video.');
    const stream = await navigator.mediaDevices.getDisplayMedia({
      video: { frameRate: { ideal: 15, max: 30 } },
      audio: false
    });
    const track = stream.getVideoTracks()[0];
    if (!track) throw new Error('display stream unavailable');
    track.addEventListener('ended', () => {
      stopLiveVideoFC(true, 'live video source ended');
    });
    fcLiveDisplayStream = stream;
    if (fcLivePreviewEl) {
      fcLivePreviewEl.srcObject = stream;
      fcLivePreviewEl.classList.remove('hidden');
      await fcLivePreviewEl.play();
    }
    setFCLiveVideoState('Live video: on', false);
    setStatus('Live video enabled in composer.');
  } catch (err) {
    setFCLiveVideoState('Live video: failed', true);
    setStatus('Live video failed: ' + err.message, true);
  }
}

async function stopLiveVideoFC(silent = false) {
  try {
    if (fcLiveDisplayStream) {
      fcLiveDisplayStream.getTracks().forEach(t => t.stop());
    }
  } catch (_) {}
  fcLiveDisplayStream = null;
  if (fcLivePreviewEl) {
    try { fcLivePreviewEl.pause(); } catch (_) {}
    fcLivePreviewEl.srcObject = null;
    fcLivePreviewEl.classList.add('hidden');
  }
  setFCLiveVideoState('Live video: off', false);
  if (!silent) {
    setStatus('Live video stopped.');
  }
}

function selectedProviderFC() {
  const provider = String(fcAgentDefaultProviderEl?.value || lastState?.runtime_codex?.default_provider || '').trim();
  if (provider) return provider;
  const adapters = Array.isArray(lastState?.adapters) ? lastState.adapters : [];
  if (adapters.includes('codex_cli')) return 'codex_cli';
  if (adapters.includes('cli')) return 'cli';
  if (adapters.length > 0) return String(adapters[0]);
  return 'codex_cli';
}

function defaultDeliveryIntentPromptFC(profile) {
  switch (String(profile || '').trim()) {
    case 'draft_plan':
      return [
        'You are receiving a canonical Knit feedback payload JSON.',
        'Produce a concrete implementation plan for the requested software changes without editing the repository.',
        '',
        'Rules:',
        '- Do not edit files or make repository changes.',
        '- Focus on scope, intended behavior, risks, sequencing, and validation steps.',
        '- Call out assumptions and edge cases clearly.',
        '- Return a concise, actionable plan rather than implementation code.',
        '- If the repository is already dirty, use that only as context; do not modify it.',
        '',
        'External tracker operations are disabled for this run.',
        'Do not call Jira/Atlassian/GitHub issue tools.'
      ].join('\n');
    case 'create_jira_tickets':
      return [
        'You are receiving a canonical Knit feedback payload JSON.',
        'Turn the approved feedback into Jira-ready implementation tickets.',
        '',
        'Rules:',
        '- Create a clear breakdown of work with concise titles, descriptions, and acceptance criteria.',
        '- If tracker tools are available, you may create or update Jira issues.',
        '- If tracker tools are unavailable, return a structured ticket bundle that can be copied into Jira.',
        '- Do not modify the repository unless it is necessary to inspect the codebase and understand the requested work.',
        '- Focus on actionable project-management output, not code edits.',
        '',
        'External tracker operations are allowed for this run when they help complete the requested Jira-ticket workflow.'
      ].join('\n');
    default:
      return [
        'You are receiving a canonical Knit feedback payload JSON.',
        'Implement the requested software changes in the current repository.',
        '',
        'Rules:',
        '- Apply changes directly in the working tree.',
        '- Keep edits minimal and maintainable.',
        '- Run relevant tests.',
        '- Return a short summary at the end.',
        '- Focus on implementation and validation, not project-management workflow.',
        '- If the repository is already dirty, continue and only modify files required for this request.',
        '- Do not stop to ask "how should I proceed" in exec mode; make the safest minimal assumption and continue.',
        '- Do not exit after only creating/updating tickets, issues, or external tracker metadata.',
        '',
        'External tracker operations are disabled for this run.',
        'Do not call Jira/Atlassian/GitHub issue tools; implement code + tests directly.'
      ].join('\n');
  }
}

function selectedDeliveryIntentProfileFC() {
  const profile = String(fcDeliveryIntentProfileEl?.value || 'implement_changes').trim();
  if (profile === 'draft_plan' || profile === 'create_jira_tickets') return profile;
  return 'implement_changes';
}

function selectedDeliveryIntentLabelFC() {
  switch (selectedDeliveryIntentProfileFC()) {
    case 'draft_plan':
      return 'Draft plan';
    case 'create_jira_tickets':
      return 'Create Jira tickets';
    default:
      return 'Implement changes';
  }
}

function selectedDeliveryInstructionTextFC() {
  const profile = selectedDeliveryIntentProfileFC();
  const text = String(fcDeliveryInstructionTextEl?.value || '').trim();
  return text || defaultDeliveryIntentPromptFC(profile);
}

function syncDeliveryIntentPromptTextFC(force = false) {
  if (!fcDeliveryInstructionTextEl) return;
  const template = defaultDeliveryIntentPromptFC(selectedDeliveryIntentProfileFC());
  if (force || !String(fcDeliveryInstructionTextEl.value || '').trim()) {
    fcDeliveryInstructionTextEl.value = template;
  }
}

async function approveSessionFC(silent = false, reason = 'send') {
  try {
    const data = await postJSON('/api/session/approve', { summary: '' });
    setPreviewState('Prepared ' + (data.change_requests?.length || 0) + ' change requests for ' + reason + '.');
    return data;
  } catch (err) {
    if (!silent) {
      setPreviewState('Could not prepare the request: ' + err.message, true);
    }
    throw err;
  }
}

async function captureScreenshotBlobFC() {
  if (!requireCompanionFC('capture snapshots')) return null;
  let stream = null;
  try {
    let video = null;
    if (fcLiveVideoActive() && fcLivePreviewEl?.srcObject) {
      video = fcLivePreviewEl;
    } else {
      setStatus('Select the target window/tab to capture a snapshot.');
      stream = await navigator.mediaDevices.getDisplayMedia({
        video: { frameRate: { ideal: 1, max: 1 } },
        audio: false,
      });
      const track = stream.getVideoTracks()[0];
      if (!track) throw new Error('display stream unavailable');
      video = document.createElement('video');
      video.srcObject = stream;
      video.muted = true;
      video.playsInline = true;
      await video.play();
      await new Promise(resolve => setTimeout(resolve, 120));
    }
    const width = Math.max(1, video.videoWidth || 1280);
    const height = Math.max(1, video.videoHeight || 720);
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext('2d');
    if (!ctx) throw new Error('canvas context unavailable');
    ctx.drawImage(video, 0, 0, width, height);
    const blob = await new Promise(resolve => canvas.toBlob(resolve, 'image/jpeg', 0.88));
    if (!blob || !blob.size) throw new Error('snapshot capture returned empty data');
    return blob;
  } finally {
    if (stream) stream.getTracks().forEach(t => t.stop());
  }
}

async function consumeSnapshotForNoteFC() {
  if (queuedScreenshotBlob) {
    return queuedScreenshotBlob;
  }
  return captureScreenshotBlobFC();
}

function getTargetWindowLabelFC() {
  return String(lastState?.session?.target_window || 'Browser Review').trim() || 'Browser Review';
}

function createAudioRecorderForStream(stream) {
  try {
    return new MediaRecorder(stream, { mimeType: 'audio/webm' });
  } catch (_) {
    return new MediaRecorder(stream);
  }
}

function clearActiveAudioNoteFC() {
  if (fcAudioNoteStream) {
    try {
      fcAudioNoteStream.getTracks().forEach(track => track.stop());
    } catch (_) {}
  }
  fcActiveAudioNote = null;
  fcAudioNoteRecorder = null;
  fcAudioNoteStream = null;
  fcAudioNoteChunks = [];
  fcAudioNoteStopPromise = null;
  setRecordingState(false, 0);
}

function clearActiveVideoNoteFC() {
  [fcVideoNoteMicStream, fcVideoNoteDisplayStream, fcVideoNoteCombinedStream].forEach((stream) => {
    try {
      stream?.getTracks?.().forEach(track => track.stop());
    } catch (_) {}
  });
  fcActiveVideoNote = null;
  fcVideoNoteAudioRecorder = null;
  fcVideoNoteClipRecorder = null;
  fcVideoNoteMicStream = null;
  fcVideoNoteDisplayStream = null;
  fcVideoNoteCombinedStream = null;
  fcVideoNoteAudioChunks = [];
  fcVideoNoteClipChunks = [];
  fcVideoNoteAudioStopPromise = null;
  fcVideoNoteClipStopPromise = null;
  fcVideoNoteFinalizing = false;
  setRecordingState(false, 0);
}

function createVideoRecorderForStream(stream) {
  const mimeTypes = ['video/webm;codecs=vp9,opus', 'video/webm;codecs=vp8,opus', 'video/webm'];
  for (const mimeType of mimeTypes) {
    try {
      if (!mimeType || MediaRecorder.isTypeSupported(mimeType)) {
        return new MediaRecorder(stream, mimeType ? { mimeType } : undefined);
      }
    } catch (_) {}
  }
  return new MediaRecorder(stream);
}

async function getMicrophoneStreamFC() {
  const mode = fcAudioModeEl?.value || 'always_on';
  if (mode === 'push_to_talk' && !fcPTTHeld) {
    throw new Error('Push to talk only records while Space is held. Switch Audio mode to always on for hands-free review.');
  }
  if (fcAudioMutedEl?.checked) {
    throw new Error('Audio is muted. Unmute before recording.');
  }
  if (fcAudioPausedEl?.checked) {
    throw new Error('Audio is paused. Resume before recording.');
  }
  const deviceID = (fcAudioInputDeviceEl && fcAudioInputDeviceEl.value) ? String(fcAudioInputDeviceEl.value) : '';
  return navigator.mediaDevices.getUserMedia({
    audio: deviceID ? { deviceId: { exact: deviceID } } : true,
    video: false
  });
}

async function submitTextNote(options = {}) {
  if (inFlight) return;
  if (recording) {
    setStatus('Stop the current recording before adding a typed note.', true);
    return;
  }
  try {
    setBusy(true);
    if (!lastState?.session?.id) throw new Error('Start a session first.');
    if (options.openEditor !== false) {
      ensureTextEditorOpenFC();
    }
    const text = String(transcriptEl.value || '').trim();
    if (!text) {
      setStatus(options.withSnapshot
        ? 'Typed note field opened. Enter your note, then click Snapshot + typed note again to capture it.'
        : 'Typed note field opened. Enter your note, then preview, submit, or capture it with a snapshot.');
      return;
    }
    const form = new FormData();
    form.append('raw_transcript', text);
    form.append('normalized', text);
    let usedQueuedSnapshot = false;
    if (options.withSnapshot) {
      const screenshot = await consumeSnapshotForNoteFC();
      if (screenshot) {
        usedQueuedSnapshot = screenshot === queuedScreenshotBlob;
        form.append('screenshot', screenshot, 'floating-note.jpg');
      }
    }
    const data = await postForm('/api/session/feedback/note', form);
    if (data && data.command_handled) {
      const cmd = String(data.command_result?.voice_command || 'voice_command');
      setStatus('Voice command handled: ' + cmd);
      await refreshState();
      return;
    }
    const eventID = data && data.event_id ? data.event_id : 'event';
    const hasScreenshot = !!(data && data.screenshot_ref);
    if (usedQueuedSnapshot) {
      queuedScreenshotBlob = null;
      renderScreenshotQueueState();
    }
    setStatus('Typed note captured: ' + eventID + '. Snapshot attached: ' + (hasScreenshot ? 'yes' : 'no') + '.');
    markPreviewStale('Preview needs refresh after your latest note.');
    transcriptEl.value = '';
    await refreshState();
  } catch (err) {
    setStatus('Text note failed: ' + err.message, true);
  } finally {
    setBusy(false);
  }
}

async function startAudioNoteCaptureFC(options = {}) {
  const stream = await getMicrophoneStreamFC();
  let rec;
  try {
    rec = createAudioRecorderForStream(stream);
    let screenshot = null;
    let usedQueuedSnapshot = false;
    if (options.withSnapshot) {
      screenshot = await consumeSnapshotForNoteFC();
      usedQueuedSnapshot = screenshot === queuedScreenshotBlob;
    }
    fcAudioNoteStream = stream;
    fcAudioNoteChunks = [];
    fcAudioNoteStopPromise = new Promise(resolve => {
      rec.addEventListener('stop', resolve, { once: true });
    });
    rec.ondataavailable = evt => { if (evt.data && evt.data.size > 0) fcAudioNoteChunks.push(evt.data); };
    fcAudioNoteRecorder = rec;
    fcActiveAudioNote = {
      screenshot,
      usedQueuedSnapshot,
      withSnapshot: !!options.withSnapshot,
    };
    fcRecordingKind = 'audio';
    fcAudioNoteMode = options.withSnapshot ? 'snapshot' : 'talk';
    rec.start();
    setRecordingState(true, 0);
    setStatus(options.withSnapshot
      ? 'Recording snapshot + voice note. Click the active button again to stop.'
      : 'Recording voice note. Click the active button again to stop.');
  } catch (err) {
    try {
      stream.getTracks().forEach(track => track.stop());
    } catch (_) {}
    clearActiveAudioNoteFC();
    throw err;
  }
}

async function finishAudioNoteCaptureFC() {
  if (!fcAudioNoteRecorder) {
    throw new Error('Voice note recording is not active.');
  }
  const rec = fcAudioNoteRecorder;
  const stopPromise = fcAudioNoteStopPromise;
  if (rec.state !== 'inactive') {
    rec.stop();
  }
  if (stopPromise) {
    await stopPromise;
  }
  const audioBlob = new Blob(fcAudioNoteChunks, { type: rec.mimeType || 'audio/webm' });
  const note = fcActiveAudioNote || { screenshot: null, usedQueuedSnapshot: false, withSnapshot: false };
  clearActiveAudioNoteFC();
  return { audioBlob, note };
}

async function startVideoNoteCaptureFC() {
  if (!requireCompanionFC('record video clips')) {
    return null;
  }
  try {
    fcVideoNoteMicStream = await getMicrophoneStreamFC();
    setStatus('Select the target window/tab for the video clip.');
    fcVideoNoteDisplayStream = await navigator.mediaDevices.getDisplayMedia({
      video: { frameRate: { ideal: 15, max: 30 } },
      audio: true
    });
    const videoTracks = fcVideoNoteDisplayStream.getVideoTracks();
    if (!videoTracks.length) {
      throw new Error('display stream unavailable');
    }
    videoTracks.forEach((track) => {
      track.addEventListener('ended', () => {
        if (fcVideoNoteFinalizing || !fcVideoNoteClipRecorder) {
          return;
        }
        setStatus('Video sharing ended. Finalizing your video note...');
        window.setTimeout(() => {
          finalizeVideoNoteCaptureFC('share end');
        }, 0);
      }, { once: true });
    });
    fcVideoNoteCombinedStream = new MediaStream();
    videoTracks.forEach(track => fcVideoNoteCombinedStream.addTrack(track));
    fcVideoNoteDisplayStream.getAudioTracks().forEach(track => fcVideoNoteCombinedStream.addTrack(track));
    fcVideoNoteMicStream.getAudioTracks().forEach(track => fcVideoNoteCombinedStream.addTrack(track));

    fcVideoNoteAudioRecorder = createAudioRecorderForStream(fcVideoNoteMicStream);
    fcVideoNoteClipRecorder = createVideoRecorderForStream(fcVideoNoteCombinedStream);
    fcVideoNoteAudioChunks = [];
    fcVideoNoteClipChunks = [];
    fcVideoNoteAudioStopPromise = new Promise(resolve => {
      fcVideoNoteAudioRecorder.addEventListener('stop', resolve, { once: true });
    });
    fcVideoNoteClipStopPromise = new Promise(resolve => {
      fcVideoNoteClipRecorder.addEventListener('stop', resolve, { once: true });
    });
    fcVideoNoteAudioRecorder.ondataavailable = (evt) => {
      if (evt.data && evt.data.size > 0) fcVideoNoteAudioChunks.push(evt.data);
    };
    fcVideoNoteClipRecorder.ondataavailable = (evt) => {
      if (evt.data && evt.data.size > 0) fcVideoNoteClipChunks.push(evt.data);
    };

    fcActiveVideoNote = {
      startedAt: Date.now(),
      window: getTargetWindowLabelFC(),
      hasAudio: fcVideoNoteCombinedStream.getAudioTracks().length > 0,
    };
    fcRecordingKind = 'video';
    setRecordingState(true, 0);
    fcVideoNoteAudioRecorder.start();
    fcVideoNoteClipRecorder.start();
    setStatus('Recording video note. Click Video + voice again to stop.');
    return true;
  } catch (err) {
    clearActiveVideoNoteFC();
    throw err;
  }
}

async function finishVideoNoteCaptureFC() {
  if (!fcVideoNoteAudioRecorder || !fcVideoNoteClipRecorder) {
    throw new Error('Video note recording is not active.');
  }
  const audioRecorder = fcVideoNoteAudioRecorder;
  const clipRecorder = fcVideoNoteClipRecorder;
  const audioStopPromise = fcVideoNoteAudioStopPromise;
  const clipStopPromise = fcVideoNoteClipStopPromise;
  const meta = fcActiveVideoNote || { startedAt: Date.now(), window: getTargetWindowLabelFC(), hasAudio: false };
  if (audioRecorder.state !== 'inactive') {
    audioRecorder.stop();
  }
  if (clipRecorder.state !== 'inactive') {
    clipRecorder.stop();
  }
  await Promise.all([audioStopPromise, clipStopPromise].filter(Boolean));
  const endedAt = Date.now();
  const audioBlob = new Blob(fcVideoNoteAudioChunks, { type: audioRecorder.mimeType || 'audio/webm' });
  const clipBlob = new Blob(fcVideoNoteClipChunks, { type: clipRecorder.mimeType || 'video/webm' });
  clearActiveVideoNoteFC();
  return {
      audioBlob,
      clip: {
        blob: clipBlob,
        codec: clipRecorder.mimeType || 'video/webm',
        hasAudio: !!meta.hasAudio,
        pointerOverlay: true,
        scope: 'window',
        window: meta.window,
        startedAt: new Date(meta.startedAt).toISOString(),
        endedAt: new Date(endedAt).toISOString(),
        durationMS: Math.max(0, endedAt - meta.startedAt)
      }
    };
}

async function submitAudioNote(options = {}) {
  if (inFlight) return;
  try {
    if (!fcAudioNoteRecorder) {
      if (recording) return;
      if (!ensureCaptureReadyFC(options.withSnapshot ? 'snapshot' : 'audio')) return;
      if (lastState?.transcription_mode === 'remote' && !lastState?.allow_remote_stt) {
        throw new Error('Remote transcription is disabled by policy.');
      }
      await startAudioNoteCaptureFC(options);
      return;
    }
    setBusy(true);
    const captured = await finishAudioNoteCaptureFC();
    const audio = captured.audioBlob;
    const usedQueuedSnapshot = !!captured.note.usedQueuedSnapshot;
    const screenshot = captured.note.screenshot;
    const form = new FormData();
    form.append('audio', audio, 'floating-note.webm');
    if (screenshot) {
      form.append('screenshot', screenshot, 'floating-note.jpg');
    }
    const data = await postForm('/api/session/feedback/note', form);
    if (data && data.command_handled) {
      const cmd = String(data.command_result?.voice_command || 'voice_command');
      setStatus('Voice command handled: ' + cmd);
      await refreshState();
      return;
    }
    const eventID = data && data.event_id ? data.event_id : 'event';
    const hasScreenshot = !!(data && data.screenshot_ref);
    if (usedQueuedSnapshot) {
      queuedScreenshotBlob = null;
      renderScreenshotQueueState();
    }
    setStatus('Voice note captured: ' + eventID + '. Snapshot attached: ' + (hasScreenshot ? 'yes' : 'no') + '.');
    markPreviewStale('Preview needs refresh after your latest note.');
    await refreshState();
  } catch (err) {
    setStatus('Audio note failed: ' + err.message, true);
  } finally {
    setBusy(false);
  }
}

async function submitAudioNoteWithSnapshot() {
  return submitAudioNote({ withSnapshot: true });
}

async function submitTextNoteWithSnapshot() {
  ensureTextEditorOpenFC();
  return submitTextNote({ withSnapshot: true, openEditor: true });
}

function appendClipMetadataFC(form, clip) {
  if (!clip || !form) return;
  if (clip.codec) form.append('video_codec', String(clip.codec));
  if (clip.scope) form.append('video_scope', String(clip.scope));
  if (clip.window) form.append('video_window', String(clip.window));
  if (clip.startedAt) form.append('clip_started_at', String(clip.startedAt));
  if (clip.endedAt) form.append('clip_ended_at', String(clip.endedAt));
  if (Number.isFinite(Number(clip.durationMS)) && Number(clip.durationMS) > 0) {
    form.append('clip_duration_ms', String(Math.round(Number(clip.durationMS))));
  }
  form.append('video_has_audio', clip.hasAudio ? '1' : '0');
  form.append('video_pointer_overlay', clip.pointerOverlay ? '1' : '0');
}

function currentSessionIDFC() {
  return String(lastState?.session?.id || '').trim();
}

function formatMediaSizeFC(bytes) {
  const size = Number(bytes || 0);
  if (!Number.isFinite(size) || size <= 0) return '0 bytes';
  if (size >= (1 << 20)) return (size / (1 << 20)).toFixed(1) + ' MB';
  if (size >= (1 << 10)) return (size / (1 << 10)).toFixed(1) + ' KB';
  return Math.round(size) + ' bytes';
}

function clipCacheMetaFromClipFC(clip) {
  if (!clip) return {};
  return {
    codec: String(clip.codec || clip.blob?.type || 'video/webm'),
    hasAudio: !!clip.hasAudio,
    pointerOverlay: !!clip.pointerOverlay,
    scope: String(clip.scope || 'window'),
    window: String(clip.window || ''),
    durationMS: Number(clip.durationMS || 0),
  };
}

function cacheClipForEventFC(eventID, clip) {
  const id = String(eventID || '').trim();
  if (!id || !clip?.blob) return;
  fcClipBlobCacheByEventID.set(id, {
    sessionID: currentSessionIDFC(),
    blob: clip.blob,
    meta: clipCacheMetaFromClipFC(clip),
  });
}

function cachedClipEntryForEventFC(eventID) {
  const id = String(eventID || '').trim();
  if (!id) return null;
  const entry = fcClipBlobCacheByEventID.get(id);
  if (!entry) return null;
  if (entry.sessionID && entry.sessionID !== currentSessionIDFC()) {
    fcClipBlobCacheByEventID.delete(id);
    return null;
  }
  return entry;
}

async function fetchStoredClipForEventFC(eventID) {
  const id = String(eventID || '').trim();
  if (!id) throw new Error('event id is required');
  const res = await fetch('/api/session/feedback/clip?event_id=' + encodeURIComponent(id), {
    headers: authHeaders(false),
  });
  if (!res.ok) {
    throw new Error(await res.text() || ('HTTP ' + res.status));
  }
  const blob = await res.blob();
  if (!blob || !blob.size) {
    throw new Error('stored clip is empty');
  }
  return {
    blob,
    meta: {
      codec: String(res.headers.get('X-Knit-Video-Codec') || blob.type || 'video/webm'),
    },
  };
}

async function clipEntryForPreviewNoteFC(note) {
  const eventID = String(note?.event_id || '').trim();
  if (!eventID) throw new Error('Preview note is missing its event id.');
  const cached = cachedClipEntryForEventFC(eventID);
  if (cached?.blob?.size) {
    return {
      blob: cached.blob,
      meta: {
        ...cached.meta,
        scope: cached.meta?.scope || String(note?.video_scope || 'window'),
        window: cached.meta?.window || String(note?.video_window || ''),
        durationMS: Number(cached.meta?.durationMS || note?.video_duration_ms || 0),
        pointerOverlay: cached.meta?.pointerOverlay ?? !!note?.video_pointer_overlay,
      },
    };
  }
  const stored = await fetchStoredClipForEventFC(eventID);
  return {
    blob: stored.blob,
    meta: {
      codec: stored.meta?.codec || String(note?.video_codec || stored.blob.type || 'video/webm'),
      hasAudio: !!note?.video_has_audio,
      pointerOverlay: !!note?.video_pointer_overlay,
      scope: String(note?.video_scope || 'window'),
      window: String(note?.video_window || ''),
      durationMS: Number(note?.video_duration_ms || 0),
    },
  };
}

async function loadVideoElementForBlobFC(blob) {
  if (!blob || !blob.size) throw new Error('clip is empty');
  const url = URL.createObjectURL(blob);
  const video = document.createElement('video');
  video.preload = 'auto';
  video.muted = true;
  video.playsInline = true;
  video.src = url;
  try {
    await new Promise((resolve, reject) => {
      const cleanup = () => {
        video.removeEventListener('loadedmetadata', onReady);
        video.removeEventListener('error', onError);
      };
      const onReady = () => {
        cleanup();
        resolve();
      };
      const onError = () => {
        cleanup();
        reject(new Error('clip could not be decoded in this browser'));
      };
      video.addEventListener('loadedmetadata', onReady, { once: true });
      video.addEventListener('error', onError, { once: true });
    });
    return { video, url };
  } catch (err) {
    URL.revokeObjectURL(url);
    throw err;
  }
}

function resizedVideoProfilesFC(width, height) {
  const even = (value, fallback) => {
    const rounded = Math.max(2, Math.round(Number(value || fallback || 2)));
    return rounded % 2 === 0 ? rounded : rounded - 1;
  };
  return [
    { scale: 1, fps: 8, bitrate: 650_000 },
    { scale: 0.85, fps: 8, bitrate: 520_000 },
    { scale: 0.7, fps: 6, bitrate: 380_000 },
    { scale: 0.55, fps: 6, bitrate: 260_000 },
    { scale: 0.45, fps: 5, bitrate: 180_000 },
  ].map((profile) => ({
    width: even(width * profile.scale, width),
    height: even(height * profile.scale, height),
    fps: profile.fps,
    bitrate: profile.bitrate,
  }));
}

function createConstrainedVideoRecorderFC(stream, bitrate) {
  const attempts = [
    { mimeType: 'video/webm;codecs=vp9', videoBitsPerSecond: bitrate },
    { mimeType: 'video/webm;codecs=vp8', videoBitsPerSecond: Math.round(bitrate * 0.9) },
    { mimeType: 'video/webm', videoBitsPerSecond: Math.round(bitrate * 0.8) },
    undefined,
  ];
  for (const options of attempts) {
    try {
      if (!options || !options.mimeType || MediaRecorder.isTypeSupported(options.mimeType)) {
        return new MediaRecorder(stream, options);
      }
    } catch (_) {}
  }
  return new MediaRecorder(stream);
}

async function recordReducedVideoProfileFC(video, profile) {
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(2, profile.width);
  canvas.height = Math.max(2, profile.height);
  const ctx = canvas.getContext('2d', { alpha: false });
  if (!ctx) throw new Error('video resize canvas is unavailable');
  const stream = canvas.captureStream(Math.max(4, Number(profile.fps || 6)));
  const recorder = createConstrainedVideoRecorderFC(stream, Math.max(120000, Number(profile.bitrate || 300000)));
  const chunks = [];
  let raf = 0;
  recorder.ondataavailable = (evt) => {
    if (evt.data && evt.data.size > 0) chunks.push(evt.data);
  };
  const drawFrame = () => {
    if (video.ended || video.paused) return;
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    raf = requestAnimationFrame(drawFrame);
  };
  const stopRecorder = new Promise((resolve) => {
    recorder.addEventListener('stop', resolve, { once: true });
  });
  const playbackEnded = new Promise((resolve, reject) => {
    video.addEventListener('ended', resolve, { once: true });
    video.addEventListener('error', () => reject(new Error('clip playback failed during resize')), { once: true });
  });
  video.currentTime = 0;
  recorder.start(250);
  ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
  raf = requestAnimationFrame(drawFrame);
  await video.play();
  await playbackEnded;
  if (raf) cancelAnimationFrame(raf);
  if (recorder.state !== 'inactive') {
    recorder.stop();
  }
  await stopRecorder;
  stream.getTracks().forEach(track => track.stop());
  return new Blob(chunks, { type: recorder.mimeType || 'video/webm' });
}

async function shrinkClipBlobToLimitFC(blob, maxBytes, statusLabel) {
  const { video, url } = await loadVideoElementForBlobFC(blob);
  try {
    const width = Math.max(2, Number(video.videoWidth || 640));
    const height = Math.max(2, Number(video.videoHeight || 360));
    const profiles = resizedVideoProfilesFC(width, height);
    let best = null;
    for (const profile of profiles) {
      if (statusLabel) {
        statusLabel('Trying ' + profile.width + '×' + profile.height + ' at ' + profile.fps + 'fps...');
      }
      const candidate = await recordReducedVideoProfileFC(video, profile);
      if (!best || candidate.size < best.size) {
        best = candidate;
      }
      if (candidate.size > 0 && candidate.size <= maxBytes) {
        return candidate;
      }
      video.pause();
    }
    return best;
  } finally {
    try { video.pause(); } catch (_) {}
    video.removeAttribute('src');
    video.load();
    URL.revokeObjectURL(url);
  }
}

function previewNoteNeedsClipResizeFC(note) {
  const status = String(note?.video_transmission_status || '').trim();
  const size = Number(note?.video_size_bytes || 0);
  const limit = Number(note?.video_send_limit_bytes || 0);
  return !!note?.has_video && ((status === 'omitted_due_to_limit') || (limit > 0 && size > limit));
}

function renderPreviewVideoDecisionActionsFC(note) {
  if (!previewNoteNeedsClipResizeFC(note)) return '';
  const eventID = String(note?.event_id || '').trim();
  if (!eventID) return '';
  const busy = fcClipResizeInFlight.has(eventID);
  const buttonLabel = busy ? 'Making clip smaller…' : 'Make clip smaller to send';
  const snapshotLabel = previewVideoEventOmittedFC(eventID)
    ? 'Send clip again'
    : (note?.has_screenshot ? 'Use snapshot instead' : 'Omit clip for this request');
  return '<div class="mini-toolbar" style="margin-top:.5rem;">' +
    '<button type="button" ' + (busy ? 'disabled ' : '') + 'onclick="fitPreviewClipToSendLimitFC(\'' + escapePreviewHTML(eventID) + '\')" title="' + escapePreviewHTML(buttonLabel) + '">' + escapePreviewHTML(buttonLabel) + '</button>' +
    '<button type="button" onclick="togglePreviewVideoEventOmissionFC(\'' + escapePreviewHTML(eventID) + '\')" title="' + escapePreviewHTML(snapshotLabel) + '">' + escapePreviewHTML(snapshotLabel) + '</button>' +
    '</div><div class="small" style="margin-top:.35rem;color:#6a7383;">Knit will lower fps, bitrate, and size until the clip fits. The smaller clip may drop clip audio.</div>';
}

async function fitPreviewClipToSendLimitFC(eventID) {
  const id = String(eventID || '').trim();
  if (!id || fcClipResizeInFlight.has(id)) return;
  const note = previewNoteByIDFC(id);
  if (!note) {
    setStatus('Clip resize failed: change request not found in the current preview.', true);
    return;
  }
  const limit = Number(note.video_send_limit_bytes || 0);
  if (!Number.isFinite(limit) || limit <= 0) {
    setStatus('Clip resize failed: send limit is unavailable for this clip.', true);
    return;
  }
  fcClipResizeInFlight.add(id);
  if (fcLatestPayloadPreviewData) {
    renderPayloadPreview(fcLatestPayloadPreviewData);
  }
  try {
    showToastFC('Reducing clip for ' + id + ' to fit the default send limit…');
    setPreviewState('Making the clip for ' + id + ' smaller so it fits the default send limit of ' + formatMediaSizeFC(limit) + '...');
    const clipEntry = await clipEntryForPreviewNoteFC(note);
    const reduced = await shrinkClipBlobToLimitFC(clipEntry.blob, limit, (message) => {
      setPreviewState('Reducing clip for ' + id + ': ' + message);
    });
    if (!reduced || !reduced.size) {
      throw new Error('Knit could not create a smaller clip in this browser.');
    }
    if (reduced.size > limit) {
      throw new Error('The reduced clip is still ' + formatMediaSizeFC(reduced.size) + '. Use a snapshot instead or allow large inline media.');
    }
    const clipMeta = {
      codec: reduced.type || String(note.video_codec || clipEntry.meta?.codec || 'video/webm'),
      hasAudio: false,
      pointerOverlay: !!note.video_pointer_overlay,
      scope: String(note.video_scope || clipEntry.meta?.scope || 'window'),
      window: String(note.video_window || clipEntry.meta?.window || ''),
      durationMS: Number(note.video_duration_ms || clipEntry.meta?.durationMS || 0),
    };
    const form = new FormData();
    form.append('event_id', id);
    form.append('clip', reduced, 'floating-fit.webm');
    appendClipMetadataFC(form, clipMeta);
    await postForm('/api/session/feedback/clip', form);
    cacheClipForEventFC(id, { blob: reduced, ...clipMeta });
    showToastFC('Clip resized for ' + id + ' (' + formatMediaSizeFC(reduced.size) + ').');
    setStatus('Clip resized for ' + id + ' (' + formatMediaSizeFC(reduced.size) + ').');
    await previewPayloadFC();
  } catch (err) {
    showToastFC('Clip resize failed for ' + id + '.', true);
    setStatus('Clip resize failed: ' + err.message, true);
  } finally {
    fcClipResizeInFlight.delete(id);
    if (fcLatestPayloadPreviewData) {
      renderPayloadPreview(fcLatestPayloadPreviewData);
    }
  }
}

async function submitVideoNote() {
  if (inFlight) return;
  try {
    if (!fcVideoNoteClipRecorder) {
      if (recording) {
        setStatus('Stop the current recording before starting a video note.', true);
        return;
      }
      if (!ensureCaptureReadyFC('video')) return;
      if (lastState?.transcription_mode === 'remote' && !lastState?.allow_remote_stt) {
        throw new Error('Remote transcription is disabled by policy.');
      }
      await startVideoNoteCaptureFC();
      return;
    }
    await finalizeVideoNoteCaptureFC('manual stop');
  } catch (err) {
    setStatus('Video note failed: ' + err.message, true);
  }
}

async function finalizeVideoNoteCaptureFC(trigger = 'manual stop') {
  if (fcVideoNoteFinalizing) {
    return;
  }
  fcVideoNoteFinalizing = true;
  try {
    setBusy(true);
    const bundle = await finishVideoNoteCaptureFC();
    if (!bundle?.audioBlob || !bundle?.clip?.blob) {
      throw new Error('video note could not be recorded');
    }
    const form = new FormData();
    form.append('audio', bundle.audioBlob, 'floating-note.webm');
    const note = await postForm('/api/session/feedback/note', form);
    if (note && note.command_handled) {
      const cmd = String(note.command_result?.voice_command || 'voice_command');
      setStatus('Voice command handled: ' + cmd);
      await refreshState();
      return;
    }
    syncPreviewSessionStateFC(note?.session);
    const eventID = String(note?.event_id || '');
    if (!eventID) {
      throw new Error('video note was captured but the event could not be created');
    }
    const clipForm = new FormData();
    clipForm.append('event_id', eventID);
    clipForm.append('clip', bundle.clip.blob, 'floating-note-clip.webm');
    appendClipMetadataFC(clipForm, bundle.clip);
    try {
      const clipData = await postForm('/api/session/feedback/clip', clipForm);
      cacheClipForEventFC(eventID, bundle.clip);
      syncPreviewSessionStateFC(clipData?.session);
      setStatus('Video note captured: ' + eventID + '. Clip attached and ready for preview.');
    } catch (clipErr) {
      await refreshState().catch(() => {});
      setStatus('Video note captured: ' + eventID + '. Clip attach failed: ' + clipErr.message + '.', true);
      markPreviewStale('Preview needs refresh after your latest note.');
      return;
    }
    markPreviewStale('Preview needs refresh after your latest note.');
    await refreshState();
  } catch (err) {
    setStatus('Video note failed: ' + err.message + (trigger ? (' (' + trigger + ')') : ''), true);
    throw err;
  } finally {
    fcVideoNoteFinalizing = false;
    setBusy(false);
  }
}

async function captureScreenshotForNextNote() {
  if (inFlight || recording) return;
  try {
    setBusy(true);
    const blob = await captureScreenshotBlobFC();
    if (!blob) throw new Error('snapshot capture returned empty data');
    queuedScreenshotBlob = blob;
    renderScreenshotQueueState();
    setStatus('Snapshot captured. Snapshot actions will use it first.');
  } catch (err) {
    setStatus('Snapshot capture failed: ' + err.message, true);
  } finally {
    setBusy(false);
  }
}

async function previewPayloadFC() {
  if (inFlight) return;
  if (recording) {
    setStatus('Stop the current recording before previewing the request.', true);
    return;
  }
  try {
    if (hasTypedNoteDraftFC()) {
      await flushTypedNoteDraftFC('preview');
    }
    setBusy(true);
    if (!lastState?.session?.id) throw new Error('Start a session first.');
    await ensureFeedbackPresentFC();
    const provider = selectedProviderFC();
    const intentProfile = selectedDeliveryIntentProfileFC();
    const instructionText = selectedDeliveryInstructionTextFC();
    await approveSessionFC(true, 'preview');
    const data = await postJSON('/api/session/payload/preview', {
      provider,
      intent_profile: intentProfile,
      instruction_text: instructionText,
      allow_large_inline_media: !!fcSettings.allow_large_inline_media,
      redact_replay_values: !!fcPreviewDeliveryOptions.redactReplayValues,
      omit_video_clips: !!fcPreviewDeliveryOptions.omitVideoClips,
      omit_video_event_ids: fcPreviewDeliveryOptions.omitVideoEventIDs || []
    });
    renderPayloadPreview(data);
    const destination = providerDestinationLabelFC(provider);
    setStatus('Preview generated for ' + destination + ' with action "' + selectedDeliveryIntentLabelFC() + '".');
    setPreviewState('Preview ready for ' + destination + '. Review the captured notes, then send when you are ready.');
  } catch (err) {
    setPreviewState('Preview failed: ' + err.message, true);
    setStatus('Preview failed: ' + err.message, true);
  } finally {
    setBusy(false);
  }
}

async function submitSessionFC() {
  if (inFlight) return;
  if (recording) {
    setStatus('Stop the current recording before sending the request.', true);
    return;
  }
  try {
    if (hasTypedNoteDraftFC()) {
      await flushTypedNoteDraftFC('send');
    }
    setBusy(true);
    if (!lastState?.session?.id) throw new Error('Start a session first.');
    await ensureFeedbackPresentFC();
    const provider = selectedProviderFC();
    const intentProfile = selectedDeliveryIntentProfileFC();
    const intentLabel = selectedDeliveryIntentLabelFC();
    const instructionText = selectedDeliveryInstructionTextFC();
    setPreviewState('Preparing the request and queueing it for ' + providerDestinationLabelFC(provider) + '...');
    await approveSessionFC(true, 'send');
    const data = await postJSON('/api/session/submit', {
      provider,
      intent_profile: intentProfile,
      instruction_text: instructionText,
      allow_large_inline_media: !!fcSettings.allow_large_inline_media,
      redact_replay_values: !!fcPreviewDeliveryOptions.redactReplayValues,
      omit_video_clips: !!fcPreviewDeliveryOptions.omitVideoClips,
      omit_video_event_ids: fcPreviewDeliveryOptions.omitVideoEventIDs || []
    });
    const attemptID = String(data.attempt_id || '');
    const queuePos = Number(data.queue_position || 0);
    const status = String(data.status || 'queued');
    if (attemptID) fcWatchedSubmitAttemptIDs.add(attemptID);
    clearSubmittedPreviewFC();
    const destination = providerDestinationLabelFC(provider);
    setPreviewState('Sent to ' + destination + ' for "' + intentLabel + '". ' + (attemptID || 'Submission queued') + (queuePos > 0 ? (' • queue position ' + queuePos) : ''));
    setStatus('Submission queued: ' + (attemptID || 'attempt') + ' (' + status + ').');
    showToastFC('Request submitted to ' + destination + ' for "' + intentLabel + '"' + (attemptID ? (' as ' + attemptID) : '.'));
    await refreshState();
  } catch (err) {
    const msg = String(err.message || '');
    if (msg.includes('over the default send limit')) {
      setBusy(false);
      setPreviewState('Send blocked until you choose how to handle the large clip.', true);
      await previewPayloadFC();
      setStatus('Send blocked by the inline clip limit. Use “Make clip smaller to send,” “Use snapshot instead,” or allow large inline media.', true);
      return;
    } else {
      setPreviewState('Send failed: ' + msg, true);
      setStatus('Send failed: ' + msg, true);
    }
  } finally {
    setBusy(false);
  }
}

async function openLastLogFC() {
  if (inFlight || recording) return;
  try {
    setBusy(true);
    const data = await postJSON('/api/session/open-last-log', {});
    const path = String(data.path || '');
    setPreviewState(path ? ('Opened last log: ' + path) : 'Opened last log.');
    setStatus(path ? ('Opened last log: ' + path) : 'Opened last log.');
  } catch (err) {
    setPreviewState('Could not open the last log: ' + err.message, true);
    setStatus('Open last log failed: ' + err.message, true);
  } finally {
    setBusy(false);
  }
}

function clearQueuedScreenshot() {
  if (inFlight || recording) return;
  queuedScreenshotBlob = null;
  renderScreenshotQueueState();
  setStatus('Queued snapshot cleared.');
}

if (fcAudioModeEl) {
  fcAudioModeEl.addEventListener('change', () => {
    if (fcAudioModeEl.value !== 'push_to_talk') {
      setPTTFC(false);
    }
    scheduleAudioConfigApplyFC();
  });
}
if (fcAudioInputDeviceEl) {
  fcAudioInputDeviceEl.addEventListener('change', scheduleAudioConfigApplyFC);
}
if (fcAudioMutedEl) {
  fcAudioMutedEl.addEventListener('change', scheduleAudioConfigApplyFC);
}
if (fcAudioPausedEl) {
  fcAudioPausedEl.addEventListener('change', scheduleAudioConfigApplyFC);
}

window.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && composerSettingsModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeComposerSettingsModalFC();
    return;
  }
  if (e.key === 'Escape' && workspaceModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeWorkspaceModal();
    return;
  }
  if (e.key === 'Escape' && transcriptionRuntimeModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeTranscriptionRuntimeModal();
    return;
  }
  if (e.key === 'Escape' && audioControlsModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeAudioControlsModal();
    return;
  }
  if (e.key === 'Escape' && videoCaptureModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeVideoCaptureModalFC();
    return;
  }
  if (e.key === 'Escape' && codexRuntimeModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeCodexRuntimeModalFC();
    return;
  }
  if (e.code === 'Space' && fcAudioModeEl?.value === 'push_to_talk') {
    setPTTFC(true);
  }
});

window.addEventListener('keyup', (e) => {
  if (e.code === 'Space') {
    setPTTFC(false);
  }
});

window.addEventListener('blur', () => {
  setPTTFC(false);
});

if (fcSttModeEl) {
  fcSttModeEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttBaseURLEl) {
  fcSttBaseURLEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttBaseURLEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttModelEl) {
  fcSttModelEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttModelEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttFasterWhisperModelEl) {
  fcSttFasterWhisperModelEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttFasterWhisperModelEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttDeviceEl) {
  fcSttDeviceEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttDeviceEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttComputeTypeEl) {
  fcSttComputeTypeEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttComputeTypeEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttLanguageEl) {
  fcSttLanguageEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttLanguageEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttLocalCommandEl) {
  fcSttLocalCommandEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttLocalCommandEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcSttTimeoutSecondsEl) {
  fcSttTimeoutSecondsEl.addEventListener('input', scheduleTranscriptionRuntimeApplyFC);
  fcSttTimeoutSecondsEl.addEventListener('change', scheduleTranscriptionRuntimeApplyFC);
}
if (fcAgentDefaultProviderEl) {
  fcAgentDefaultProviderEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexCliCmdEl) {
  fcCodexCliCmdEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexCliCmdEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcClaudeCliCmdEl) {
  fcClaudeCliCmdEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcClaudeCliCmdEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcOpenCodeCliCmdEl) {
  fcOpenCodeCliCmdEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcOpenCodeCliCmdEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCliTimeoutSecondsEl) {
  fcCliTimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCliTimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcClaudeCliTimeoutSecondsEl) {
  fcClaudeCliTimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcClaudeCliTimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcOpenCodeCliTimeoutSecondsEl) {
  fcOpenCodeCliTimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcOpenCodeCliTimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexOutputDirEl) {
  fcCodexOutputDirEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexOutputDirEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcSubmitExecutionModeEl) {
  fcSubmitExecutionModeEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexSandboxEl) {
  fcCodexSandboxEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexApprovalEl) {
  fcCodexApprovalEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexSkipRepoCheckEl) {
  fcCodexSkipRepoCheckEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexProfileEl) {
  fcCodexProfileEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexProfileEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexModelEl) {
  fcCodexModelEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexReasoningEl) {
  fcCodexReasoningEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexAPIBaseURLEl) {
  fcCodexAPIBaseURLEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexAPIBaseURLEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexAPITimeoutSecondsEl) {
  fcCodexAPITimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexAPITimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexAPIOrgEl) {
  fcCodexAPIOrgEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexAPIOrgEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcCodexAPIProjectEl) {
  fcCodexAPIProjectEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcCodexAPIProjectEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcClaudeAPIBaseURLEl) {
  fcClaudeAPIBaseURLEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcClaudeAPIBaseURLEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcClaudeAPITimeoutSecondsEl) {
  fcClaudeAPITimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcClaudeAPITimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcClaudeAPIModelEl) {
  fcClaudeAPIModelEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcClaudeAPIModelEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcDeliveryIntentProfileEl) {
  fcDeliveryIntentProfileEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcDeliveryInstructionTextEl) {
  fcDeliveryInstructionTextEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcDeliveryInstructionTextEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcPostSubmitRebuildCmdEl) {
  fcPostSubmitRebuildCmdEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcPostSubmitRebuildCmdEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcPostSubmitVerifyCmdEl) {
  fcPostSubmitVerifyCmdEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcPostSubmitVerifyCmdEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}
if (fcPostSubmitTimeoutSecEl) {
  fcPostSubmitTimeoutSecEl.addEventListener('input', scheduleCodexRuntimeApplyFC);
  fcPostSubmitTimeoutSecEl.addEventListener('change', scheduleCodexRuntimeApplyFC);
}

setInterval(refreshState, 1500);
initFCPersistentSettings();
syncDeliveryIntentPromptTextFC(true);
syncFCSTTRuntimeModeUI();
syncFCCodexRuntimeModeUI();
syncComposerSettingsSummaryFC();
refreshState();
renderScreenshotQueueState();
renderTextEditorStateFC();
</script>
</body>
</html>`
