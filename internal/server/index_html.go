package server

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Knit Daemon</title>
  <link rel="icon" href="/favicon.ico" sizes="any" />
  <style>
    :root {
      --bg: #f7f4ee;
      --bg-accent: #fffaf3;
      --panel: rgba(255, 255, 255, 0.92);
      --panel-solid: #ffffff;
      --text: #192434;
      --muted: #6a7383;
      --accent: #1c7c74;
      --accent-strong: #155b56;
      --accent-soft: #e4f4f1;
      --warn: #b97c1f;
      --danger: #c34f4f;
      --danger-soft: #fbe7e5;
      --good: #1f8f63;
      --good-soft: #e7f7ef;
      --line: #e7dfd4;
      --line-strong: #d8cdbd;
      --shadow: 0 18px 55px rgba(27, 39, 59, 0.08);
      --guide-width: 360px;
    }
    html[data-theme="dark"] {
      --bg: #0f1722;
      --bg-accent: #121d2b;
      --panel: rgba(18, 28, 42, 0.9);
      --panel-solid: #132033;
      --text: #eef4fb;
      --muted: #aab7ca;
      --accent: #62c6bc;
      --accent-strong: #8de0d7;
      --accent-soft: rgba(98, 198, 188, 0.16);
      --warn: #f0b35d;
      --danger: #ff8f8f;
      --danger-soft: rgba(195, 79, 79, 0.14);
      --good: #72d8a9;
      --good-soft: rgba(31, 143, 99, 0.16);
      --line: rgba(157, 178, 205, 0.18);
      --line-strong: rgba(157, 178, 205, 0.28);
      --shadow: 0 24px 60px rgba(3, 8, 15, 0.42);
    }
    *, *::before, *::after { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Inter", "SF Pro Text", "Segoe UI", sans-serif;
      background:
        radial-gradient(1200px 600px at 10% -10%, rgba(28, 124, 116, 0.10), transparent 55%),
        radial-gradient(900px 500px at 90% 0%, rgba(219, 188, 137, 0.16), transparent 45%),
        linear-gradient(180deg, var(--bg-accent) 0%, var(--bg) 100%);
      color: var(--text);
      line-height: 1.5;
    }
    .wrap {
      max-width: 1120px;
      margin: 0 auto;
      padding: 2rem 1.25rem 3rem;
    }
    .theme-toggle {
      width: 3rem;
      height: 3rem;
      padding: 0;
      border-radius: 999px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-size: 1.1rem;
      background: var(--panel-solid);
      box-shadow: var(--shadow);
    }
    .hero-copy-block {
      display: grid;
      gap: .9rem;
    }
    .hero-brand {
      display: inline-flex;
      align-items: center;
      gap: .85rem;
    }
    .hero-brand-mark {
      width: 3.6rem;
      height: 3.6rem;
      border-radius: 18px;
      display: block;
      background: var(--panel-solid);
      border: 1px solid var(--line);
      box-shadow: var(--shadow);
      object-fit: cover;
    }
    .hero-brand-copy {
      display: grid;
      gap: .15rem;
    }
    .hero-brand-name {
      font-size: 1.18rem;
      font-weight: 800;
      letter-spacing: -0.02em;
      color: var(--text);
    }
    .hero-brand-tagline {
      font-size: .9rem;
      color: var(--muted);
    }
    .hero-topline {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 1rem;
    }
    body.guide-open .wrap { margin-right: calc(var(--guide-width) + 1.5rem); }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 1.35rem;
      margin-bottom: 1rem;
      box-shadow: var(--shadow);
      backdrop-filter: blur(16px);
    }
    .hero {
      padding: 1.75rem;
      background:
        linear-gradient(135deg, var(--panel-solid), var(--panel)),
        var(--panel);
    }
    .hero-grid {
      display: grid;
      grid-template-columns: minmax(0, 1.8fr) minmax(320px, 1fr);
      gap: 1.25rem;
      align-items: start;
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: .45rem;
      font-size: .8rem;
      font-weight: 700;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent-strong);
      background: var(--accent-soft);
      border-radius: 999px;
      padding: .35rem .7rem;
      margin-bottom: 1rem;
    }
    h1, h2, h3, h4, p { margin: 0; }
    h1 {
      font-size: clamp(2rem, 4vw, 3.3rem);
      line-height: 1.05;
      letter-spacing: -0.03em;
      margin-bottom: .9rem;
      max-width: 12ch;
    }
    .hero-copy {
      max-width: 62ch;
      font-size: 1.02rem;
      color: var(--muted);
    }
    .metric-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: .85rem;
      align-content: start;
    }
    .metric-card {
      background: var(--panel-solid);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 1rem;
      min-height: 0;
      aspect-ratio: 1 / 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
    .metric-label {
      font-size: .82rem;
      color: var(--muted);
      font-weight: 600;
    }
    .metric-value {
      font-size: 1.05rem;
      font-weight: 700;
      color: var(--text);
      word-break: break-word;
    }
    .flow-stack { display: grid; gap: 1rem; }
    .step-card { padding: 1.5rem; }
    .step-header {
      display: flex;
      flex-wrap: wrap;
      gap: .9rem;
      align-items: flex-start;
      margin-bottom: 1rem;
    }
    .step-number {
      flex: 0 0 auto;
      width: 2.3rem;
      height: 2.3rem;
      border-radius: 999px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: var(--accent-soft);
      color: var(--accent-strong);
      font-weight: 800;
    }
    .step-title {
      font-size: 1.35rem;
      font-weight: 750;
      letter-spacing: -0.02em;
      margin-bottom: .2rem;
    }
    .step-copy {
      color: var(--muted);
      max-width: 68ch;
    }
    .field-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: .9rem;
      margin: 1rem 0;
    }
    .field {
      display: flex;
      flex-direction: column;
      gap: .38rem;
    }
    .field-label {
      font-size: .9rem;
      font-weight: 650;
      color: var(--text);
    }
    button {
      appearance: none;
      background: var(--panel-solid);
      color: var(--text);
      border: 1px solid var(--line-strong);
      border-radius: 14px;
      padding: .72rem 1rem;
      cursor: pointer;
      font-weight: 650;
      font-size: .95rem;
      transition: transform .14s ease, border-color .14s ease, box-shadow .14s ease, background .14s ease;
      box-shadow: none;
    }
    button:hover {
      border-color: var(--accent);
      transform: translateY(-1px);
      box-shadow: 0 10px 24px rgba(25, 36, 52, 0.08);
    }
    button:focus-visible,
    input:focus-visible,
    textarea:focus-visible,
    select:focus-visible,
    summary:focus-visible {
      outline: 3px solid rgba(28, 124, 116, 0.22);
      outline-offset: 2px;
    }
    button:disabled {
      opacity: .55;
      cursor: not-allowed;
      transform: none;
      box-shadow: none;
    }
    .primary,
    .ok {
      background: var(--accent);
      color: #fff;
      border-color: var(--accent);
    }
    .primary:hover,
    .ok:hover {
      border-color: var(--accent-strong);
      background: var(--accent-strong);
    }
    .primary-action {
      min-height: 60px;
      padding: 1rem 1.75rem;
      font-size: 1.1rem;
      line-height: 1.2;
      font-weight: 760;
      letter-spacing: .01em;
      color: #fff;
      background: #072b29;
      border-color: #072b29;
      box-shadow: 0 18px 38px rgba(7, 43, 41, 0.28);
    }
    .primary-action:hover {
      background: #041b19;
      border-color: #041b19;
      box-shadow: 0 20px 40px rgba(4, 27, 25, 0.32);
    }
    .primary-action:focus-visible {
      outline: 3px solid rgba(7, 43, 41, 0.28);
    }
    .secondary {
      background: var(--panel-solid);
      color: var(--text);
    }
    .danger {
      color: var(--danger);
      border-color: rgba(195, 79, 79, 0.35);
      background: var(--danger-soft);
    }
    .ghost {
      background: transparent;
      color: var(--muted);
    }
    .icon-btn {
      min-width: auto;
      padding-inline: .95rem;
    }
    input, textarea, select {
      width: 100%;
      max-width: 100%;
      background: var(--panel-solid);
      color: var(--text);
      border: 1px solid var(--line-strong);
      border-radius: 16px;
      padding: .82rem .9rem;
      margin: 0;
      font: inherit;
    }
    textarea {
      min-height: 144px;
      resize: vertical;
    }
    .row {
      display: flex;
      flex-wrap: wrap;
      gap: .65rem;
      align-items: center;
    }
    .action-row {
      display: flex;
      flex-wrap: wrap;
      gap: .75rem;
      margin-top: 1rem;
    }
    .sub-actions {
      display: flex;
      flex-wrap: wrap;
      gap: .6rem;
      margin-top: .8rem;
    }
    .helper {
      margin-top: .85rem;
      color: var(--muted);
      font-size: .93rem;
    }
    .helper a {
      color: var(--accent-strong);
      font-weight: 700;
      text-decoration: none;
      border-bottom: 1px solid transparent;
    }
    .helper a:hover,
    .helper a:focus-visible {
      border-bottom-color: currentColor;
      outline: none;
    }
    .chip-row {
      display: flex;
      flex-wrap: wrap;
      gap: .7rem;
      margin-top: .85rem;
    }
    .status-chip,
    .status-card {
      display: inline-flex;
      align-items: center;
      gap: .5rem;
      border-radius: 999px;
      border: 1px solid var(--line);
      background: var(--panel-solid);
      padding: .55rem .85rem;
      color: var(--muted);
      font-weight: 600;
    }
    .status-stack {
      display: grid;
      gap: .75rem;
      margin-top: 1rem;
    }
    .status-card {
      border-radius: 18px;
      padding: .85rem 1rem;
      min-height: 60px;
      align-items: flex-start;
      flex-direction: column;
    }
    .status-card strong {
      color: var(--text);
      font-size: .84rem;
      letter-spacing: .02em;
      text-transform: uppercase;
    }
    .status { font-weight: 700; }
    .step-header-main {
      display: grid;
      gap: .2rem;
      flex: 1 1 auto;
      min-width: 0;
    }
    .step-header-status {
      flex: 0 0 auto;
      min-width: 220px;
      justify-content: center;
      text-align: center;
    }
    .step-header-toolbar {
      display: flex;
      flex: 0 0 auto;
      flex-wrap: wrap;
      gap: .75rem;
      margin-left: auto;
      justify-content: flex-end;
    }
    .status-list {
      margin: .5rem 0 0;
      padding-left: 1rem;
      color: var(--text);
      display: grid;
      gap: .28rem;
    }
    .status-list li {
      line-height: 1.4;
    }
    .grid { display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); }
    .hidden,
    [hidden] { display:none !important; }
    code {
      background: var(--bg-accent);
      color: var(--text);
      padding: .18rem .38rem;
      border-radius: 8px;
      font-size: .92em;
    }
    details.panel > summary {
      cursor: pointer;
      font-weight: 700;
      list-style-position: inside;
      outline: none;
    }
    details.panel[open] > summary {
      margin-bottom: .9rem;
    }
    .spinner {
      display:inline-block;
      width: 12px;
      height: 12px;
      border: 2px solid rgba(28, 124, 116, 0.22);
      border-top-color: var(--accent);
      border-radius: 50%;
      animation: spin .8s linear infinite;
      vertical-align: middle;
      margin-right: .35rem;
    }
    .meter {
      width: 100%;
      height: 12px;
      border-radius: 8px;
      border: 1px solid var(--line);
      background: var(--bg-accent);
      overflow: hidden;
      margin-top: .35rem;
    }
    .meter-fill {
      width: 0%;
      height: 100%;
      background: linear-gradient(90deg, #4fd1c5 0%, #48bb78 50%, #f6ad55 80%, #f56565 100%);
      transition: width .08s linear;
    }
    .modal-overlay {
      position: fixed;
      inset: 0;
      background: rgba(19, 27, 39, 0.32);
      display: none;
      align-items: center;
      justify-content: center;
      z-index: 9999;
      padding: 1rem;
    }
    .modal-overlay.open {
      display: flex;
    }
    .modal-card {
      width: min(820px, 100%);
      max-height: 90vh;
      overflow: auto;
      background: var(--panel-solid);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 1.25rem;
      box-shadow: 0 28px 80px rgba(22, 31, 48, 0.16);
    }
    .capture-guide-sidebar {
      position: fixed;
      top: 0;
      right: 0;
      width: min(var(--guide-width), 96vw);
      height: 100vh;
      overflow-y: auto;
      overflow-x: hidden;
      background: var(--panel-solid);
      border-left: 1px solid var(--line);
      box-shadow: -12px 0 32px rgba(22, 31, 48, 0.08);
      z-index: 2000;
      padding: 1rem 1.2rem 1.2rem;
    }
    .capture-guide-sidebar h3 { margin: .2rem 0 .6rem 0; }
    .capture-guide-header {
      position: relative;
      padding-top: 1.1rem;
      margin-bottom: .4rem;
    }
    .capture-guide-close {
      position: absolute;
      top: 0;
      right: 0;
      width: 1.7rem;
      height: 1.7rem;
      min-width: 1.7rem;
      min-height: 1.7rem;
      padding: 0;
      border-radius: 999px;
      font-size: .9rem;
    }
    .capture-guide-toggle {
      position: fixed;
      top: 14px;
      right: 68px;
      z-index: 2100;
      appearance: none;
      background: transparent;
      border: none;
      box-shadow: none;
      padding: 0;
      width: auto;
      height: auto;
      border-radius: 0;
      color: var(--muted);
      font-size: 1.25rem;
      line-height: 1;
    }
    .capture-guide-toggle:hover {
      color: var(--accent-strong);
      transform: none;
      box-shadow: none;
      border-color: transparent;
      background: transparent;
    }
    .capture-guide-toggle:focus-visible {
      outline: 3px solid rgba(28, 124, 116, 0.22);
      outline-offset: 6px;
    }
    .toast {
      position: fixed;
      left: 50%;
      bottom: 1.25rem;
      transform: translateX(-50%) translateY(12px);
      background: rgba(25, 36, 52, 0.92);
      color: #fff;
      border-radius: 999px;
      padding: .78rem 1.05rem;
      font-size: .9rem;
      font-weight: 650;
      box-shadow: 0 14px 30px rgba(15, 23, 34, 0.22);
      opacity: 0;
      pointer-events: none;
      transition: opacity .18s ease, transform .18s ease;
      z-index: 10030;
    }
    .toast.visible {
      opacity: 1;
      transform: translateX(-50%) translateY(0);
    }
    .toast.error {
      background: rgba(195, 79, 79, 0.94);
    }
    .console-card {
      border: 1px solid var(--line);
      border-radius: 18px;
      background: var(--panel);
      padding: .3rem .3rem .35rem;
    }
    .console-card summary {
      padding: .6rem .7rem;
      font-size: .95rem;
    }
    .request-preview {
      display: grid;
      gap: .85rem;
      margin-top: .25rem;
    }
    .preview-summary-card,
    .preview-note-card,
    .preview-warning-card {
      background: var(--bg-accent);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 1rem;
    }
    .preview-note-card {
      background: var(--panel-solid);
    }
    .preview-warning-card {
      background: var(--danger-soft);
    }
    .preview-kicker {
      font-size: .78rem;
      font-weight: 700;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent-strong);
      margin-bottom: .35rem;
    }
    .preview-summary-line,
    .preview-note-header {
      display: flex;
      justify-content: space-between;
      gap: .75rem;
      align-items: center;
      flex-wrap: wrap;
    }
    .preview-note-header strong,
    .preview-summary-line strong {
      font-size: 1rem;
    }
    .preview-note-meta {
      display: flex;
      gap: .45rem;
      flex-wrap: wrap;
      margin-top: .45rem;
      color: var(--muted);
      font-size: .86rem;
    }
    .preview-note-meta span {
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: .2rem .55rem;
      background: rgba(255,255,255,0.65);
    }
    html[data-theme="dark"] .preview-note-meta span {
      background: rgba(11, 19, 30, 0.45);
    }
    .preview-note-text {
      margin-top: .65rem;
      white-space: pre-wrap;
      line-height: 1.55;
    }
    .preview-media-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: .8rem;
      margin-top: .8rem;
    }
    .preview-media {
      margin: 0;
      display: grid;
      gap: .45rem;
    }
    .preview-media figcaption {
      font-size: .84rem;
      font-weight: 650;
      color: var(--muted);
    }
    .preview-media img,
    .preview-media video,
    .preview-media audio {
      width: 100%;
      border-radius: 14px;
      border: 1px solid var(--line);
      background: var(--panel-solid);
    }
    .preview-media audio {
      min-height: 46px;
    }
    pre {
      background: var(--bg-accent);
      color: var(--text);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 1rem;
      max-width: 100%;
      min-width: 0;
      overflow: auto;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
      margin: .25rem 0 0;
      font-size: .9rem;
      line-height: 1.55;
    }
    .empty-tone {
      color: var(--muted);
      font-size: .94rem;
    }
    .modal-section {
      padding: 1rem 0 0;
      border-top: 1px solid var(--line);
      margin-top: 1rem;
    }
    .modal-section:first-of-type {
      border-top: 0;
      margin-top: .5rem;
      padding-top: 0;
    }
    .modal-section h4 {
      margin-bottom: .3rem;
      font-size: 1rem;
    }
    .modal-help {
      color: var(--muted);
      font-size: .92rem;
      margin-bottom: .7rem;
    }
    .runtime-section {
      padding-top: 1rem;
      margin-top: 1rem;
      border-top: 1px solid var(--line);
    }
    .runtime-section:first-of-type {
      padding-top: 0;
      margin-top: .35rem;
      border-top: 0;
    }
    .runtime-section h4 {
      margin-bottom: .25rem;
    }
    .runtime-inline-note {
      color: var(--muted);
      font-size: .88rem;
      margin: .35rem 0 0;
    }
    .toolbar-button {
      display: inline-flex;
      align-items: center;
      gap: .48rem;
    }
    .capture-option-actions {
      justify-content: flex-end;
      align-items: center;
    }
    .capture-option-grid {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 1rem;
      margin-top: 1rem;
      align-items: start;
    }
    .capture-option-card {
      background: rgba(255,255,255,0.04);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 1rem;
      display: grid;
      gap: .7rem;
    }
    .main-ui-action-grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: .65rem;
      margin-top: .15rem;
    }
    .main-ui-icon-button {
      width: 100%;
      min-width: 0;
      min-height: 3rem;
      justify-content: center;
      padding: .72rem;
      font-size: 1.05rem;
      gap: 0;
    }
    .capture-option-kicker {
      font-size: .72rem;
      font-weight: 800;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--muted);
    }
    .capture-option-title {
      font-size: 1.05rem;
      font-weight: 800;
      margin: 0;
    }
    .capture-option-copy {
      margin: 0;
      color: var(--muted);
      line-height: 1.6;
    }
    .capture-option-code {
      margin: 0;
      display: block;
      width: fit-content;
      max-width: 100%;
    }
    @media (max-width: 1200px) {
      body.guide-open .wrap { margin-right: 1rem; }
    }
    @media (max-width: 860px) {
      .hero-grid,
      .field-grid {
        grid-template-columns: 1fr;
      }
      .capture-option-grid {
        grid-template-columns: 1fr;
      }
      .capture-option-card.option-main-ui {
        grid-column: auto;
      }
      .main-ui-action-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
      .wrap {
        padding: 1.2rem .9rem 2rem;
      }
      .panel,
      .hero,
      .step-card {
        border-radius: 20px;
        padding: 1.1rem;
      }
      .action-row,
      .sub-actions {
        flex-direction: column;
        align-items: stretch;
      }
      .step-header-toolbar {
        width: 100%;
        margin-left: 0;
      }
      button {
        width: 100%;
        justify-content: center;
      }
    }
    @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
  </style>
</head>
<body class="guide-open">
  <div class="wrap">
    <section class="hero panel">
      <div class="hero-grid">
        <div class="hero-copy-block">
          <div class="hero-brand">
            <img class="hero-brand-mark" src="/docs/assets/knit-mark.png" alt="Knit logo" />
            <div class="hero-brand-copy">
              <span class="hero-brand-name">Knit</span>
              <span class="hero-brand-tagline">Local-first multimodal AI feedback runtime</span>
            </div>
          </div>
          <div class="hero-topline">
            <h1>Capture what should change. Tell your agent.</h1>
            <button id="themeToggleBtn" class="icon-btn theme-toggle" onclick="toggleTheme()" title="Switch to dark theme" aria-label="Switch to dark theme">☾</button>
          </div>
          <p class="hero-copy">Start a review session, connect to a webpage, use the composer to point at the interface, record what needs to change, and hand off a structured request to your coding agent.</p>
        </div>
        <div class="metric-grid" aria-label="Session summary">
          <div class="metric-card">
            <span class="metric-label">Capture status</span>
            <span class="metric-value status" id="capture">unknown</span>
          </div>
          <div class="metric-card">
            <span class="metric-label">Current session</span>
            <span class="metric-value status" id="sessionId">none</span>
          </div>
        </div>
      </div>
    </section>

    <div class="flow-stack">
      <section class="panel step-card">
        <div class="step-header">
          <div class="step-number">1</div>
          <div class="step-header-main">
            <div class="step-title">Connect this review</div>
            <p class="step-copy">Start the review, then connect the browser companion. Knit uses a safe default label first and fills in page title and URL automatically once the browser is attached.</p>
          </div>
        </div>
        <div class="action-row">
          <button id="sessionPlayBtn" class="primary toolbar-button" onclick="startSession()" title="Start Session" aria-label="Start Session">▶ <span>Start review</span></button>
          <button id="sessionPauseResumeBtn" class="secondary toolbar-button" onclick="togglePauseResume()" title="Pause Capture" aria-label="Pause Capture" disabled hidden>⏸ <span>Pause</span></button>
          <button id="sessionStopBtn" class="secondary toolbar-button" onclick="stopSession()" title="Stop Session" aria-label="Stop Session" disabled hidden>⏹ <span>Stop</span></button>
          <button class="danger toolbar-button" onclick="deleteSession()" title="Delete Session" aria-label="Delete Session">🗑️ <span>Delete session</span></button>
          <button class="danger toolbar-button" onclick="purgeAllData()" title="Purge All Data" aria-label="Purge All Data">🗑️🗑️ <span>Delete all data</span></button>
        </div>
        <p class="helper">Delete session removes the current review session. Delete all data clears every saved artifact across sessions.</p>
        <details class="console-card" style="margin-top:1rem;">
          <summary>Advanced session details</summary>
          <p class="helper" style="margin:.25rem .7rem .6rem;">Use this only if you want to override the default review label or enter a URL manually before the browser companion attaches.</p>
          <div class="field-grid" style="padding:0 .7rem .7rem;">
            <label class="field">
              <span class="field-label">Review label</span>
              <input id="targetWindow" placeholder="Browser Review" value="" />
            </label>
            <label class="field">
              <span class="field-label">Route or URL</span>
              <input id="targetURL" placeholder="https://localhost:3000" value="" />
            </label>
          </div>
        </details>
      </section>

      <section class="panel step-card">
        <div class="step-header">
          <div class="step-number">2</div>
          <div class="step-header-main">
            <div class="step-title">Capture, review, and send</div>
            <p class="step-copy">Choose how you want to capture the request. Docs and settings stay separate so you can pick a path without digging through setup controls.</p>
            <p id="captureAgentNotice" class="helper">Current coding agent: <strong>codex_cli</strong>. Change it in Settings → Agent or in <code>knit.toml</code>.</p>
          </div>
        </div>
        <div id="captureOptionActions" class="action-row capture-option-actions">
          <button id="openDocsLibraryBtn" class="secondary toolbar-button" onclick="openDocsBrowser()" title="Open docs library in a new tab">📚 <span>Docs</span></button>
          <button class="secondary toolbar-button" onclick="openCaptureSettingsModal()" title="Capture settings" aria-label="Capture settings">⚙️ <span>Settings</span></button>
        </div>
        <div id="captureOptionGrid" class="capture-option-grid">
          <section id="captureOptionExtension" class="capture-option-card">
            <div class="capture-option-kicker">Easy</div>
            <h3 class="capture-option-title">Chrome Extension</h3>
            <p class="capture-option-copy">Pair the browser extension and work from the page you are reviewing without bouncing back to the main UI.</p>
            <div style="display:grid;gap:.45rem;justify-items:start;">
              <button class="secondary toolbar-button" onclick="startExtensionPairing()" title="Generate a browser extension pairing code">⌁ <span>Generate extension token</span></button>
              <button class="secondary toolbar-button" onclick="openDocsBrowser('GETTING_STARTED.md')" title="Open extension install guide">↗ <span>Install guide</span></button>
              <code id="extensionPairingCodeState" class="capture-option-code">No active pairing code.</code>
            </div>
          </section>
          <section id="captureOptionComposer" class="capture-option-card">
            <div class="capture-option-kicker">Intermediate</div>
            <h3 class="capture-option-title">Popout Composer</h3>
            <p class="capture-option-copy">Open the popout composer to stay close to the browser surface while still using Knit’s capture controls.</p>
            <div style="display:flex;flex-wrap:wrap;gap:.65rem;">
              <button class="primary toolbar-button" onclick="openFloatingComposerPopup()" title="Open popout composer">✦ <span>Open popout composer</span></button>
            </div>
          </section>
          <section id="captureOptionMainUI" class="capture-option-card option-main-ui">
            <div class="capture-option-kicker">Main UI</div>
            <h3 class="capture-option-title">Main UI Interface</h3>
            <p class="capture-option-copy">If you want to stay here, write the note directly in the main page, record media from here, then preview and submit without opening another surface.</p>
            <div class="field-grid" style="margin-top:.1rem;">
              <div class="field" style="grid-column:1 / -1;">
                <span class="field-label">Written note</span>
                <textarea id="transcript" placeholder="Describe the change in plain language. If you prefer, leave this blank and record an audio note instead."></textarea>
              </div>
            </div>
            <div class="main-ui-action-grid" aria-label="Main UI capture actions">
              <button id="audioNoteBtn" class="secondary icon-btn toolbar-button main-ui-icon-button" onclick="submitAudioNote()" title="Record audio note" aria-label="Record audio note">🎙️</button>
              <button id="videoNoteBtn" class="secondary icon-btn toolbar-button main-ui-icon-button" onclick="submitVideoNote()" title="Record video note" aria-label="Record video note">🎥</button>
              <button id="previewBtn" class="secondary icon-btn toolbar-button main-ui-icon-button" onclick="previewPayload()" title="Preview request" aria-label="Preview request">👁</button>
              <button id="submitBtn" class="secondary icon-btn toolbar-button main-ui-icon-button" onclick="submitSession()" title="Submit request" aria-label="Submit request">⇪</button>
            </div>
          </section>
        </div>
        <div id="noteStatus" class="helper">Nothing captured yet. Open the composer, or write a note here and use Preview request or Submit when you are ready.</div>
        <details class="console-card" style="margin-top:1rem;" open>
          <summary>Structured request preview</summary>
          <div id="payloadPreview" class="request-preview empty-tone">Preview the request to inspect what Knit will send.</div>
        </details>
      </section>

      <section class="panel step-card">
        <div class="step-header">
          <div class="step-number">3</div>
          <div class="step-header-main">
            <div class="step-title">Queue and delivery</div>
            <p class="step-copy">This step shows what is queued, what is running, live agent output, and your recent runs.</p>
          </div>
          <div id="deliveryBadge" class="status-chip step-header-status">Waiting for your first note.</div>
        </div>
        <div class="status-stack">
          <div class="status-card"><strong>Queue</strong><div id="queueState" class="empty-tone">No queued submissions.</div></div>
          <div class="status-card"><strong>Current run</strong><div id="submitState" class="empty-tone">No active run.</div></div>
        </div>
        <div class="flow-stack" style="margin-top:1rem;">
          <details class="console-card" open>
            <summary>Live agent output</summary>
            <div class="flow-stack" style="margin-top:.65rem;">
              <div>
                <div class="helper" style="margin-bottom:.35rem;"><strong>Work log</strong></div>
                <pre id="liveSubmitLog" style="max-height:180px;overflow:auto;">No live work log yet. Work activity appears here after the adapter starts writing logs.</pre>
              </div>
              <div>
                <div class="helper" style="margin-bottom:.35rem;"><strong>Agent commentary</strong></div>
                <pre id="liveSubmitCommentary" style="max-height:120px;overflow:auto;">No agent commentary yet. Plain-language progress updates appear here when the agent explains what it is doing.</pre>
              </div>
              <div class="helper">Raw prompt/payload details stay available in the full execution log via <strong>Open log</strong>.</div>
            </div>
          </details>
          <details class="console-card">
            <summary>Recent runs</summary>
            <div id="submitResult" class="empty-tone">No runs yet. Recent agent requests will appear here.</div>
          </details>
        </div>
      </section>

      <details class="panel">
        <summary>Environment profiles</summary>
        <p class="helper" style="margin-top:.35rem;">Use this only when you want to switch between personal, managed, or high-security operating modes.</p>
        <div class="row" style="margin-top:.8rem;">
          <div><strong>Config lock:</strong> <span class="status" id="configLockStatus">unknown</span></div>
        </div>
        <div class="row" style="margin-top:.8rem;">
          <select id="configProfile">
            <option value="personal_local_dev">personal_local_dev</option>
            <option value="enterprise_managed_workstation">enterprise_managed_workstation</option>
            <option value="high_security_restricted_mode">high_security_restricted_mode</option>
          </select>
          <button class="secondary" onclick="exportConfig()" title="Export profile">Export profile</button>
          <button id="save" data-testid="settings-save" class="primary primary-action" onclick="applyProfile()" title="Save settings">Save settings</button>
        </div>
        <pre id="configExport">Profile details will appear here after export.</pre>
      </details>

      <details class="panel">
        <summary>Technical state</summary>
        <p class="helper" style="margin-top:.35rem;">This is intentionally tucked away. Most people should not need it during normal review work.</p>
        <pre id="state">loading...</pre>
      </details>

      <details class="panel">
        <summary>Safety and retention</summary>
        <p class="helper" style="margin-top:.35rem;">Review capture scope, retention, and redaction policy details here.</p>
        <pre id="capturePolicy">loading policy...</pre>
      </details>
    </div>
  </div>

  <button id="guideInfoBtn" class="capture-guide-toggle hidden" onclick="openCaptureGuideSidebar()" title="Open Capture Guide" aria-label="Open Capture Guide">ℹ️</button>
  <div id="appToast" class="toast" role="status" aria-live="polite"></div>
  <aside id="captureGuideSidebar" class="capture-guide-sidebar">
    <div class="capture-guide-header">
      <button class="danger icon-btn capture-guide-close" onclick="closeCaptureGuideSidebar()" title="Close Capture Guide" aria-label="Close Capture Guide">✕</button>
      <h3>Capture Guide (Step By Step)</h3>
    </div>
    <div style="font-size:.95rem;color:var(--muted);line-height:1.65;">
      1. Start the review session.<br/>
      2. Click <strong>Connect browser</strong>, then run the copied snippet in the browser tab you are reviewing.<br/>
      3. Open <strong>Video</strong> and enable visual capture for the same browser window or tab.<br/>
      4. Open <strong>Audio</strong> and confirm your mic mode and input. Changes apply automatically.<br/>
      5. Open the <strong>popout composer</strong> and record audio notes while you stay on the page being reviewed.<br/>
      6. Come back here to preview the request or submit it to your coding agent.
    </div>
    <pre id="captureGuideStatus">capture readiness: loading...</pre>
    <div id="platformRuntimeState" style="font-size:.9rem;color:var(--muted);margin-top:.45rem;">platform runtime: loading...</div>
    <div id="composerSupportState" style="font-size:.9rem;color:var(--muted);margin-top:.45rem;">composer popup status: loading...</div>
  </aside>

  <div id="captureSettingsModal" class="modal-overlay hidden" onclick="onCaptureSettingsModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <h3 style="margin:.1rem 0;">Capture Settings</h3>
        <button class="danger" onclick="closeCaptureSettingsModal()" title="Close">Close</button>
      </div>
      <div style="font-size:.9rem;opacity:.9;margin:.2rem 0 .6rem 0;">
        Keep the main composer simple. Browser connection, capture configuration, and agent settings live behind this gear menu.
      </div>
      <div class="action-row" style="margin-top:0;">
        <button class="secondary toolbar-button" onclick="openWorkspaceModal()" title="Workspace" aria-label="Workspace">📁 <span>Workspace</span></button>
        <button class="secondary toolbar-button" onclick="copyCompanionSnippet()" title="Copy Browser Companion Snippet" aria-label="Copy Browser Companion Snippet">🔗 <span>Connect browser</span></button>
        <button class="secondary toolbar-button" onclick="openVideoCaptureModal()" title="Video Capture Settings" aria-label="Video Capture Settings">🎥 <span>Video</span></button>
        <button class="secondary toolbar-button" onclick="openAudioControlsModal()" title="Audio Controls" aria-label="Audio Controls">🎙️ <span>Audio</span></button>
        <button class="secondary toolbar-button" onclick="openCodexRuntimeModal()" title="Agent Runtime" aria-label="Agent Runtime">🤖 <span>Agent</span></button>
        <button id="openDocsLibrarySettingsBtn" class="secondary toolbar-button" onclick="openDocsBrowser()" title="Open docs library in a new tab">📚 <span>Docs</span></button>
        <button id="openLogBtn" class="secondary toolbar-button" onclick="openLastLog()" title="Open last log">↗ <span>Open log</span></button>
      </div>
      <div class="field-grid" style="margin-top:1rem;">
        <div class="field">
          <span class="field-label">More capture options</span>
          <div class="row">
            <label for="reviewMode">Review mode</label>
            <select id="reviewMode">
              <option value="">general</option>
              <option value="accessibility">accessibility</option>
            </select>
            <button class="secondary" onclick="applyReviewMode()" title="Apply review mode">Apply</button>
          </div>
        </div>
        <div class="field">
          <span class="field-label">Replay and pointer tools</span>
          <label title="Include typed form values in the replay bundle for this session. Enabled by default for new sessions.">
            <input id="captureInputValuesToggle" type="checkbox" onchange="toggleReplayValueCapture()" />
            Capture typed values for replay
          </label>
          <label><input id="laserModeEnabled" type="checkbox" /> Laser pointer mode</label>
        </div>
        <div class="field">
          <span class="field-label">Browser Extension</span>
          <div class="small" style="margin-bottom:.45rem;">Generate the extension token from Step 2 on the main page, then use this section to confirm or revoke paired popup access.</div>
          <div id="extensionPairingList" class="small" style="margin-top:.55rem;">No paired extensions.</div>
        </div>
      </div>
      <div class="action-row">
        <button class="secondary" onclick="startVoiceCommands()" title="Start voice commands">Start voice commands</button>
        <button class="secondary" onclick="stopVoiceCommands()" title="Stop voice commands">Stop voice commands</button>
      </div>
    </div>
  </div>

  <div id="workspaceModal" class="modal-overlay hidden" onclick="onWorkspaceModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <h3 style="margin:.1rem 0;">Workspace (Required)</h3>
        <button id="workspaceModalCloseBtn" class="danger" onclick="closeWorkspaceModal()" title="Close">Close</button>
      </div>
      <div style="font-size:.9rem;opacity:.9;margin:.2rem 0 .4rem 0;">
        Select the repository/workspace directory for coding-agent submissions.
      </div>
      <div class="row">
        <input id="workspaceDir" placeholder="/abs/path/repo" style="min-width:420px;" />
        <button onclick="applyWorkspaceDir()" title="Apply workspace">Apply Workspace</button>
        <button class="ok" onclick="pickWorkspaceDir()" title="Choose folder">Choose Folder...</button>
      </div>
      <div id="workspaceModalStatus" style="font-size:.9rem;opacity:.9;">Workspace selection required.</div>
      <pre id="workspaceBrowserState">workspace browser not loaded</pre>
    </div>
  </div>

  <div id="audioControlsModal" class="modal-overlay hidden" onclick="onAudioControlsModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <h3 style="margin:.1rem 0;">Audio Controls</h3>
        <div class="row">
          <button id="openTranscriptionFromAudioBtn" class="icon-btn" onclick="openTranscriptionRuntimeFromAudioModal()" title="Transcription Runtime" aria-label="Transcription Runtime">⚙️</button>
          <button class="danger" onclick="closeAudioControlsModal()" title="Close">Close</button>
        </div>
      </div>
      <div class="row">
        <label for="audioMode">Mode:</label>
        <select id="audioMode">
          <option value="always_on">Always on</option>
          <option value="push_to_talk">Push to talk</option>
        </select>
        <label for="audioInputDevice">Input device:</label>
        <select id="audioInputDevice" style="min-width:260px;">
          <option value="default">default</option>
        </select>
        <button onclick="refreshAudioDevices()" title="Refresh devices">Refresh Devices</button>
      </div>
      <div class="row">
        <label><input id="audioMuted" type="checkbox" /> muted</label>
        <label><input id="audioPaused" type="checkbox" /> paused</label>
        <button id="testMicBtn" onclick="testMicrophone()" title="Test microphone for 10 seconds">Test Mic (10s)</button>
      </div>
      <div id="micTestState" style="font-size:.88rem;opacity:.92;">Mic test idle.</div>
      <div class="meter"><div id="micTestMeterFill" class="meter-fill"></div></div>
      <div id="audioLevelState" class="hidden" style="font-size:.9rem;opacity:.9;">Audio level: unknown</div>
    </div>
  </div>

  <div id="transcriptionRuntimeModal" class="modal-overlay hidden" onclick="onTranscriptionRuntimeModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <h3 style="margin:.1rem 0;">Transcription Runtime</h3>
        <button class="danger" onclick="closeTranscriptionRuntimeModal()" title="Close">Close</button>
      </div>
      <div class="row">
        <label for="sttMode">Mode:</label>
        <select id="sttMode">
          <option value="remote">OpenAI</option>
          <option value="lmstudio">LM Studio</option>
          <option value="faster_whisper">Faster Whisper (local)</option>
          <option value="local">Custom local command</option>
        </select>
      </div>
      <div id="sttModeHelp" style="font-size:.9rem;opacity:.9;margin:.15rem 0 .3rem 0;">
        Remote OpenAI transcription uses a base URL and model.
      </div>
      <div id="sttConnectionRow" class="row">
        <span id="sttBaseURLWrap"><input id="sttBaseURL" placeholder="OpenAI base URL" style="min-width:280px;" /></span>
        <span id="sttModelWrap"><input id="sttModel" placeholder="OpenAI STT model" style="min-width:220px;" /></span>
      </div>
      <div id="sttFasterWhisperRow" class="row hidden">
        <span id="sttFasterWhisperModelWrap">
          <select id="sttFasterWhisperModel" style="min-width:220px;">
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
        <span id="sttDeviceWrap">
          <select id="sttDevice" style="min-width:220px;">
            <option value="cpu">cpu</option>
            <option value="cuda">cuda</option>
            <option value="metal">metal</option>
          </select>
        </span>
        <span id="sttComputeTypeWrap">
          <select id="sttComputeType" style="min-width:220px;">
            <option value="int8">int8</option>
            <option value="float16">float16</option>
            <option value="int8_float16">int8_float16</option>
            <option value="float32">float32</option>
          </select>
        </span>
        <span id="sttLanguageWrap"><input id="sttLanguage" placeholder="language (optional, e.g. en)" style="min-width:220px;" maxlength="24" pattern="[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8}){0,2}" title="Use a short language tag such as en or en-US" /></span>
      </div>
      <div id="sttCommandRow" class="row hidden">
        <span id="sttLocalCommandWrap"><input id="sttLocalCommand" placeholder="local command (KNIT_LOCAL_STT_CMD)" style="min-width:420px;" maxlength="2048" spellcheck="false" title="Single-line command only" /></span>
        <span id="sttTimeoutWrap"><input id="sttTimeoutSeconds" type="number" min="1" max="600" step="1" placeholder="timeout seconds (1-600)" style="min-width:220px;" /></span>
      </div>
      <div class="row">
        <button onclick="checkTranscriptionHealth()" title="Check transcription connection">Check connection</button>
      </div>
      <div id="sttHealthState" style="font-size:.9rem;opacity:.9;">Transcription health: unknown</div>
      <pre id="sttRuntimeState">transcription runtime settings will appear here</pre>
    </div>
  </div>

  <div id="videoCaptureModal" class="modal-overlay hidden" onclick="onVideoCaptureModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <h3 style="margin:.1rem 0;">Video Capture</h3>
        <button class="danger" onclick="closeVideoCaptureModal()" title="Close">Close</button>
      </div>
      <div style="font-size:.9rem;opacity:.9;margin:.2rem 0 .4rem 0;">
        Companion (🔗) is required. Use this modal to control snapshots and event clips.
      </div>
      <div class="row" style="margin-top:.2rem;">
        <button onclick="enableVisualCapture()" title="Enable visual capture">Enable Visual Capture</button>
        <button onclick="disableVisualCapture()" title="Disable visual capture">Disable Visual Capture</button>
        <label><input id="enableClips" type="checkbox" checked /> Enable event clips (5s pre + 5s post)</label>
        <label><input id="clipIncludeAudio" type="checkbox" checked /> Include microphone audio in clips (optional)</label>
      </div>
      <div class="row" style="margin-top:.3rem;">
        <label for="videoMode">Video mode:</label>
        <select id="videoMode">
          <option value="event_triggered">event_triggered</option>
          <option value="on_demand">on_demand</option>
          <option value="continuous">continuous</option>
        </select>
        <button onclick="startOnDemandClip()" title="Start on-demand clip">Start On-Demand Clip</button>
        <button onclick="stopOnDemandClip()" title="Stop on-demand clip">Stop On-Demand Clip</button>
      </div>
      <div class="row" style="margin-top:.3rem;">
        <label for="videoQualityProfile">Video quality:</label>
        <select id="videoQualityProfile">
          <option value="smaller">smaller</option>
          <option value="balanced">balanced</option>
          <option value="detail">detail</option>
        </select>
        <label><input id="allowLargeInlineMedia" type="checkbox" /> Allow large inline media when needed</label>
      </div>
      <div style="font-size:.9rem;opacity:.85;margin:.25rem 0 .1rem 0;">If preview warns a clip is too large, lower video quality, use a screenshot instead, or explicitly allow large inline media before sending.</div>
      <div class="row" style="margin-top:.3rem;">
        <label for="screenshotMode">Screenshot mode:</label>
        <select id="screenshotMode">
          <option value="full-window">Full window</option>
          <option value="selected-region">Selected region</option>
          <option value="pointer-highlighted">Pointer highlighted</option>
        </select>
        <button onclick="captureManualScreenshot()" title="Capture manual screenshot">Capture Manual Screenshot</button>
        <button onclick="clearSelection()" title="Clear selection">Clear Selection</button>
      </div>
      <div style="font-size:.9rem;opacity:.85;margin:.3rem 0 .2rem 0;">Hotkey: <code>Ctrl+Shift+S</code> captures manual screenshot.</div>
      <div id="capturePerfState" style="font-size:.85rem;opacity:.85;margin:.2rem 0 .3rem 0;">capture profile: idle</div>
      <video id="preview" autoplay muted playsinline style="width:100%;max-height:220px;border-radius:16px;border:1px solid var(--line);margin-top:.4rem;background:#f4efe7;"></video>
    </div>
  </div>

  <div id="codexRuntimeModal" class="modal-overlay hidden" onclick="onCodexRuntimeModalBackdrop(event)">
    <div class="modal-card">
      <div class="row" style="justify-content:space-between;">
        <h3 style="margin:.1rem 0;">Agent Runtime (Codex/Claude/OpenCode)</h3>
        <button class="danger" onclick="closeCodexRuntimeModal()" title="Close">Close</button>
      </div>
      <div class="modal-help">
        Choose the adapter you want to send with. The visible fields below match that adapter, and changes save automatically.
      </div>
      <div class="row">
        <label for="agentDefaultProvider">Default submit adapter:</label>
        <select id="agentDefaultProvider" style="min-width:220px;">
          <option value="codex_cli">codex_cli</option>
          <option value="claude_cli">claude_cli</option>
          <option value="codex_api">codex_api</option>
          <option value="claude_api">claude_api</option>
          <option value="opencode_cli">opencode_cli</option>
        </select>
      </div>
      <div id="runtimeProviderHelp" class="runtime-inline-note">Knit will show only the fields used by the selected adapter.</div>

      <section id="codexCliSection" class="runtime-section">
        <h4>Codex CLI</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">CLI command</span>
            <input id="codexCliCmd" placeholder="codex_cli command (KNIT_CLI_ADAPTER_CMD)" style="min-width:320px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Timeout seconds</span>
            <input id="cliTimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 600)" style="min-width:190px;" />
          </label>
        </div>
      </section>

      <section id="claudeCliSection" class="runtime-section hidden">
        <h4>Claude CLI</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">CLI command</span>
            <input id="claudeCliCmd" placeholder="claude_cli command (KNIT_CLAUDE_CLI_ADAPTER_CMD)" style="min-width:320px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Timeout seconds</span>
            <input id="claudeCliTimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 600)" style="min-width:190px;" />
          </label>
        </div>
      </section>

      <section id="opencodeCliSection" class="runtime-section hidden">
        <h4>OpenCode CLI</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">CLI command</span>
            <input id="opencodeCliCmd" placeholder="opencode_cli command (KNIT_OPENCODE_CLI_ADAPTER_CMD)" style="min-width:320px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Timeout seconds</span>
            <input id="opencodeCliTimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 600)" style="min-width:190px;" />
          </label>
        </div>
      </section>

      <section id="codexAPISection" class="runtime-section hidden">
        <h4>Codex API</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Base URL</span>
            <input id="codexAPIBaseURL" type="url" placeholder="https://api.openai.com" style="min-width:320px;" maxlength="1024" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">API timeout seconds</span>
            <input id="codexAPITimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 60)" style="min-width:190px;" />
          </label>
          <label class="field">
            <span class="field-label">OpenAI org ID</span>
            <input id="codexAPIOrg" placeholder="OPENAI_ORG_ID (optional)" style="min-width:250px;" maxlength="256" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">OpenAI project ID</span>
            <input id="codexAPIProject" placeholder="OPENAI_PROJECT_ID (optional)" style="min-width:250px;" maxlength="256" spellcheck="false" />
          </label>
        </div>
      </section>

      <section id="claudeAPISection" class="runtime-section hidden">
        <h4>Claude API</h4>
        <div id="claudeAPIKeyStatus" class="runtime-inline-note">Set ANTHROPIC_API_KEY in the environment before using claude_api.</div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Base URL</span>
            <input id="claudeAPIBaseURL" type="url" placeholder="https://api.anthropic.com" style="min-width:320px;" maxlength="1024" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">API timeout seconds</span>
            <input id="claudeAPITimeoutSeconds" type="number" min="1" max="3600" step="1" inputmode="numeric" placeholder="1-3600 (default 60)" style="min-width:190px;" />
          </label>
          <label class="field">
            <span class="field-label">Model</span>
            <input id="claudeAPIModel" placeholder="KNIT_CLAUDE_API_MODEL" style="min-width:250px;" maxlength="128" spellcheck="false" />
          </label>
        </div>
      </section>

      <section id="codexSharedSection" class="runtime-section">
        <h4>Shared Submission Settings</h4>
        <div class="runtime-inline-note">Workspace is managed from the Workspace modal and is reused across adapters.</div>
        <div class="row" style="margin-top:.5rem;">
          <div><strong>Workspace:</strong> <code id="codexWorkdirLabel">(not set)</code></div>
        </div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Output directory</span>
            <input id="codexOutputDir" placeholder="/tmp" style="min-width:180px;" maxlength="1024" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">Submit mode</span>
            <select id="submitExecutionMode">
              <option value="series">series (default)</option>
              <option value="parallel">parallel</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Post-submit rebuild command</span>
            <input id="postSubmitRebuildCmd" placeholder="post-submit rebuild command (optional)" style="min-width:420px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Post-submit verify/test command</span>
            <input id="postSubmitVerifyCmd" placeholder="post-submit verify/test command (optional)" style="min-width:420px;" maxlength="2048" spellcheck="false" title="Single-line command only" />
          </label>
          <label class="field">
            <span class="field-label">Post-submit timeout seconds</span>
            <input id="postSubmitTimeoutSec" type="number" min="1" max="7200" step="1" inputmode="numeric" placeholder="1-7200 (default 600)" style="min-width:220px;" />
          </label>
        </div>
      </section>

      <section id="codexCommonSection" class="runtime-section">
        <h4>Codex Model Settings</h4>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Profile</span>
            <input id="codexProfile" placeholder="profile (optional)" style="min-width:180px;" maxlength="128" spellcheck="false" />
          </label>
          <label class="field">
            <span class="field-label">Model</span>
            <select id="codexModel" style="min-width:220px;">
              <option value="">Use Codex default model</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Reasoning effort</span>
            <select id="codexReasoning" style="min-width:220px;">
              <option value="">Use Codex default reasoning</option>
            </select>
          </label>
        </div>
        <div class="runtime-inline-note">Profile maps to your local Codex config.toml profile. Use a separate profile for Knit if you want different MCP servers or auth behavior. Knit only loads Codex model options when you click Refresh.</div>
        <div class="row" style="margin-top:.6rem;">
          <button onclick="refreshCodexOptions()" title="Refresh Codex options">Refresh Codex Options</button>
        </div>
      </section>

      <section id="codexCLIDefaultsSection" class="runtime-section">
        <h4>Codex CLI Defaults</h4>
        <div class="runtime-inline-note" id="codexDefaultBehavior">Knit defaults local coding-agent runs to <code>workspace-write</code> sandbox and <code>never</code> approval so implementation requests can complete without falling back to read-only behavior.</div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Sandbox</span>
            <select id="codexSandbox">
              <option value="read-only">read-only</option>
              <option value="workspace-write">workspace-write</option>
              <option value="danger-full-access">danger-full-access</option>
            </select>
          </label>
          <label class="field">
            <span class="field-label">Approval policy</span>
            <select id="codexApproval">
              <option value="untrusted">untrusted</option>
              <option value="on-request">on-request</option>
              <option value="never">never</option>
            </select>
          </label>
          <div class="field">
            <span class="field-label">Repository safety</span>
            <label style="display:flex;align-items:center;gap:.55rem;padding-top:.75rem;">
              <input id="codexSkipRepoCheck" type="checkbox" checked />
              <span>Skip Git repo check</span>
            </label>
          </div>
        </div>
      </section>

      <section id="deliveryPromptSection" class="runtime-section">
        <h4>Delivery Prompt</h4>
        <div class="runtime-inline-note">Choose what the agent should do with the approved Knit feedback, then edit the resolved prompt directly if you want to adjust the handoff.</div>
        <div class="field-grid">
          <label class="field">
            <span class="field-label">Prompt template</span>
            <select id="deliveryIntentProfile" onchange="syncDeliveryIntentPromptText(true)">
              <option value="implement_changes">Implement changes</option>
              <option value="draft_plan">Draft plan</option>
              <option value="create_jira_tickets">Create Jira tickets</option>
            </select>
          </label>
          <label class="field" style="grid-column:1 / -1;">
            <span class="field-label">Prompt text</span>
            <textarea id="deliveryInstructionText" rows="9" placeholder="The selected prompt template will appear here."></textarea>
          </label>
        </div>
      </section>

      <pre id="codexRuntimeState">runtime codex settings will appear here</pre>
    </div>
  </div>

<script>
const controlToken = '__KNIT_TOKEN__';
const val = id => document.getElementById(id).value;
const targetWindowEl = document.getElementById('targetWindow');
const targetURLEl = document.getElementById('targetURL');
const stateEl = document.getElementById('state');
const noteStatus = document.getElementById('noteStatus');
const sensitiveCaptureBadgesEl = document.getElementById('sensitiveCaptureBadges');
const captureSettingsModalEl = document.getElementById('captureSettingsModal');
const workspaceModalEl = document.getElementById('workspaceModal');
const workspaceModalCloseBtnEl = document.getElementById('workspaceModalCloseBtn');
const workspaceModalStatusEl = document.getElementById('workspaceModalStatus');
const audioControlsModalEl = document.getElementById('audioControlsModal');
const transcriptionRuntimeModalEl = document.getElementById('transcriptionRuntimeModal');
const videoCaptureModalEl = document.getElementById('videoCaptureModal');
const codexRuntimeModalEl = document.getElementById('codexRuntimeModal');
const captureGuideSidebarEl = document.getElementById('captureGuideSidebar');
const guideInfoBtnEl = document.getElementById('guideInfoBtn');
const appToastEl = document.getElementById('appToast');
const captureGuideStatusEl = document.getElementById('captureGuideStatus');
const platformRuntimeStateEl = document.getElementById('platformRuntimeState');
const composerSupportStateEl = document.getElementById('composerSupportState');
const extensionPairingCodeStateEl = document.getElementById('extensionPairingCodeState');
const extensionPairingListEl = document.getElementById('extensionPairingList');
const audioNoteBtnEl = document.getElementById('audioNoteBtn');
const videoNoteBtnEl = document.getElementById('videoNoteBtn');
const audioModeEl = document.getElementById('audioMode');
const audioInputDeviceEl = document.getElementById('audioInputDevice');
const audioMutedEl = document.getElementById('audioMuted');
const audioPausedEl = document.getElementById('audioPaused');
const audioLevelStateEl = document.getElementById('audioLevelState');
const testMicBtnEl = document.getElementById('testMicBtn');
const micTestStateEl = document.getElementById('micTestState');
const micTestMeterFillEl = document.getElementById('micTestMeterFill');
const sttModeEl = document.getElementById('sttMode');
const sttBaseURLEl = document.getElementById('sttBaseURL');
const sttModelEl = document.getElementById('sttModel');
const sttFasterWhisperModelEl = document.getElementById('sttFasterWhisperModel');
const sttDeviceEl = document.getElementById('sttDevice');
const sttComputeTypeEl = document.getElementById('sttComputeType');
const sttLanguageEl = document.getElementById('sttLanguage');
const sttLocalCommandEl = document.getElementById('sttLocalCommand');
const sttTimeoutSecondsEl = document.getElementById('sttTimeoutSeconds');
const sttModeHelpEl = document.getElementById('sttModeHelp');
const sttConnectionRowEl = document.getElementById('sttConnectionRow');
const sttFasterWhisperRowEl = document.getElementById('sttFasterWhisperRow');
const sttCommandRowEl = document.getElementById('sttCommandRow');
const sttBaseURLWrapEl = document.getElementById('sttBaseURLWrap');
const sttModelWrapEl = document.getElementById('sttModelWrap');
const sttFasterWhisperModelWrapEl = document.getElementById('sttFasterWhisperModelWrap');
const sttDeviceWrapEl = document.getElementById('sttDeviceWrap');
const sttComputeTypeWrapEl = document.getElementById('sttComputeTypeWrap');
const sttLanguageWrapEl = document.getElementById('sttLanguageWrap');
const sttLocalCommandWrapEl = document.getElementById('sttLocalCommandWrap');
const sttTimeoutWrapEl = document.getElementById('sttTimeoutWrap');
const sttHealthStateEl = document.getElementById('sttHealthState');
const sttRuntimeStateEl = document.getElementById('sttRuntimeState');
const preview = document.getElementById('preview');
const enableClips = document.getElementById('enableClips');
const clipIncludeAudioEl = document.getElementById('clipIncludeAudio');
const videoModeEl = document.getElementById('videoMode');
const videoQualityProfileEl = document.getElementById('videoQualityProfile');
const allowLargeInlineMediaEl = document.getElementById('allowLargeInlineMedia');
const screenshotMode = document.getElementById('screenshotMode');
const capturePerfStateEl = document.getElementById('capturePerfState');
const agentDefaultProviderEl = document.getElementById('agentDefaultProvider');
const deliveryIntentProfileEl = document.getElementById('deliveryIntentProfile');
const deliveryInstructionTextEl = document.getElementById('deliveryInstructionText');
const previewBtnEl = document.getElementById('previewBtn');
const submitBtnEl = document.getElementById('submitBtn');
const openLogBtnEl = document.getElementById('openLogBtn');
const submitStateEl = document.getElementById('submitState');
const deliveryBadgeEl = document.getElementById('deliveryBadge');
const queueStateEl = document.getElementById('queueState');
const submitResultEl = document.getElementById('submitResult');
const payloadPreviewEl = document.getElementById('payloadPreview');
const captureInputValuesToggleEl = document.getElementById('captureInputValuesToggle');
const liveSubmitLogEl = document.getElementById('liveSubmitLog');
const liveSubmitCommentaryEl = document.getElementById('liveSubmitCommentary');
const configExportEl = document.getElementById('configExport');
const configProfileEl = document.getElementById('configProfile');
const applyProfileBtn = document.getElementById('save');
const configLockStatusEl = document.getElementById('configLockStatus');
const capturePolicyEl = document.getElementById('capturePolicy');
const codexRuntimeStateEl = document.getElementById('codexRuntimeState');
const workspaceDirEl = document.getElementById('workspaceDir');
const workspaceBrowserStateEl = document.getElementById('workspaceBrowserState');
const codexCliCmdEl = document.getElementById('codexCliCmd');
const claudeCliCmdEl = document.getElementById('claudeCliCmd');
const opencodeCliCmdEl = document.getElementById('opencodeCliCmd');
const codexWorkdirLabelEl = document.getElementById('codexWorkdirLabel');
const codexOutputDirEl = document.getElementById('codexOutputDir');
const cliTimeoutSecondsEl = document.getElementById('cliTimeoutSeconds');
const claudeCliTimeoutSecondsEl = document.getElementById('claudeCliTimeoutSeconds');
const opencodeCliTimeoutSecondsEl = document.getElementById('opencodeCliTimeoutSeconds');
const submitExecutionModeEl = document.getElementById('submitExecutionMode');
const codexSandboxEl = document.getElementById('codexSandbox');
const codexApprovalEl = document.getElementById('codexApproval');
const codexSkipRepoCheckEl = document.getElementById('codexSkipRepoCheck');
const codexProfileEl = document.getElementById('codexProfile');
const codexModelEl = document.getElementById('codexModel');
const codexReasoningEl = document.getElementById('codexReasoning');
const codexAPIBaseURLEl = document.getElementById('codexAPIBaseURL');
const claudeAPIBaseURLEl = document.getElementById('claudeAPIBaseURL');
const submitAttemptOutputPreviewByID = new Map();
const submitAttemptOutputPreviewInflight = new Set();
const seenSubmitRecoveryNotices = new Set();
const codexAPITimeoutSecondsEl = document.getElementById('codexAPITimeoutSeconds');
const claudeAPITimeoutSecondsEl = document.getElementById('claudeAPITimeoutSeconds');
const codexAPIOrgEl = document.getElementById('codexAPIOrg');
const codexAPIProjectEl = document.getElementById('codexAPIProject');
const claudeAPIModelEl = document.getElementById('claudeAPIModel');
const claudeAPIKeyStatusEl = document.getElementById('claudeAPIKeyStatus');
const codexOptionsStateEl = document.getElementById('codexOptionsState');
const runtimeProviderHelpEl = document.getElementById('runtimeProviderHelp');
const codexCliSectionEl = document.getElementById('codexCliSection');
const claudeCliSectionEl = document.getElementById('claudeCliSection');
const opencodeCliSectionEl = document.getElementById('opencodeCliSection');
const codexAPISectionEl = document.getElementById('codexAPISection');
const claudeAPISectionEl = document.getElementById('claudeAPISection');
const codexSharedSectionEl = document.getElementById('codexSharedSection');
const codexCommonSectionEl = document.getElementById('codexCommonSection');
const codexCLIDefaultsSectionEl = document.getElementById('codexCLIDefaultsSection');
const postSubmitRebuildCmdEl = document.getElementById('postSubmitRebuildCmd');
const postSubmitVerifyCmdEl = document.getElementById('postSubmitVerifyCmd');
const postSubmitTimeoutSecEl = document.getElementById('postSubmitTimeoutSec');
const reviewModeEl = document.getElementById('reviewMode');
const laserModeEnabledEl = document.getElementById('laserModeEnabled');
const sessionPlayBtnEl = document.getElementById('sessionPlayBtn');
const sessionPauseResumeBtnEl = document.getElementById('sessionPauseResumeBtn');
const sessionStopBtnEl = document.getElementById('sessionStopBtn');

let currentState = null;
let previewDeliveryOptions = { redactReplayValues: false, omitVideoClips: false, omitVideoEventIDs: [] };
let displayStream = null;
let clipRecorder = null;
let clipSourceStream = null;
let clipAudioStream = null;
let clipRenderCanvas = null;
let clipRenderCtx = null;
let clipRenderRAF = 0;
let clipMimeType = 'video/webm';
let clipPointerOverlayEnabled = false;
let clipProfileState = { fps: 12, bitrate: 1200000 };
let preChunks = [];
let clipSubscribers = [];
let onDemandChunks = [];
let onDemandRecording = false;
let pendingOnDemandClip = null;
let continuousChunks = [];
let manualScreenshotBlob = null;
let frozenFrameBlob = null;
let laserTrail = [];
let laserTrailMax = 80;
let selectionRect = null;
let selecting = false;
let selectionStart = null;
let submitInFlight = false;
let submitTimer = null;
let liveLogAttemptId = '';
let liveLogOffset = 0;
let liveLogRawText = '';
let liveLogCompletedForAttempt = '';
let liveLogUnavailableForAttempt = '';
let watchedSubmitAttemptIDs = new Set();
let submitAttemptStatusByID = new Map();
let submitAttemptNotificationsReady = false;
let openSubmitAttemptRawJSONIDs = new Set();
let submitAttemptRawJSONScrollTopByID = new Map();
let latestPayloadPreviewData = null;
let stateRefreshError = '';
let clipBlobCacheByEventID = new Map();
let clipResizeInFlight = new Set();
let pttHeld = false;
let audioConfigDirty = false;
let audioConfigApplying = false;
let audioConfigApplyTimer = 0;
let sttRuntimeDirty = false;
let sttRuntimeApplying = false;
let sttRuntimeApplyTimer = 0;
let micTestRunning = false;
let workspacePrompted = false;
let workspaceSelectionRequired = false;
let codexOptionsLoaded = false;
let codexOptionsAttempted = false;
let codexRuntimeDirty = false;
let codexRuntimeApplying = false;
let codexRuntimeApplyTimer = 0;
let audioNoteRecorder = null;
let audioNoteStream = null;
let audioNoteChunks = [];
let audioNoteStopPromise = null;
let videoNoteAudioRecorder = null;
let videoNoteClipRecorder = null;
let videoNoteMicStream = null;
let videoNoteRenderCanvas = null;
let videoNoteRenderCtx = null;
let videoNoteRenderRAF = 0;
let videoNoteCanvasStream = null;
let videoNoteCombinedStream = null;
let videoNoteAudioChunks = [];
let videoNoteClipChunks = [];
let videoNoteAudioStopPromise = null;
let videoNoteClipStopPromise = null;
let activeVideoNote = null;
let videoNoteFinalizing = false;
let laserForcedByVideo = false;
let voiceRecognition = null;
let voiceListening = false;
const DEFAULT_TARGET_WINDOW = 'Browser Review';
const UI_SETTINGS_KEY = 'knit_ui_settings_v1';
let uiSettings = {};
let currentTheme = 'light';
let appToastTimer = 0;

function loadUISettings() {
  try {
    const raw = window.localStorage ? window.localStorage.getItem(UI_SETTINGS_KEY) : '';
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
    return parsed;
  } catch (_) {
    return {};
  }
}

function saveUISettings() {
  try {
    if (!window.localStorage) return;
    window.localStorage.setItem(UI_SETTINGS_KEY, JSON.stringify(uiSettings));
  } catch (_) {}
}

function hasUISetting(key) {
  return Object.prototype.hasOwnProperty.call(uiSettings, key);
}

function setUISetting(key, value) {
  if (!key) return;
  uiSettings[key] = value;
  saveUISettings();
}

function showToast(msg, isError = false) {
  if (!appToastEl) return;
  appToastEl.textContent = msg;
  appToastEl.classList.toggle('error', !!isError);
  appToastEl.classList.add('visible');
  if (appToastTimer) {
    clearTimeout(appToastTimer);
  }
  appToastTimer = window.setTimeout(() => {
    appToastEl.classList.remove('visible');
  }, 2200);
}

function handleStateRefreshFailure(message) {
  const msg = String(message || 'State refresh failed.').trim();
  if (!msg) return;
  if (stateRefreshError !== msg) {
    showToast('Main UI stopped refreshing: ' + msg, true);
  }
  stateRefreshError = msg;
  if (submitStateEl) {
    submitStateEl.textContent = 'Main UI is out of sync: ' + msg;
    submitStateEl.style.color = '#c34f4f';
  }
  if (queueStateEl) {
    queueStateEl.innerHTML = '<div style="color:#c34f4f;"><strong>State refresh failed.</strong> Reload this page or reopen the daemon UI after fixing the token or daemon connection.</div>';
  }
}

function isTerminalSubmitStatus(status) {
  const value = String(status || '').trim();
  return value === 'submitted' || value === 'failed' || value === 'canceled';
}

function isCancelableSubmitStatus(status) {
  const value = String(status || '').trim();
  return value === 'queued' || value === 'deferred_offline' || value === 'retry_wait' || value === 'in_progress';
}

function truncateSubmitToastText(value, limit = 88) {
  const text = String(value || '').trim().replace(/\s+/g, ' ');
  if (!text) return '';
  if (text.length <= limit) return text;
  return text.slice(0, Math.max(0, limit - 1)).trimEnd() + '...';
}

function submitAttemptToastMessage(attempt) {
  const status = String(attempt?.status || '').trim();
  const request = truncateSubmitToastText(requestPreviewText(attempt));
  const attemptID = String(attempt?.attempt_id || '').trim();
  if (status === 'failed') {
    if (request) return 'Request failed: ' + request;
    return 'Request failed' + (attemptID ? ' (' + attemptID + ')' : '.');
  }
  if (status === 'canceled') {
    if (request) return 'Request canceled: ' + request;
    return 'Request canceled' + (attemptID ? ' (' + attemptID + ')' : '.');
  }
  if (request) return 'Request completed: ' + request;
  return 'Request completed' + (attemptID ? ' (' + attemptID + ')' : '.');
}

function notifySubmitAttemptTransitions(attempts) {
  const nextStatuses = new Map();
  const list = Array.isArray(attempts) ? attempts : [];
  if (!submitAttemptNotificationsReady) {
    list.forEach(attempt => {
      const attemptID = String(attempt?.attempt_id || '').trim();
      if (!attemptID) return;
      nextStatuses.set(attemptID, String(attempt?.status || '').trim());
    });
    submitAttemptStatusByID = nextStatuses;
    submitAttemptNotificationsReady = true;
    return;
  }
  list.forEach(attempt => {
    const attemptID = String(attempt?.attempt_id || '').trim();
    if (!attemptID) return;
    const status = String(attempt?.status || '').trim();
    const prevStatus = String(submitAttemptStatusByID.get(attemptID) || '').trim();
    const watched = watchedSubmitAttemptIDs.has(attemptID);
    const transitioned = prevStatus !== status;
    if (isTerminalSubmitStatus(status) && transitioned && (watched || prevStatus)) {
      showToast(submitAttemptToastMessage(attempt), status === 'failed');
      watchedSubmitAttemptIDs.delete(attemptID);
    }
    nextStatuses.set(attemptID, status);
  });
  submitAttemptStatusByID = nextStatuses;
}

function normalizeTheme(theme) {
  return String(theme || '').trim().toLowerCase() === 'dark' ? 'dark' : 'light';
}

function applyTheme(theme) {
  currentTheme = normalizeTheme(theme);
  document.documentElement.setAttribute('data-theme', currentTheme);
  const btn = document.getElementById('themeToggleBtn');
  if (!btn) return;
  const nextTheme = currentTheme === 'dark' ? 'light' : 'dark';
  btn.textContent = currentTheme === 'dark' ? '☀' : '☾';
  btn.title = nextTheme === 'dark' ? 'Switch to dark theme' : 'Switch to light theme';
  btn.setAttribute('aria-label', btn.title);
}

function toggleTheme() {
  const nextTheme = currentTheme === 'dark' ? 'light' : 'dark';
  applyTheme(nextTheme);
  setUISetting('theme', nextTheme);
}

function bindPersistedField(el, key, opts = {}) {
  if (!el || !key) return;
  const isCheckbox = !!opts.checkbox;
  if (hasUISetting(key)) {
    if (isCheckbox) {
      el.checked = !!uiSettings[key];
    } else {
      el.value = String(uiSettings[key] ?? '');
    }
  }
  const save = () => {
    if (isCheckbox) {
      setUISetting(key, !!el.checked);
      if (typeof opts.afterSave === 'function') opts.afterSave();
      return;
    }
    setUISetting(key, String(el.value ?? ''));
    if (typeof opts.afterSave === 'function') opts.afterSave();
  };
  el.addEventListener(isCheckbox ? 'change' : 'input', save);
  if (!isCheckbox && (el.tagName === 'SELECT' || el.type === 'number')) {
    el.addEventListener('change', save);
  }
}

function initPersistentSettings() {
  uiSettings = loadUISettings();
  applyTheme(normalizeTheme(uiSettings.theme || 'light'));
  bindPersistedField(targetWindowEl, 'target_window');
  bindPersistedField(targetURLEl, 'target_url');
  if (targetWindowEl && !String(targetWindowEl.value || '').trim()) {
    targetWindowEl.value = DEFAULT_TARGET_WINDOW;
  }
  bindPersistedField(workspaceDirEl, 'workspace_dir');
  bindPersistedField(agentDefaultProviderEl, 'default_provider');
  bindPersistedField(codexCliCmdEl, 'cli_adapter_cmd');
  bindPersistedField(claudeCliCmdEl, 'claude_cli_adapter_cmd');
  bindPersistedField(opencodeCliCmdEl, 'opencode_cli_adapter_cmd');
  bindPersistedField(cliTimeoutSecondsEl, 'cli_timeout_seconds');
  bindPersistedField(claudeCliTimeoutSecondsEl, 'claude_cli_timeout_seconds');
  bindPersistedField(opencodeCliTimeoutSecondsEl, 'opencode_cli_timeout_seconds');
  bindPersistedField(submitExecutionModeEl, 'submit_execution_mode');
  bindPersistedField(codexOutputDirEl, 'codex_output_dir');
  bindPersistedField(codexSandboxEl, 'codex_sandbox');
  bindPersistedField(codexApprovalEl, 'codex_approval_policy');
  bindPersistedField(codexSkipRepoCheckEl, 'codex_skip_git_repo_check', { checkbox: true });
  bindPersistedField(codexProfileEl, 'codex_profile');
  bindPersistedField(codexModelEl, 'codex_model');
  bindPersistedField(codexReasoningEl, 'codex_reasoning_effort');
  bindPersistedField(codexAPIBaseURLEl, 'openai_base_url');
  bindPersistedField(codexAPITimeoutSecondsEl, 'codex_api_timeout_seconds');
  bindPersistedField(codexAPIOrgEl, 'openai_org_id');
  bindPersistedField(codexAPIProjectEl, 'openai_project_id');
  bindPersistedField(claudeAPIBaseURLEl, 'anthropic_base_url');
  bindPersistedField(claudeAPITimeoutSecondsEl, 'claude_api_timeout_seconds');
  bindPersistedField(claudeAPIModelEl, 'claude_api_model');
  bindPersistedField(deliveryIntentProfileEl, 'delivery_intent_profile');
  bindPersistedField(deliveryInstructionTextEl, 'delivery_instruction_text');
  bindPersistedField(postSubmitRebuildCmdEl, 'post_submit_rebuild_cmd');
  bindPersistedField(postSubmitVerifyCmdEl, 'post_submit_verify_cmd');
  bindPersistedField(postSubmitTimeoutSecEl, 'post_submit_timeout_seconds');
  bindPersistedField(audioModeEl, 'audio_mode', { afterSave: renderSensitiveCaptureBadges });
  bindPersistedField(reviewModeEl, 'review_mode');
  bindPersistedField(laserModeEnabledEl, 'laser_mode_enabled', { checkbox: true });
  bindPersistedField(enableClips, 'enable_clips', { checkbox: true });
  bindPersistedField(clipIncludeAudioEl, 'clip_include_audio', { checkbox: true });
  bindPersistedField(videoModeEl, 'video_mode', { afterSave: renderSensitiveCaptureBadges });
  bindPersistedField(videoQualityProfileEl, 'video_quality_profile');
  bindPersistedField(allowLargeInlineMediaEl, 'allow_large_inline_media', { checkbox: true, afterSave: renderSensitiveCaptureBadges });
  bindPersistedField(screenshotMode, 'screenshot_mode');
  bindPersistedField(sttModeEl, 'stt_mode');
  bindPersistedField(sttBaseURLEl, 'stt_base_url');
  bindPersistedField(sttModelEl, 'stt_model');
  bindPersistedField(sttFasterWhisperModelEl, 'stt_model');
  bindPersistedField(sttDeviceEl, 'stt_device');
  bindPersistedField(sttComputeTypeEl, 'stt_compute_type');
  bindPersistedField(sttLanguageEl, 'stt_language');
  bindPersistedField(sttLocalCommandEl, 'stt_local_command');
  bindPersistedField(sttTimeoutSecondsEl, 'stt_timeout_seconds');
}

function syncSessionDetailInputsFromState() {
  const sess = currentState?.session || {};
  const ptr = currentState?.pointer_latest || {};
  const derivedWindow = String(sess.target_window || ptr.window || '').trim();
  const derivedURL = String(sess.target_url || ptr.url || '').trim();
  if (targetWindowEl) {
    targetWindowEl.value = derivedWindow || String(targetWindowEl.value || '').trim() || DEFAULT_TARGET_WINDOW;
  }
  if (targetURLEl && derivedURL) {
    targetURLEl.value = derivedURL;
  }
}

function logNote(msg, isError=false) {
  noteStatus.textContent = msg;
  noteStatus.style.color = isError ? '#f56565' : '#48bb78';
}

function setCaptureGuideSidebarOpen(open) {
  const isOpen = !!open;
  if (captureGuideSidebarEl) {
    captureGuideSidebarEl.classList.toggle('hidden', !isOpen);
  }
  if (guideInfoBtnEl) {
    guideInfoBtnEl.classList.toggle('hidden', isOpen);
  }
  document.body.classList.toggle('guide-open', isOpen);
  setUISetting('capture_guide_open', isOpen);
}

function closeCaptureGuideSidebar() {
  setCaptureGuideSidebarOpen(false);
}

function openCaptureGuideSidebar() {
  setCaptureGuideSidebarOpen(true);
}

function setPTT(active) {
  pttHeld = !!active;
}

function updateWorkspaceModalState() {
  const selected = (workspaceDirEl?.value || '').trim();
  const locked = !!currentState?.config_locked;
  workspaceSelectionRequired = !locked && !selected;
  if (workspaceModalCloseBtnEl) {
    workspaceModalCloseBtnEl.disabled = workspaceSelectionRequired;
    workspaceModalCloseBtnEl.title = workspaceSelectionRequired ? 'Select a workspace first.' : 'Close';
  }
  if (workspaceModalStatusEl) {
    if (locked) {
      workspaceModalStatusEl.textContent = 'Workspace is managed by policy in this environment.';
      workspaceModalStatusEl.style.color = '#f6ad55';
    } else if (workspaceSelectionRequired) {
      workspaceModalStatusEl.textContent = 'Workspace selection required before continuing.';
      workspaceModalStatusEl.style.color = '#f6ad55';
    } else {
      workspaceModalStatusEl.textContent = 'Workspace selected: ' + selected;
      workspaceModalStatusEl.style.color = '#48bb78';
    }
  }
}

function openWorkspaceModal() {
  if (!workspaceModalEl) return;
  closeCaptureSettingsModal();
  workspaceModalEl.classList.remove('hidden');
  workspaceModalEl.classList.add('open');
  updateWorkspaceModalState();
}

function closeWorkspaceModal() {
  if (!workspaceModalEl) return;
  updateWorkspaceModalState();
  if (workspaceSelectionRequired) {
    logNote('Select a workspace folder before closing this modal.', true);
    return;
  }
  workspaceModalEl.classList.remove('open');
  workspaceModalEl.classList.add('hidden');
}

function onWorkspaceModalBackdrop(event) {
  if (!workspaceModalEl) return;
  if (event && event.target === workspaceModalEl) {
    closeWorkspaceModal();
  }
}

function openCaptureSettingsModal() {
  if (!captureSettingsModalEl) return;
  captureSettingsModalEl.classList.remove('hidden');
  captureSettingsModalEl.classList.add('open');
}

function closeCaptureSettingsModal() {
  if (!captureSettingsModalEl) return;
  captureSettingsModalEl.classList.remove('open');
  captureSettingsModalEl.classList.add('hidden');
}

function onCaptureSettingsModalBackdrop(event) {
  if (!captureSettingsModalEl) return;
  if (event && event.target === captureSettingsModalEl) {
    closeCaptureSettingsModal();
  }
}

function openDocsBrowser(name) {
  closeCaptureSettingsModal();
  const url = new URL(window.location.origin + '/docs');
  if (controlToken) url.searchParams.set('token', controlToken);
  if (name) url.searchParams.set('name', String(name).trim());
  const tab = window.open(url.pathname + url.search, 'knitDocs', 'noopener');
  if (!tab) {
    logNote('Docs tab blocked. Allow popups for 127.0.0.1 and try again.', true);
    return;
  }
  try { tab.focus(); } catch (_) {}
}

function openAudioControlsModal() {
  if (!audioControlsModalEl) return;
  closeCaptureSettingsModal();
  audioControlsModalEl.classList.remove('hidden');
  audioControlsModalEl.classList.add('open');
  refreshAudioDevices();
}

function closeAudioControlsModal() {
  if (!audioControlsModalEl) return;
  audioControlsModalEl.classList.remove('open');
  audioControlsModalEl.classList.add('hidden');
  setPTT(false);
}

function onAudioControlsModalBackdrop(event) {
  if (!audioControlsModalEl) return;
  if (event && event.target === audioControlsModalEl) {
    closeAudioControlsModal();
  }
}

function openTranscriptionRuntimeFromAudioModal() {
  closeAudioControlsModal();
  openTranscriptionRuntimeModal();
}

function openTranscriptionRuntimeModal() {
  if (!transcriptionRuntimeModalEl) return;
  transcriptionRuntimeModalEl.classList.remove('hidden');
  transcriptionRuntimeModalEl.classList.add('open');
  syncSTTRuntimeUIFromState();
}

function closeTranscriptionRuntimeModal() {
  if (!transcriptionRuntimeModalEl) return;
  transcriptionRuntimeModalEl.classList.remove('open');
  transcriptionRuntimeModalEl.classList.add('hidden');
}

function onTranscriptionRuntimeModalBackdrop(event) {
  if (!transcriptionRuntimeModalEl) return;
  if (event && event.target === transcriptionRuntimeModalEl) {
    closeTranscriptionRuntimeModal();
  }
}

function openVideoCaptureModal() {
  if (!videoCaptureModalEl) return;
  closeCaptureSettingsModal();
  videoCaptureModalEl.classList.remove('hidden');
  videoCaptureModalEl.classList.add('open');
}

function closeVideoCaptureModal() {
  if (!videoCaptureModalEl) return;
  videoCaptureModalEl.classList.remove('open');
  videoCaptureModalEl.classList.add('hidden');
}

function onVideoCaptureModalBackdrop(event) {
  if (!videoCaptureModalEl) return;
  if (event && event.target === videoCaptureModalEl) {
    closeVideoCaptureModal();
  }
}

function openCodexRuntimeModal() {
  if (!codexRuntimeModalEl) return;
  closeCaptureSettingsModal();
  codexRuntimeModalEl.classList.remove('hidden');
  codexRuntimeModalEl.classList.add('open');
  syncDeliveryIntentPromptText(false);
  syncProviderOptionsFromState();
  syncCodexRuntimeModeUI();
}

function openAgentSettingsFromNotice(event) {
  if (event) event.preventDefault();
  openCodexRuntimeModal();
  window.setTimeout(() => {
    try {
      agentDefaultProviderEl?.focus();
    } catch (_) {}
  }, 0);
}

function closeCodexRuntimeModal() {
  if (!codexRuntimeModalEl) return;
  codexRuntimeModalEl.classList.remove('open');
  codexRuntimeModalEl.classList.add('hidden');
}

function onCodexRuntimeModalBackdrop(event) {
  if (!codexRuntimeModalEl) return;
  if (event && event.target === codexRuntimeModalEl) {
    closeCodexRuntimeModal();
  }
}

function setCodexRuntimeStatus(message, isError = false) {
  if (!codexOptionsStateEl) return;
  codexOptionsStateEl.textContent = message;
  codexOptionsStateEl.style.color = isError ? '#c34f4f' : '#1c7c74';
}

function syncCodexRuntimeModeUI() {
  const provider = selectedProvider();
  const toggle = (el, visible) => {
    if (el) el.classList.toggle('hidden', !visible);
  };
  toggle(codexCliSectionEl, provider === 'codex_cli');
  toggle(claudeCliSectionEl, provider === 'claude_cli');
  toggle(opencodeCliSectionEl, provider === 'opencode_cli');
  toggle(codexAPISectionEl, provider === 'codex_api');
  toggle(claudeAPISectionEl, provider === 'claude_api');
  toggle(codexCommonSectionEl, provider === 'codex_cli' || provider === 'codex_api');
  toggle(codexCLIDefaultsSectionEl, provider === 'codex_cli');
  toggle(codexSharedSectionEl, true);
  if (runtimeProviderHelpEl) {
    switch (provider) {
    case 'codex_api':
      runtimeProviderHelpEl.textContent = 'Codex API uses a base URL, API timeout, and optional OpenAI org/project IDs. Shared submission settings still apply.';
      break;
    case 'claude_api':
      runtimeProviderHelpEl.textContent = 'Claude API uses Anthropic base URL, model, and timeout settings. Codex-only sandbox, approval, reasoning, and org/project fields stay hidden.';
      break;
    case 'claude_cli':
      runtimeProviderHelpEl.textContent = 'Claude CLI only needs its command and timeout here. Codex-only sandbox, approval, model, and reasoning settings stay hidden.';
      break;
    case 'opencode_cli':
      runtimeProviderHelpEl.textContent = 'OpenCode CLI only needs its command and timeout here. Codex-only sandbox, approval, model, and reasoning settings stay hidden.';
      break;
    default:
      runtimeProviderHelpEl.textContent = 'Codex CLI uses Knit defaults of workspace-write sandbox and never approval unless you explicitly choose different values here.';
      break;
    }
  }
}

function resetRuntimeFieldValidity(el) {
  if (el && typeof el.setCustomValidity === 'function') {
    el.setCustomValidity('');
  }
}

function invalidRuntimeField(el, message) {
  if (el && typeof el.setCustomValidity === 'function') {
    el.setCustomValidity(message);
    try { el.reportValidity(); } catch (_) {}
  }
  throw new Error(message);
}

function readRuntimeSingleLine(el, label, maxLen = 2048) {
  resetRuntimeFieldValidity(el);
  const value = String(el?.value || '').trim();
  if (!value) return '';
  if (/[\r\n]/.test(value)) {
    invalidRuntimeField(el, label + ' must stay on one line.');
  }
  if (value.length > maxLen) {
    invalidRuntimeField(el, label + ' must be ' + maxLen + ' characters or fewer.');
  }
  return value;
}

function readRuntimeURL(el, label, maxLen = 1024) {
  const value = readRuntimeSingleLine(el, label, maxLen);
  if (!value) return '';
  let parsed;
  try {
    parsed = new URL(value);
  } catch (_) {
    invalidRuntimeField(el, label + ' must be a valid http or https URL.');
  }
  if (!parsed || !/^https?:$/.test(parsed.protocol)) {
    invalidRuntimeField(el, label + ' must use http or https.');
  }
  return value;
}

function readRuntimeSeconds(el, label, maxSeconds = 3600) {
  resetRuntimeFieldValidity(el);
  const raw = String(el?.value || '').trim();
  if (!raw) return 0;
  const value = Number.parseInt(raw, 10);
  if (!Number.isFinite(value) || value < 1 || value > maxSeconds) {
    invalidRuntimeField(el, label + ' must be between 1 and ' + maxSeconds + '.');
  }
  return value;
}

function scheduleCodexRuntimeApply() {
  codexRuntimeDirty = true;
  syncCodexRuntimeModeUI();
  setCodexRuntimeStatus('Saving runtime settings...');
  if (codexRuntimeApplyTimer) {
    clearTimeout(codexRuntimeApplyTimer);
  }
  codexRuntimeApplyTimer = window.setTimeout(() => {
    applyCodexRuntime();
  }, 350);
}

function syncAudioUIFromState() {
  const audio = currentState?.audio?.state || {};
  const devices = Array.isArray(currentState?.audio?.devices) ? currentState.audio.devices : [];
  const preservingUserDraft = audioConfigDirty || audioConfigApplying;
  const selectedMode = String((preservingUserDraft ? audioModeEl?.value : (audio.mode || audioModeEl?.value)) || 'always_on');
  if (audioModeEl && !preservingUserDraft) audioModeEl.value = audio.mode || audioModeEl.value || 'always_on';
  if (audioMutedEl && !preservingUserDraft) audioMutedEl.checked = !!audio.muted;
  if (audioPausedEl && !preservingUserDraft) audioPausedEl.checked = !!audio.paused;
  if (selectedMode !== 'push_to_talk' && pttHeld) {
    setPTT(false);
  }
  setAudioLevelStateVisible(micTestRunning);
  if (audioLevelStateEl) {
    const lvl = Number(audio.last_level || 0);
    const valid = !!audio.level_valid;
    const mode = audio.mode || 'unknown';
    audioLevelStateEl.textContent = 'Audio level: ' + lvl.toFixed(3) + ' | valid=' + valid + ' | mode=' + mode;
    audioLevelStateEl.style.color = valid ? '#48bb78' : '#f6ad55';
  }
  if (audioInputDeviceEl && !preservingUserDraft) {
    const curr = String(audio.input_device_id || audioInputDeviceEl.value || 'default');
    audioInputDeviceEl.innerHTML = '';
    if (!devices.length) {
      const fallback = document.createElement('option');
      fallback.value = 'default';
      fallback.textContent = 'default';
      audioInputDeviceEl.appendChild(fallback);
    } else {
      devices.forEach(d => {
        const opt = document.createElement('option');
        const id = String((d && d.id) || '').trim() || 'default';
        const label = String((d && d.label) || id).trim();
        opt.value = id;
        opt.textContent = label;
        audioInputDeviceEl.appendChild(opt);
      });
    }
    audioInputDeviceEl.value = curr;
  }
}

function syncSTTRuntimeUIFromState() {
  const rt = currentState?.runtime_transcription || {};
  if (sttModeEl) sttModeEl.value = rt.mode || currentState?.transcription_mode || sttModeEl.value || 'faster_whisper';
  if (sttBaseURLEl) sttBaseURLEl.value = rt.endpoint || sttBaseURLEl.value || '';
  if (sttModelEl) sttModelEl.value = rt.model || sttModelEl.value || '';
  if (sttFasterWhisperModelEl) sttFasterWhisperModelEl.value = normalizeFasterWhisperModel(rt.model || sttModelEl?.value || sttFasterWhisperModelEl.value || '');
  if (sttDeviceEl) sttDeviceEl.value = rt.device || sttDeviceEl.value || '';
  if (sttComputeTypeEl) sttComputeTypeEl.value = rt.compute_type || sttComputeTypeEl.value || '';
  if (sttLanguageEl) sttLanguageEl.value = rt.language || sttLanguageEl.value || '';
  if (sttLocalCommandEl) sttLocalCommandEl.value = rt.local_command || sttLocalCommandEl.value || '';
  if (sttTimeoutSecondsEl) sttTimeoutSecondsEl.value = rt.timeout_seconds || sttTimeoutSecondsEl.value || '';
  syncSTTRuntimeModeUI();
  if (sttRuntimeStateEl) sttRuntimeStateEl.textContent = JSON.stringify(rt, null, 2);
}

const fasterWhisperModelOptions = [
  'tiny.en', 'tiny', 'base.en', 'base', 'small.en', 'small', 'medium.en', 'medium',
  'large-v1', 'large-v2', 'large-v3', 'large', 'distil-large-v2', 'distil-medium.en',
  'distil-small.en', 'distil-large-v3', 'distil-large-v3.5', 'large-v3-turbo', 'turbo'
];
const defaultFasterWhisperModel = 'small';

function normalizeFasterWhisperModel(value) {
  const model = String(value || '').trim();
  return fasterWhisperModelOptions.includes(model) ? model : defaultFasterWhisperModel;
}

function currentSTTModelValue() {
  const mode = String(sttModeEl?.value || 'faster_whisper').trim().toLowerCase();
  if (mode === 'faster_whisper') {
    return normalizeFasterWhisperModel(sttFasterWhisperModelEl?.value || sttModelEl?.value || '');
  }
  return String(sttModelEl?.value || '').trim();
}

function syncSTTRuntimeModeUI() {
  const mode = String(sttModeEl?.value || 'faster_whisper').trim().toLowerCase();
  const isRemote = mode === 'remote';
  const isLMStudio = mode === 'lmstudio';
  const isFasterWhisper = mode === 'faster_whisper';
  const isLocal = mode === 'local';

  const toggle = (el, hidden) => {
    if (el) el.classList.toggle('hidden', !!hidden);
  };

  toggle(sttBaseURLWrapEl, !(isRemote || isLMStudio));
  toggle(sttModelWrapEl, !(isRemote || isLMStudio));
  toggle(sttFasterWhisperModelWrapEl, !isFasterWhisper);
  toggle(sttDeviceWrapEl, !isFasterWhisper);
  toggle(sttComputeTypeWrapEl, !isFasterWhisper);
  toggle(sttLanguageWrapEl, !isFasterWhisper);
  toggle(sttLocalCommandWrapEl, !isLocal);
  toggle(sttTimeoutWrapEl, !(isLMStudio || isFasterWhisper || isLocal));

  toggle(sttConnectionRowEl, !(isRemote || isLMStudio || isFasterWhisper));
  toggle(sttFasterWhisperRowEl, !isFasterWhisper);
  toggle(sttCommandRowEl, !(isLMStudio || isFasterWhisper || isLocal));

  if (sttModeHelpEl) {
    if (isRemote) {
      sttModeHelpEl.textContent = 'Remote OpenAI transcription uses a base URL and model.';
    } else if (isLMStudio) {
      sttModeHelpEl.textContent = 'LM Studio uses a local OpenAI-compatible endpoint, model, and optional timeout.';
    } else if (isFasterWhisper) {
      sttModeHelpEl.textContent = 'Managed faster-whisper runs locally inside Knit and uses model/device/compute settings.';
    } else {
      sttModeHelpEl.textContent = 'Local command mode shells out to your configured command with an optional timeout.';
    }
  }
  if (sttBaseURLEl) {
    sttBaseURLEl.placeholder = isLMStudio ? 'LM Studio base URL' : 'OpenAI base URL';
  }
  if (sttModelEl) {
    if (isRemote) {
      sttModelEl.placeholder = 'OpenAI STT model';
    } else if (isLMStudio) {
      sttModelEl.placeholder = 'LM Studio model';
    } else if (isFasterWhisper) {
      sttModelEl.placeholder = 'faster-whisper model';
    }
  }
  if (isFasterWhisper) {
    const normalized = normalizeFasterWhisperModel(sttFasterWhisperModelEl?.value || sttModelEl?.value || '');
    if (sttFasterWhisperModelEl) sttFasterWhisperModelEl.value = normalized;
    if (sttModelEl) sttModelEl.value = normalized;
  }
}

function syncEnhancementUIFromState() {
  if (reviewModeEl) {
    const currentMode = String(currentState?.session?.review_mode || '');
    reviewModeEl.value = currentMode;
  }
  syncLaserModeForVideo();
}

function isVideoCaptureActive() {
  const hasStream = !!(displayStream && displayStream.getVideoTracks && displayStream.getVideoTracks().length > 0);
  const screenStatus = String(currentState?.capture_sources?.screen?.status || '').toLowerCase();
  return hasStream || screenStatus === 'available';
}

function syncLaserModeForVideo() {
  if (!laserModeEnabledEl) return;
  const videoActive = isVideoCaptureActive();
  if (videoActive) {
    if (!laserModeEnabledEl.checked) {
      laserModeEnabledEl.checked = true;
      laserForcedByVideo = true;
    }
    laserModeEnabledEl.disabled = true;
    laserModeEnabledEl.title = 'Laser pointer is enabled automatically while video capture is active.';
    return;
  }
  laserModeEnabledEl.disabled = false;
  laserModeEnabledEl.title = 'Include pointer trail/highlight in note artifacts.';
  if (laserForcedByVideo) {
    laserModeEnabledEl.checked = false;
    laserForcedByVideo = false;
  }
}

function isCompanionAttached() {
  const status = String(currentState?.capture_sources?.companion?.status || '').toLowerCase();
  return status === 'available';
}

function requireCompanionFor(action) {
  if (isCompanionAttached()) {
    return true;
  }
  const msg = 'Browser companion is required to ' + action + '. Click the 🔗 icon in Feedback Note/Composer, run the snippet in target-tab DevTools, then retry.';
  logNote(msg, true);
  return false;
}

function renderCaptureGuideStatus() {
  if (!captureGuideStatusEl) return;
  const sess = currentState?.session || {};
  const sessionActive = !!sess.id;
  const sources = currentState?.capture_sources || {};
  const companion = String(sources?.companion?.status || 'unknown');
  const screen = String(sources?.screen?.status || 'unknown');
  const audio = currentState?.audio?.state || {};
  const mode = String(audio.mode || 'always_on');
  const muted = !!audio.muted;
  const paused = !!audio.paused;
  const transcriptMode = String(currentState?.transcription_mode || 'faster_whisper');
  const targetWindow = String(sess.target_window || DEFAULT_TARGET_WINDOW);
  const targetURL = String(sess.target_url || 'pending browser companion');

  const lines = [
    'Session started: ' + (sessionActive ? 'yes' : 'no'),
    'Review label: ' + targetWindow,
    'Target URL: ' + targetURL,
    'Companion attached (target app tab): ' + companion,
    'Visual capture enabled: ' + screen,
    'Audio ready: mode=' + mode + ', muted=' + muted + ', paused=' + paused,
    'Transcription mode: ' + transcriptMode,
    'Tip: use the 🔗 icon to copy companion snippet, then run it in target-tab DevTools.',
    'Tip: mouse capture comes from the target app tab after companion injection, not from this Knit tab.',
    mode === 'push_to_talk'
      ? 'Tip: push_to_talk only captures while this Knit tab is focused and Space is held. Use always_on for cross-tab feedback.'
      : 'Tip: always_on is recommended while switching between tabs/windows during review.'
  ];
  captureGuideStatusEl.textContent = lines.join('\n');
}

function renderComposerSupportStatus() {
  if (!composerSupportStateEl) return;
  const secure = !!window.isSecureContext;
  composerSupportStateEl.textContent = 'Composer popup uses window.open. Browser may block popups; allow popups for 127.0.0.1:7777. Always-on-top is browser/OS dependent.';
}

function renderPlatformRuntimeStatus() {
  if (!platformRuntimeStateEl) return;
  const runtimePlatform = currentState?.runtime_platform || {};
  const profile = currentState?.platform_profile || {};
  const summary = String(runtimePlatform.runtime_summary || '').trim();
  const hostTarget = String(runtimePlatform.host_target || '').trim();
  const installerHint = String(runtimePlatform.installer_hint || '').trim();
  const displayName = String(profile.display_name || 'Current OS').trim();
  const fallback = displayName + ': browser-first review, local web UI, and ' + (installerHint || 'portable archive') + ' packaging.';
  platformRuntimeStateEl.textContent = 'Platform runtime: ' + (summary || fallback) + (hostTarget ? ' Host target: ' + hostTarget + '.' : '');
}

function currentSessionStatus() {
  return String(currentState?.session?.status || '').trim().toLowerCase();
}

function hasLiveSession() {
  if (!currentState?.session?.id) return false;
  const status = currentSessionStatus();
  return status !== 'stopped' && status !== 'submitted';
}

function renderSessionTransportControls() {
  const hasSession = hasLiveSession();
  const captureState = String(currentState?.capture_state || 'inactive');
  const paused = captureState === 'paused';
  if (sessionPlayBtnEl) {
    sessionPlayBtnEl.textContent = '▶ Start review';
    sessionPlayBtnEl.disabled = hasSession;
    sessionPlayBtnEl.hidden = hasSession;
    sessionPlayBtnEl.title = 'Start Session';
    sessionPlayBtnEl.setAttribute('aria-label', 'Start Session');
  }
  if (sessionPauseResumeBtnEl) {
    sessionPauseResumeBtnEl.textContent = paused ? '▶ Resume' : '⏸ Pause';
    sessionPauseResumeBtnEl.disabled = !hasSession;
    sessionPauseResumeBtnEl.hidden = !hasSession;
    sessionPauseResumeBtnEl.title = paused ? 'Resume Capture' : 'Pause Capture';
    sessionPauseResumeBtnEl.setAttribute('aria-label', paused ? 'Resume Capture' : 'Pause Capture');
  }
  if (sessionStopBtnEl) {
    sessionStopBtnEl.textContent = '⏹ Stop';
    sessionStopBtnEl.disabled = !hasSession;
    sessionStopBtnEl.hidden = !hasSession;
    sessionStopBtnEl.title = 'Stop Session';
    sessionStopBtnEl.setAttribute('aria-label', 'Stop Session');
  }
}

function sensitiveBadge(label, value, tone = '') {
  return '<span class="status-pill small' + (tone ? ' ' + tone : '') + '">' + escapePreviewHTML(label) + ': ' + escapePreviewHTML(value) + '</span>';
}

function replayTypedValueStatusLabel() {
  return !!(currentState?.session?.capture_input_values) ? 'on' : 'redacted';
}

function largeMediaStatusLabel() {
  return !!allowLargeInlineMediaEl?.checked ? 'allowed' : 'ask first';
}

function currentVideoModeLabel() {
  return String(currentState?.video_mode || videoModeEl?.value || 'event_triggered').trim() || 'event_triggered';
}

function currentAudioModeLabel() {
  return String(currentState?.audio?.state?.mode || audioModeEl?.value || 'always_on').trim() || 'always_on';
}

function renderSensitiveCaptureBadges() {
  if (!sensitiveCaptureBadgesEl) return;
  sensitiveCaptureBadgesEl.innerHTML = [
    sensitiveBadge('Replay typed values', replayTypedValueStatusLabel(), currentState?.session?.capture_input_values ? 'ok' : ''),
    sensitiveBadge('Large media', largeMediaStatusLabel(), allowLargeInlineMediaEl?.checked ? 'ok' : ''),
    sensitiveBadge('Video mode', currentVideoModeLabel()),
    sensitiveBadge('Audio mode', currentAudioModeLabel())
  ].join('');
}

function disclosureStatusLabel(value) {
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

function renderDisclosureSummary(preview) {
  const disclosure = preview?.disclosure;
  if (!disclosure || typeof disclosure !== 'object') return '';
  const requestCount = Number(disclosure.request_text_count || 0);
  const actions = [];
  const typedStatus = String(disclosure.typed_values_status || '').trim();
  if (typedStatus === 'included' || typedStatus === 'mixed' || previewDeliveryOptions.redactReplayValues) {
    const label = previewDeliveryOptions.redactReplayValues ? 'Send typed values again' : 'Redact typed values for this preview';
    actions.push('<button type="button" class="secondary" onclick="togglePreviewReplayRedaction()" title="' + escapePreviewHTML(label) + '">' + escapePreviewHTML(label) + '</button>');
  }
  const videoCount = Number(disclosure.videos_sent || 0) + Number(disclosure.videos_omitted || 0);
  if (videoCount > 0 || previewDeliveryOptions.omitVideoClips) {
    const label = previewDeliveryOptions.omitVideoClips
      ? 'Send clips again'
      : (Number(disclosure.screenshots_sent || 0) > 0 ? 'Use screenshot instead of clip' : 'Omit clip for this request');
    actions.push('<button type="button" class="secondary" onclick="togglePreviewVideoDelivery()" title="' + escapePreviewHTML(label) + '">' + escapePreviewHTML(label) + '</button>');
  }
  return '<div class="preview-warning-card">' +
    '<strong>What will be sent</strong>' +
    '<div class="preview-note-text" style="margin-top:.35rem;">' +
      '<div>Destination: ' + escapePreviewHTML(String(disclosure.destination || 'local adapter')) + '</div>' +
      '<div>Request text: ' + escapePreviewHTML(String(requestCount)) + ' change request' + (requestCount === 1 ? '' : 's') + '</div>' +
      '<div>Typed values: ' + escapePreviewHTML(disclosureStatusLabel(disclosure.typed_values_status)) + '</div>' +
      '<div>Screenshots: ' + escapePreviewHTML(String(disclosure.screenshots_sent || 0)) + ' sent' + (Number(disclosure.screenshots_omitted || 0) > 0 ? ', ' + escapePreviewHTML(String(disclosure.screenshots_omitted || 0)) + ' omitted' : '') + '</div>' +
      '<div>Video clips: ' + escapePreviewHTML(String(disclosure.videos_sent || 0)) + ' sent' + (Number(disclosure.videos_omitted || 0) > 0 ? ', ' + escapePreviewHTML(String(disclosure.videos_omitted || 0)) + ' omitted' : '') + '</div>' +
      '<div>Audio clips: ' + escapePreviewHTML(String(disclosure.audio_sent || 0)) + ' sent' + (Number(disclosure.audio_omitted || 0) > 0 ? ', ' + escapePreviewHTML(String(disclosure.audio_omitted || 0)) + ' omitted' : '') + '</div>' +
    '</div>' +
    (actions.length ? '<div class="sub-actions" style="margin-top:.55rem;">' + actions.join('') + '</div>' : '') +
  '</div>';
}

async function togglePreviewReplayRedaction() {
  previewDeliveryOptions.redactReplayValues = !previewDeliveryOptions.redactReplayValues;
  await previewPayload();
}

async function togglePreviewVideoDelivery() {
  previewDeliveryOptions.omitVideoClips = !previewDeliveryOptions.omitVideoClips;
  await previewPayload();
}

function previewVideoEventOmitted(eventID) {
  const id = String(eventID || '').trim();
  return !!id && Array.isArray(previewDeliveryOptions.omitVideoEventIDs) && previewDeliveryOptions.omitVideoEventIDs.includes(id);
}

function setPreviewVideoEventOmitted(eventID, omitted) {
  const id = String(eventID || '').trim();
  const current = Array.isArray(previewDeliveryOptions.omitVideoEventIDs) ? previewDeliveryOptions.omitVideoEventIDs.slice() : [];
  const next = current.filter(item => item !== id);
  if (omitted && id) {
    next.push(id);
  }
  previewDeliveryOptions.omitVideoEventIDs = next;
}

async function togglePreviewVideoEventOmission(eventID) {
  const id = String(eventID || '').trim();
  if (!id) return;
  setPreviewVideoEventOmitted(id, !previewVideoEventOmitted(id));
  await previewPayload();
}

function oversizedVideoPreviewNotes(preview) {
  const notes = Array.isArray(preview?.notes) ? preview.notes : [];
  return notes.filter(note => previewNoteNeedsClipResize(note));
}

function renderOversizedVideoWarningActions(preview) {
  const notes = oversizedVideoPreviewNotes(preview);
  if (!notes.length) return '';
  const items = notes.map((note) => {
    const eventID = String(note?.event_id || '').trim();
    const useSnapshotLabel = previewVideoEventOmitted(eventID)
      ? 'Send clip again'
      : (note?.has_screenshot ? 'Use snapshot instead' : 'Omit clip for this request');
    return '<div class="preview-note-card" style="margin-top:.6rem;">' +
      '<div class="preview-note-header"><strong>' + escapePreviewHTML(eventID || 'change request') + '</strong><span class="empty-tone">' + escapePreviewHTML(formatMediaSize(note?.video_size_bytes || 0) + ' over ' + formatMediaSize(note?.video_send_limit_bytes || 0)) + '</span></div>' +
      '<div class="preview-note-text">' + escapePreviewHTML(String(note?.video_transmission_note || 'This clip is too large to send with the current inline media setting.')) + '</div>' +
      '<div class="sub-actions" style="margin-top:.55rem;">' +
      '<button type="button" class="secondary" ' + (clipResizeInFlight.has(eventID) ? 'disabled ' : '') + 'onclick="fitPreviewClipToSendLimit(\'' + escapePreviewHTML(eventID) + '\')" title="Make clip smaller to send">' + escapePreviewHTML(clipResizeInFlight.has(eventID) ? 'Making clip smaller…' : 'Make clip smaller to send') + '</button>' +
      '<button type="button" class="secondary" onclick="togglePreviewVideoEventOmission(\'' + escapePreviewHTML(eventID) + '\')" title="' + escapePreviewHTML(useSnapshotLabel) + '">' + escapePreviewHTML(useSnapshotLabel) + '</button>' +
      '</div>' +
      '</div>';
  }).join('');
  return '<div class="preview-warning-card">' +
    '<strong>Large clip needs a decision</strong>' +
    '<div class="preview-note-text" style="margin-top:.35rem;">Choose how to handle the affected request before you submit.</div>' +
    items +
  '</div>';
}

function renderSubmitAttempts() {
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];
  if (!attempts.length) {
    openSubmitAttemptRawJSONIDs = new Set();
    submitAttemptRawJSONScrollTopByID = new Map();
    submitResultEl.textContent = 'No runs yet. Recent agent requests will appear here.';
    return;
  }
  snapshotSubmitAttemptOpenState();
  submitResultEl.innerHTML = '<div class="flow-stack">' + attempts.slice(0, 8).map(renderSubmitAttemptHistoryCard).join('') + '</div>';
  restoreSubmitAttemptOpenState();
  void hydrateSubmitAttemptOutputs();
}

function formatAttemptClock(value) {
  const ms = Date.parse(String(value || ''));
  if (!Number.isFinite(ms) || ms <= 0) return '';
  return new Date(ms).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', second: '2-digit' });
}

function formatAgeShort(value) {
  const ms = Date.parse(String(value || ''));
  if (!Number.isFinite(ms) || ms <= 0) return '';
  const delta = Math.max(0, Math.round((Date.now() - ms) / 1000));
  if (delta < 60) return delta + 's ago';
  const minutes = Math.floor(delta / 60);
  const seconds = delta % 60;
  if (minutes < 60) return minutes + 'm ' + String(seconds).padStart(2, '0') + 's ago';
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  return hours + 'h ' + String(remMinutes).padStart(2, '0') + 'm ago';
}

function requestPreviewText(attempt) {
  if (!attempt || typeof attempt !== 'object') return '';
  return String(attempt.request_preview || '').trim();
}

function requestPreviewListItem(attempt) {
  const preview = requestPreviewText(attempt);
  if (!preview) return '';
  return '<li><strong>Request:</strong> ' + escapePreviewHTML(preview) + '</li>';
}

function submitAttemptWorkspaceText(attempt) {
  if (!attempt || typeof attempt !== 'object') return '';
  return String(attempt.workdir_used || '').trim();
}

function submitAttemptWorkspaceListItem(attempt) {
  const workspace = submitAttemptWorkspaceText(attempt);
  if (!workspace) return '';
  return '<li><strong>Workspace used:</strong> ' + escapePreviewHTML(workspace) + '</li>';
}

function renderCancelSubmitAttemptButton(attempt, label = 'Stop request') {
  const attemptID = String(attempt?.attempt_id || '').trim();
  if (!attemptID || !isCancelableSubmitStatus(attempt?.status)) return '';
  return '<button type="button" class="secondary" onclick="cancelSubmitAttempt(\'' + escapePreviewHTML(attemptID) + '\')" title="' + escapePreviewHTML(label) + '" style="padding:.38rem .72rem;white-space:nowrap;">' + escapePreviewHTML(label) + '</button>';
}

function providerDestinationLabel(provider) {
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

function snapshotSubmitAttemptOpenState() {
  if (!submitResultEl) return;
  const nextRawJSONIDs = new Set();
  const nextRawJSONScrollTopByID = new Map();
  submitResultEl.querySelectorAll('[data-submit-attempt-id]').forEach(card => {
    const attemptID = String(card.getAttribute('data-submit-attempt-id') || '').trim();
    if (!attemptID) return;
    const rawJSON = card.querySelector('[data-submit-attempt-raw-json]');
    if (!rawJSON?.open) return;
    nextRawJSONIDs.add(attemptID);
    const rawJSONPre = rawJSON.querySelector('[data-submit-attempt-raw-json-body]');
    if (rawJSONPre && rawJSONPre.scrollTop > 0) {
      nextRawJSONScrollTopByID.set(attemptID, rawJSONPre.scrollTop);
    }
  });
  openSubmitAttemptRawJSONIDs = nextRawJSONIDs;
  submitAttemptRawJSONScrollTopByID = nextRawJSONScrollTopByID;
}

function restoreSubmitAttemptOpenState() {
  if (!submitResultEl) return;
  submitResultEl.querySelectorAll('[data-submit-attempt-id]').forEach(card => {
    const attemptID = String(card.getAttribute('data-submit-attempt-id') || '').trim();
    if (!attemptID) return;
    const rawJSON = card.querySelector('[data-submit-attempt-raw-json]');
    if (rawJSON) {
      rawJSON.open = openSubmitAttemptRawJSONIDs.has(attemptID);
      const rawJSONPre = rawJSON.querySelector('[data-submit-attempt-raw-json-body]');
      const scrollTop = Number(submitAttemptRawJSONScrollTopByID.get(attemptID) || 0);
      if (rawJSON.open && rawJSONPre && scrollTop > 0) {
        rawJSONPre.scrollTop = scrollTop;
      }
    }
  });
}

function submitAttemptOutputText(attempt) {
  if (!attempt || typeof attempt !== 'object') return '';
  const attemptID = String(attempt.attempt_id || '').trim();
  if (attemptID) {
    const cached = submitAttemptOutputPreviewByID.get(attemptID);
    if (cached?.status === 'ready') return String(cached.text || '').trim();
    if (cached?.status === 'empty') return 'Adapter log is empty.';
    if (cached?.status === 'loading') return 'Loading adapter output...';
  }
  const status = String(attempt.status || '');
  const error = String(attempt.error || '').trim();
  if (error) return 'Error: ' + error;
  if (looksLikeLocalAttemptLogRef(attemptLogRef(attempt))) return 'Loading adapter output...';
  const ref = String(attempt.ref || '').trim();
  if (ref) return 'Result reference: ' + ref;
  const runID = String(attempt.run_id || '').trim();
  if (runID) return 'Run ID: ' + runID;
  const note = String(attempt.note || '').trim();
  if (note && status !== 'in_progress') return note;
  const postSubmit = attempt.post_submit;
  if (postSubmit && typeof postSubmit === 'object') {
    const rebuildStatus = String(postSubmit?.rebuild?.status || '').trim();
    const verifyStatus = String(postSubmit?.verify?.status || '').trim();
    if (rebuildStatus || verifyStatus) {
      return 'Post-submit: rebuild ' + (rebuildStatus || 'not run') + ', verify ' + (verifyStatus || 'not run');
    }
  }
  return '';
}

function submitAttemptOutputHasPreview(attempt) {
  if (!attempt || typeof attempt !== 'object') return false;
  const attemptID = String(attempt.attempt_id || '').trim();
  if (!attemptID) return false;
  const cached = submitAttemptOutputPreviewByID.get(attemptID);
  return cached?.status === 'ready' || cached?.status === 'empty' || cached?.status === 'loading';
}

function normalizeSubmitAttemptOutputPreview(text, truncated, truncatedHead) {
  const compact = String(text || '').replace(/\r\n/g, '\n').trim();
  if (!compact) return '';
  let preview = compact;
  if (truncatedHead) preview = '…\n' + preview;
  if (truncated) preview = preview + '\n…';
  return preview;
}

async function hydrateSubmitAttemptOutputs() {
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];
  const visible = attempts.slice(0, 8);
  for (const attempt of visible) {
    if (!attempt || !looksLikeLocalAttemptLogRef(attemptLogRef(attempt))) continue;
    const attemptID = String(attempt.attempt_id || '').trim();
    if (!attemptID) continue;
    const cached = submitAttemptOutputPreviewByID.get(attemptID);
    if (cached?.status === 'ready' || cached?.status === 'empty') continue;
    if (submitAttemptOutputPreviewInflight.has(attemptID)) continue;
    submitAttemptOutputPreviewInflight.add(attemptID);
    submitAttemptOutputPreviewByID.set(attemptID, { status: 'loading', text: '' });
    try {
      const status = String(attempt?.status || '').trim();
      const useTail = status !== 'in_progress' && status !== 'queued';
      const path = '/api/session/attempt/log?attempt_id=' + encodeURIComponent(attemptID) +
        '&offset=0&limit=24000' + (useTail ? '&tail=1' : '');
      const res = await fetch(path, { headers: authHeaders(false) });
      const txt = await res.text();
      if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
      const data = txt ? JSON.parse(txt) : {};
      const chunk = String(data.chunk || '');
      const preview = normalizeSubmitAttemptOutputPreview(chunk, !data.eof, !!data.truncated_head);
      submitAttemptOutputPreviewByID.set(attemptID, {
        status: preview ? 'ready' : 'empty',
        text: preview,
      });
    } catch (_) {
      submitAttemptOutputPreviewByID.delete(attemptID);
    } finally {
      submitAttemptOutputPreviewInflight.delete(attemptID);
    }
  }
  if (submitResultEl) {
    snapshotSubmitAttemptOpenState();
    submitResultEl.innerHTML = '<div class="flow-stack">' + visible.map(renderSubmitAttemptHistoryCard).join('') + '</div>';
    restoreSubmitAttemptOpenState();
  }
}

function renderSubmitAttemptOutput(attempt) {
  const output = submitAttemptOutputText(attempt);
  if (!output) {
    return '<li><strong>Output:</strong> Still running, queued, or waiting for a result reference.</li>';
  }
  if (!submitAttemptOutputHasPreview(attempt)) {
    return '<li><strong>Output:</strong> ' + escapePreviewHTML(output) + '</li>';
  }
  const split = splitLiveAgentOutputForDisplay(output);
  const work = split.work || 'No work log captured for this run.';
  const commentary = split.commentary || 'No agent commentary captured for this run.';
  return '<li><strong>Output:</strong>' +
    '<div class="flow-stack" style="margin-top:.35rem;">' +
      '<div>' +
        '<div class="helper" style="margin-bottom:.3rem;"><strong>Work log</strong></div>' +
        '<pre style="max-height:180px;overflow:auto;white-space:pre-wrap;">' + escapePreviewHTML(work) + '</pre>' +
      '</div>' +
      '<div>' +
        '<div class="helper" style="margin-bottom:.3rem;"><strong>Agent commentary</strong></div>' +
        '<pre style="max-height:180px;overflow:auto;white-space:pre-wrap;">' + escapePreviewHTML(commentary) + '</pre>' +
      '</div>' +
    '</div>' +
  '</li>';
}

function submitAttemptLastUpdate(attempt) {
  const timeline = Array.isArray(attempt?.timeline) ? attempt.timeline : [];
  const last = timeline.length ? timeline[timeline.length - 1] : null;
  if (!last) return '';
  const parts = [];
  if (last.status) parts.push(String(last.status));
  if (last.note) parts.push(String(last.note));
  return parts.join(' - ');
}

function submitAttemptWhen(attempt) {
  const completedAt = String(attempt?.completed_at || '').trim();
  if (completedAt) return formatAttemptClock(completedAt) || formatAgeShort(completedAt);
  const startedAt = String(attempt?.started_at || '').trim();
  if (startedAt) return formatAttemptClock(startedAt) || formatAgeShort(startedAt);
  const enqueuedAt = String(attempt?.enqueued_at || '').trim();
  if (enqueuedAt) return formatAttemptClock(enqueuedAt) || formatAgeShort(enqueuedAt);
  return '';
}

function renderSubmitAttemptHistoryCard(attempt) {
  const id = String(attempt?.attempt_id || 'attempt');
  const status = String(attempt?.status || 'unknown');
  const provider = String(attempt?.provider || 'agent');
  const destination = providerDestinationLabel(provider);
  const mode = String(attempt?.mode || 'series');
  const when = submitAttemptWhen(attempt);
  const request = requestPreviewText(attempt) || 'No request preview captured.';
  const output = submitAttemptOutputText(attempt);
  const retry = Number(attempt?.retry_count || 0);
  const maxAttempts = Math.max(1, Number(attempt?.max_attempts || 1));
  const wait = Number(attempt?.queue_wait_ms || 0);
  const lastUpdate = submitAttemptLastUpdate(attempt);
  const rawJSON = escapePreviewHTML(JSON.stringify(attempt, null, 2));
  return '<div class="status-card" data-submit-attempt-id="' + escapePreviewHTML(id) + '">' +
    '<div class="row" style="justify-content:space-between;align-items:flex-start;gap:.75rem;">' +
      '<div><strong>' + escapePreviewHTML(destination) + '</strong><div class="helper">Status: ' + escapePreviewHTML(status) + '</div></div>' +
      '<div style="text-align:right;">' +
        '<div class="helper">' + escapePreviewHTML(when || 'recently') + '</div>' +
        (renderCancelSubmitAttemptButton(attempt, status === 'in_progress' ? 'Stop request' : 'Remove from queue') ? '<div style="margin-top:.4rem;">' + renderCancelSubmitAttemptButton(attempt, status === 'in_progress' ? 'Stop request' : 'Remove from queue') + '</div>' : '') +
      '</div>' +
    '</div>' +
    '<ul class="status-list">' +
      '<li><strong>Request:</strong> ' + escapePreviewHTML(request) + '</li>' +
      submitAttemptWorkspaceListItem(attempt) +
      renderSubmitAttemptOutput(attempt) +
      '<li><strong>Details:</strong> ' + escapePreviewHTML(id) + ' • ' + escapePreviewHTML(mode) + ' mode • retry ' + retry + '/' + maxAttempts + (wait > 0 ? ' • waited ' + wait + 'ms' : '') + '</li>' +
      (lastUpdate ? '<li><strong>Latest update:</strong> ' + escapePreviewHTML(lastUpdate) + '</li>' : '') +
    '</ul>' +
    '<details data-submit-attempt-raw-json style="margin-top:.45rem;">' +
      '<summary>Raw JSON</summary>' +
      '<pre data-submit-attempt-raw-json-body style="margin-top:.45rem;max-height:180px;overflow:auto;white-space:pre-wrap;">' + rawJSON + '</pre>' +
    '</details>' +
  '</div>';
}

function renderQueueStateCard(mode, running, queued, postSubmitRunning, attempts) {
  if (!queueStateEl) return;
  const runningAttempts = attempts.filter(a => String(a?.status || '') === 'in_progress').slice(0, 3);
  const waitingAttempts = attempts.filter(a => {
    const status = String(a?.status || '');
    return status === 'queued' || status === 'deferred_offline' || status === 'retry_wait';
  }).slice(0, 3);
  let html = '<div><strong>' + running + '</strong> running • <strong>' + queued + '</strong> waiting • ' + escapePreviewHTML(mode) + ' mode' +
    (postSubmitRunning ? ' • rebuild/check running' : '') + '</div>';
  if (runningAttempts.length) {
    html += '<ul class="status-list">' + runningAttempts.map(a =>
      '<li><strong>Running:</strong> ' + escapePreviewHTML(providerDestinationLabel(a.provider || 'agent')) +
      ' • ' + escapePreviewHTML(String(a.attempt_id || 'attempt')) +
      (a.started_at ? ' • started ' + escapePreviewHTML(formatAttemptClock(a.started_at)) + ' (' + escapePreviewHTML(formatAgeShort(a.started_at)) + ')' : '') +
      (renderCancelSubmitAttemptButton(a) ? '<div style="margin-top:.45rem;">' + renderCancelSubmitAttemptButton(a) + '</div>' : '') +
      (requestPreviewText(a) ? '<div class="empty-tone">' + escapePreviewHTML(requestPreviewText(a)) + '</div>' : '') +
      '</li>'
    ).join('') + '</ul>';
  } else if (waitingAttempts.length) {
    html += '<ul class="status-list"><li>No request is running right now.</li></ul>';
  }
  if (waitingAttempts.length) {
    html += '<ul class="status-list">' + waitingAttempts.map(a =>
      '<li><strong>Waiting:</strong> ' + escapePreviewHTML(providerDestinationLabel(a.provider || 'agent')) +
      ' • ' + escapePreviewHTML(String(a.attempt_id || 'attempt')) +
      ' • queued ' + escapePreviewHTML(formatAgeShort(a.enqueued_at) || 'recently') +
      (renderCancelSubmitAttemptButton(a) ? '<div style="margin-top:.45rem;">' + renderCancelSubmitAttemptButton(a, 'Remove from queue') + '</div>' : '') +
      (requestPreviewText(a) ? '<div class="empty-tone">' + escapePreviewHTML(requestPreviewText(a)) + '</div>' : '') +
      '</li>'
    ).join('') + '</ul>';
  } else if (running <= 0 && queued <= 0) {
    html += '<ul class="status-list"><li>No queued submissions.</li></ul>';
  }
  queueStateEl.innerHTML = html;
}

function renderSubmissionStateCard(running, queued, attempts) {
  if (!submitStateEl) return;
  const runningAttempt = attempts.find(a => String(a?.status || '') === 'in_progress') || null;
  if (runningAttempt) {
    const note = String(runningAttempt.note || 'Sending to your coding agent');
    const startedAt = runningAttempt.started_at ? formatAttemptClock(runningAttempt.started_at) : '';
    submitStateEl.innerHTML = '<div><span class="spinner"></span>' + escapePreviewHTML(note) + '</div>' +
      '<ul class="status-list"><li><strong>Destination:</strong> ' + escapePreviewHTML(providerDestinationLabel(runningAttempt.provider || 'agent')) + '</li>' +
      '<li><strong>Started:</strong> ' + escapePreviewHTML(startedAt || 'just now') +
      (runningAttempt.started_at ? ' • ' + escapePreviewHTML(formatAgeShort(runningAttempt.started_at)) : '') + '</li>' +
      requestPreviewListItem(runningAttempt) +
      submitAttemptWorkspaceListItem(runningAttempt) +
      (renderCancelSubmitAttemptButton(runningAttempt) ? '<li>' + renderCancelSubmitAttemptButton(runningAttempt) + '</li>' : '') +
      (queued > 0 ? '<li><strong>Next up:</strong> ' + queued + ' waiting in queue.</li>' : '') +
      '</ul>';
    submitStateEl.style.color = '#1c7c74';
    return;
  }
  submitStateEl.textContent = 'No active run.';
  submitStateEl.style.color = '#6a7383';
}

function earliestRunningSubmitStartMS() {
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];
  let earliest = 0;
  for (const a of attempts) {
    if (!a || a.status !== 'in_progress' || !a.started_at) continue;
    const ms = Date.parse(String(a.started_at));
    if (!Number.isFinite(ms) || ms <= 0) continue;
    if (earliest === 0 || ms < earliest) earliest = ms;
  }
  return earliest;
}

function renderRunningSubmitTimer() {
  if (!submitStateEl) return;
  const queue = currentState?.submit_queue || {};
  const running = Number(queue.running || 0);
  if (running <= 0) return;
  const queued = Number(queue.queued || 0);
  const startMS = earliestRunningSubmitStartMS() || Date.now();
  const sec = Math.max(1, Math.floor((Date.now() - startMS) / 1000));
  const runningAttempt = activeRunningSubmitAttempt();
  const note = String(runningAttempt?.note || 'Sending to your coding agent');
  const startedAt = runningAttempt?.started_at ? formatAttemptClock(runningAttempt.started_at) : '';
  submitStateEl.innerHTML = '<div><span class="spinner"></span>' + escapePreviewHTML(note) + ' (' + sec + 's)</div>' +
    '<ul class="status-list"><li><strong>Destination:</strong> ' + escapePreviewHTML(providerDestinationLabel(runningAttempt?.provider || 'agent')) + '</li>' +
    '<li><strong>Started:</strong> ' + escapePreviewHTML(startedAt || 'just now') +
    (runningAttempt?.started_at ? ' • ' + escapePreviewHTML(formatAgeShort(runningAttempt.started_at)) : '') + '</li>' +
    requestPreviewListItem(runningAttempt) +
    submitAttemptWorkspaceListItem(runningAttempt) +
    (renderCancelSubmitAttemptButton(runningAttempt) ? '<li>' + renderCancelSubmitAttemptButton(runningAttempt) + '</li>' : '') +
    (queued > 0 ? '<li><strong>Next up:</strong> ' + queued + ' waiting in queue.</li>' : '') +
    '</ul>';
  submitStateEl.style.color = '#1c7c74';
}

function ensureSubmitTimer() {
  if (submitTimer) return;
  submitTimer = setInterval(() => {
    renderRunningSubmitTimer();
  }, 1000);
}

function stopSubmitTimer() {
  if (!submitTimer) return;
  clearInterval(submitTimer);
  submitTimer = null;
}

function activeRunningSubmitAttempt() {
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];
  for (const a of attempts) {
    if (!a) continue;
    if (String(a.status || '') === 'in_progress') return a;
  }
  return null;
}

function findSubmitAttemptById(attemptID) {
  const id = String(attemptID || '').trim();
  if (!id) return null;
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];
  for (const a of attempts) {
    if (!a) continue;
    if (String(a.attempt_id || '') === id) return a;
  }
  return null;
}

function looksLikeLocalAttemptLogRef(ref) {
  const value = String(ref || '').trim();
  if (!value) return false;
  const isAbsolute = value.startsWith('/') || /^[A-Za-z]:[\\/]/.test(value);
  if (!isAbsolute) return false;
  return /(?:^|[\\/])knit-codex-[^\\/]*\.log[^\\/]*$/i.test(value);
}

function attemptLogRef(attempt) {
  if (!attempt) return '';
  return String(attempt.execution_ref || attempt.ref || '');
}

function trimLiveOutputBuffer(text, maxChars = 80000) {
  const value = String(text || '').replace(/\r\n/g, '\n');
  if (value.length <= maxChars) return value;
  const sliced = value.slice(-maxChars);
  const newline = sliced.indexOf('\n');
  return newline >= 0 ? sliced.slice(newline + 1) : sliced;
}

function trimLiveOutputSection(text, maxChars = 32000) {
  const value = String(text || '').replace(/\r\n/g, '\n').trim();
  if (!value) return '';
  if (value.length <= maxChars) return value;
  const sliced = value.slice(-maxChars);
  const newline = sliced.indexOf('\n');
  const body = newline >= 0 ? sliced.slice(newline + 1) : sliced;
  return '…\n' + body;
}

function isLiveOutputWorkMarker(line) {
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

function isLikelyLivePayloadLine(line) {
  const text = String(line || '');
  const trimmed = text.trim();
  if (!trimmed) return false;
  if (trimmed.startsWith('{"created":') || trimmed.includes('"inline_data_url"')) return true;
  if (trimmed.includes('data:image/') || trimmed.includes('data:video/') || trimmed.includes('data:audio/')) return true;
  if (text.length > 4000) return true;
  return /^[A-Za-z0-9+/]{256,}={0,2}$/.test(trimmed);
}

function isLikelyLiveCommentaryLine(line) {
  const trimmed = String(line || '').trim();
  if (!trimmed) return false;
  return /^(I('|’)m|I am|I('|’)ll|I will|I have|I found|I can|Next I|The current |This is |That means|I’m|I’ll)/.test(trimmed);
}

function splitLiveAgentOutputForDisplay(raw) {
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
      if (trimmed === 'codex' || isLiveOutputWorkMarker(trimmed)) {
        omittingPrompt = false;
        pushPromptNotice();
      } else {
        if (trimmed === 'Canonical payload JSON:' || isLikelyLivePayloadLine(line)) {
          pushPayloadNotice();
        }
        continue;
      }
    }
    if (trimmed === 'codex') {
      mode = 'commentary';
      continue;
    }
    if (isLikelyLivePayloadLine(line)) {
      pushPayloadNotice();
      continue;
    }
    if (isLiveOutputWorkMarker(trimmed)) {
      mode = 'work';
      if (trimmed === 'exec') continue;
      work.push(line);
      continue;
    }
    if (mode === 'commentary' || isLikelyLiveCommentaryLine(line)) {
      commentary.push(line);
      continue;
    }
    work.push(line);
  }
  if (omittingPrompt) pushPromptNotice();
  return {
    work: trimLiveOutputSection(work.join('\n')),
    commentary: trimLiveOutputSection(commentary.join('\n')),
  };
}

function renderLiveAgentOutput() {
  if (!liveSubmitLogEl) return;
  const split = splitLiveAgentOutputForDisplay(liveLogRawText);
  liveSubmitLogEl.textContent = split.work || 'No live work log yet. Work activity appears here after the adapter starts writing logs.';
  liveSubmitLogEl.scrollTop = liveSubmitLogEl.scrollHeight;
  if (liveSubmitCommentaryEl) {
    liveSubmitCommentaryEl.textContent = split.commentary || 'No agent commentary yet. Plain-language progress updates appear here when the agent explains what it is doing.';
    liveSubmitCommentaryEl.scrollTop = liveSubmitCommentaryEl.scrollHeight;
  }
}

function activeSubmitAttemptForLog() {
  const runningAttempt = activeRunningSubmitAttempt();
  if (runningAttempt && looksLikeLocalAttemptLogRef(attemptLogRef(runningAttempt))) return runningAttempt;
  if (liveLogAttemptId && liveLogCompletedForAttempt !== liveLogAttemptId) {
    const latestAttempt = findSubmitAttemptById(liveLogAttemptId);
    if (latestAttempt && looksLikeLocalAttemptLogRef(attemptLogRef(latestAttempt))) return latestAttempt;
  }
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];
  const latestLoggedAttempt = attempts.find(a => {
    if (!a) return false;
    const attemptId = String(a.attempt_id || '').trim();
    if (!attemptId || attemptId === liveLogCompletedForAttempt) return false;
    if (String(a.status || '') === 'queued') return false;
    return looksLikeLocalAttemptLogRef(attemptLogRef(a));
  });
  return latestLoggedAttempt || null;
}

function hasOpenableSubmitLog() {
  return !!activeSubmitAttemptForLog();
}

async function refreshActiveSubmitLog() {
  if (!liveSubmitLogEl) return;
  const logAttempt = activeSubmitAttemptForLog();
  if (!logAttempt) return;

  const attemptId = String(logAttempt.attempt_id || '');
  if (!attemptId) return;
  const executionRef = attemptLogRef(logAttempt);
  if (!executionRef) {
    if (liveLogAttemptId !== attemptId) {
      liveLogAttemptId = attemptId;
      liveLogOffset = 0;
      liveLogRawText = '[' + attemptId + '] waiting for adapter log path...';
      liveLogCompletedForAttempt = '';
      liveLogUnavailableForAttempt = '';
      renderLiveAgentOutput();
    }
    return;
  }

  if (!looksLikeLocalAttemptLogRef(executionRef)) {
    if (liveLogUnavailableForAttempt !== attemptId) {
      liveLogAttemptId = attemptId;
      liveLogOffset = 0;
      liveLogRawText = '[' + attemptId + '] live log unavailable for this adapter.';
      liveLogCompletedForAttempt = attemptId;
      liveLogUnavailableForAttempt = attemptId;
      renderLiveAgentOutput();
    }
    return;
  }

  if (liveLogAttemptId !== attemptId) {
    liveLogAttemptId = attemptId;
    liveLogOffset = 0;
    liveLogRawText = '[' + attemptId + '] streaming adapter output...\n';
    liveLogCompletedForAttempt = '';
    liveLogUnavailableForAttempt = '';
    renderLiveAgentOutput();
  }

  try {
    const path = '/api/session/attempt/log?attempt_id=' + encodeURIComponent(attemptId) +
      '&offset=' + encodeURIComponent(String(liveLogOffset)) +
      '&limit=12000';
    const res = await fetch(path, { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    const chunk = String(data.chunk || '');
    if (chunk) {
      liveLogRawText = trimLiveOutputBuffer(liveLogRawText + chunk);
      renderLiveAgentOutput();
    }
    const nextOffset = Number(data.next_offset || liveLogOffset);
    if (Number.isFinite(nextOffset) && nextOffset >= 0) {
      liveLogOffset = nextOffset;
    }
    const status = String(data.status || '');
    if (status !== 'in_progress' && data.eof && liveLogCompletedForAttempt !== attemptId) {
      liveLogCompletedForAttempt = attemptId;
      liveLogRawText = trimLiveOutputBuffer(liveLogRawText + '\n[' + attemptId + '] stream complete.\n');
      renderLiveAgentOutput();
    }
  } catch (err) {
    const message = String(err?.message || '');
    if (message.includes('submission reference is not') || message.includes('last submission reference is not')) {
      liveLogCompletedForAttempt = attemptId;
      liveLogUnavailableForAttempt = attemptId;
      liveLogRawText = '[' + attemptId + '] live log unavailable for this adapter.';
      renderLiveAgentOutput();
      return;
    }
    liveLogRawText = trimLiveOutputBuffer(liveLogRawText + '\n[' + attemptId + '] stream error: ' + err.message + '\n');
    renderLiveAgentOutput();
  }
}

function renderSubmitControls() {
  const sess = currentState?.session || null;
  const feedbackCount = Array.isArray(sess?.feedback) ? sess.feedback.length : 0;
  const queue = currentState?.submit_queue || {};
  const mode = String(queue.mode || 'series');
  const running = Number(queue.running || 0);
  const queued = Number(queue.queued || 0);
  const postSubmitRunning = !!queue.post_submit_running;
  const attempts = Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : [];

  if (deliveryBadgeEl) {
    if (running > 0 || queued > 0) {
      deliveryBadgeEl.textContent = 'Delivery active';
      deliveryBadgeEl.style.color = '#1c7c74';
    } else if (postSubmitRunning) {
      deliveryBadgeEl.textContent = 'Post-submit check running';
      deliveryBadgeEl.style.color = '#1c7c74';
    } else if (!sess) {
      deliveryBadgeEl.textContent = 'Start a review session';
      deliveryBadgeEl.style.color = '#6a7383';
    } else if (feedbackCount === 0) {
      deliveryBadgeEl.textContent = 'Capture a note to continue';
      deliveryBadgeEl.style.color = '#6a7383';
    } else {
      deliveryBadgeEl.textContent = 'Notes ready to preview or submit';
      deliveryBadgeEl.style.color = '#1f8f63';
    }
  }

  renderQueueStateCard(mode, running, queued, postSubmitRunning, attempts);
  renderSubmissionStateCard(running, queued, attempts);
  if (running > 0) {
    if (!submitInFlight) {
      renderRunningSubmitTimer();
    }
    ensureSubmitTimer();
  } else {
    stopSubmitTimer();
  }

  const recordingActive = isNoteRecordingActive();
  if (previewBtnEl) previewBtnEl.disabled = recordingActive || !sess || feedbackCount === 0;
  if (openLogBtnEl) openLogBtnEl.disabled = !hasOpenableSubmitLog();
  if (submitBtnEl) submitBtnEl.disabled = recordingActive || submitInFlight || !sess || feedbackCount === 0;
}

async function cancelSubmitAttempt(attemptID) {
  const id = String(attemptID || '').trim();
  const attempt = findSubmitAttemptById(id);
  if (!id || !attempt || !isCancelableSubmitStatus(attempt.status)) return;
  const destination = providerDestinationLabel(attempt.provider || 'agent');
  const prompt = String(attempt.status || '') === 'in_progress'
    ? 'Stop the running request to ' + destination + '?'
    : 'Remove the queued request to ' + destination + '?';
  if (!window.confirm(prompt)) return;
  try {
    const data = await post('/api/session/attempt/cancel', { attempt_id: id });
    currentState = currentState || {};
    currentState.submit_queue = data.submit_queue_state || currentState.submit_queue;
    currentState.submit_attempts = Array.isArray(data.submit_attempts) ? data.submit_attempts : currentState.submit_attempts;
    notifySubmitAttemptTransitions(currentState.submit_attempts);
    renderSubmitAttempts();
    renderSubmitControls();
    hydrateSubmitAttemptOutputs();
    showToast(submitAttemptToastMessage(data.attempt || attempt));
  } catch (err) {
    showToast('Could not stop request: ' + err.message, true);
  }
}

function notifySubmitRecoveryNotices(notices) {
  const list = Array.isArray(notices) ? notices : [];
  list.forEach(note => {
    const message = String(note || '').trim();
    if (!message || seenSubmitRecoveryNotices.has(message)) return;
    seenSubmitRecoveryNotices.add(message);
    logNote(message, true);
    showToast(message, true);
  });
}

function setSubmitProgress(active, msg='') {
  submitInFlight = !!active;
  renderSubmitControls();
  if (!submitStateEl) return;
  if (active) {
    submitStateEl.innerHTML = '<span class="spinner"></span>' + (msg || 'Sending your request...');
    ensureSubmitTimer();
  } else {
    submitStateEl.textContent = msg || '';
    if (Number(currentState?.submit_queue?.running || 0) <= 0) {
      stopSubmitTimer();
    }
  }
  submitStateEl.style.color = '#1c7c74';
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

function renderExtensionPairings() {
  if (!extensionPairingListEl) return;
  const pairings = Array.isArray(currentState?.extension_pairings) ? currentState.extension_pairings : [];
  if (!pairings.length) {
    extensionPairingListEl.innerHTML = 'No paired extensions.';
    return;
  }
  extensionPairingListEl.innerHTML = pairings.map(item => {
    const revoked = !!item?.revoked_at;
    const lastUsed = item?.last_used_at ? ('Last used ' + new Date(item.last_used_at).toLocaleString()) : 'Not used yet';
    const label = [item?.name || item?.id || 'browser extension', item?.browser || '', item?.platform || ''].filter(Boolean).join(' · ');
    return '<div class="row" style="justify-content:space-between;align-items:center;margin:.3rem 0;">' +
      '<span>' + escapePreviewHTML(label) + ' <span class="small" style="color:' + (revoked ? '#c34f4f' : '#6a7383') + ';">' + escapePreviewHTML(revoked ? 'revoked' : lastUsed) + '</span></span>' +
      (revoked ? '' : '<button class="danger" onclick="revokeExtensionPairing(\'' + escapePreviewHTML(String(item.id || '')) + '\')" title="Revoke extension pairing">Revoke</button>') +
      '</div>';
  }).join('');
}

async function startExtensionPairing() {
  try {
    const data = await post('/api/extension/pair/start', { name: 'Chromium Popup', browser: 'chromium' });
    if (extensionPairingCodeStateEl) {
      const expires = data?.expires_at ? (' Expires ' + new Date(data.expires_at).toLocaleTimeString()) : '';
      extensionPairingCodeStateEl.textContent = String(data?.pairing_code || 'No code issued.') + expires;
    }
    showToast('Extension pairing code ready');
    logNote('Browser extension pairing code generated. Open the extension popup and enter the code before it expires.');
  } catch (err) {
    logNote('Extension pairing failed: ' + err.message, true);
    showToast('Could not generate extension pairing code', true);
  }
}

async function revokeExtensionPairing(pairingID) {
  try {
    await post('/api/extension/pair/revoke', { pairing_id: pairingID });
    showToast('Extension pairing revoked');
    logNote('Browser extension pairing revoked.');
  } catch (err) {
    logNote('Revoke extension pairing failed: ' + err.message, true);
    showToast('Could not revoke extension pairing', true);
  }
}

async function post(path, body) {
  const headers = { ...authHeaders(true), 'Content-Type': 'application/json' };
  const res = await fetch(path, {method:'POST',headers,body:JSON.stringify(body)});
  const txt = await res.text();
  if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
  refresh();
  return txt ? JSON.parse(txt) : {};
}

async function postForm(path, formData) {
  const res = await fetch(path, { method: 'POST', headers: authHeaders(true), body: formData });
  const txt = await res.text();
  if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
  refresh();
  return txt ? JSON.parse(txt) : {};
}

function normalizeVoiceCommand(text) {
  const normalized = String(text || '').toLowerCase().trim().replace(/[.!?,]/g, '');
  if (normalized.includes('start session')) return 'start_session';
  if (normalized.includes('pause capture')) return 'pause_capture';
  if (normalized.includes('capture note')) return 'capture_note';
  if (normalized.includes('freeze screen')) return 'freeze_screen';
  if (normalized.includes('submit feedback')) return 'submit_feedback';
  if (normalized.includes('discard last note')) return 'discard_last_note';
  return '';
}

async function runVoiceCommand(command, spoken) {
  switch (command) {
    case 'start_session':
      if (!currentState?.session?.id) {
        await startSession();
      } else {
        await post('/api/session/resume', {});
      }
      logNote('Voice command: start session');
      return true;
    case 'pause_capture':
      await post('/api/session/pause', {});
      logNote('Voice command: pause capture');
      return true;
    case 'capture_note':
      await submitTextNote();
      logNote('Voice command: capture note');
      return true;
    case 'freeze_screen':
      await freezeFrame();
      logNote('Voice command: freeze screen');
      return true;
    case 'submit_feedback':
      await submitSession();
      logNote('Voice command: submit feedback');
      return true;
    case 'discard_last_note':
      await discardLastNote();
      logNote('Voice command: discard last note');
      return true;
    default:
      if (spoken) {
        logNote('Voice command not recognized: "' + spoken + '"', true);
      }
      return false;
  }
}

function buildVoiceRecognition() {
  const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
  if (!SR) return null;
  const rec = new SR();
  rec.continuous = true;
  rec.interimResults = false;
  rec.lang = 'en-US';
  rec.onresult = async (evt) => {
    try {
      const idx = evt.results.length - 1;
      if (idx < 0) return;
      const transcript = String(evt.results[idx][0].transcript || '').trim();
      const command = normalizeVoiceCommand(transcript);
      await runVoiceCommand(command, transcript);
    } catch (err) {
      logNote('Voice command error: ' + err.message, true);
    }
  };
  rec.onend = () => {
    if (voiceListening) {
      try {
        rec.start();
      } catch (_) {}
    }
  };
  rec.onerror = (evt) => {
    const msg = evt && evt.error ? String(evt.error) : 'unknown';
    logNote('Voice recognition error: ' + msg, true);
  };
  return rec;
}

function startVoiceCommands() {
  if (voiceListening) {
    logNote('Voice commands already listening.');
    return;
  }
  voiceRecognition = buildVoiceRecognition();
  if (!voiceRecognition) {
    logNote('SpeechRecognition is not supported in this browser.', true);
    return;
  }
  voiceListening = true;
  try {
    voiceRecognition.start();
    logNote('Voice command listening started.');
  } catch (err) {
    voiceListening = false;
    logNote('Voice command start failed: ' + err.message, true);
  }
}

function stopVoiceCommands() {
  voiceListening = false;
  if (voiceRecognition) {
    try {
      voiceRecognition.stop();
    } catch (_) {}
  }
  logNote('Voice command listening stopped.');
}

async function reportCaptureSource(source, status, reason='') {
  try {
    await post('/api/capture/source', { source, status, reason });
  } catch (_) {}
}

async function refreshAudioDevices() {
  try {
    let devicePayload = [];
    if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
      const devices = await navigator.mediaDevices.enumerateDevices();
      devicePayload = devices
        .filter(d => d.kind === 'audioinput')
        .map(d => ({ id: d.deviceId || 'default', label: d.label || 'Audio Input' }));
    }
    const data = await post('/api/audio/devices', { devices: devicePayload });
    if (data && data.state) {
      currentState = currentState || {};
      currentState.audio = data;
    }
    syncAudioUIFromState();
    logNote('Audio devices refreshed.');
  } catch (err) {
    logNote('Audio device refresh failed: ' + err.message, true);
  }
}

async function applyAudioConfig() {
  if (audioConfigApplyTimer) {
    clearTimeout(audioConfigApplyTimer);
    audioConfigApplyTimer = 0;
  }
  audioConfigApplying = true;
  try {
    if (currentState?.config_locked) {
      throw new Error('Config is locked by policy.');
    }
    const payload = {
      mode: audioModeEl.value || 'always_on',
      input_device_id: audioInputDeviceEl.value || 'default',
      muted: !!audioMutedEl.checked,
      paused: !!audioPausedEl.checked
    };
    const data = await post('/api/audio/config', payload);
    if (data && data.state) {
      currentState = currentState || {};
      currentState.audio = data;
    }
    audioConfigDirty = false;
    setUISetting('audio_mode', payload.mode);
    syncAudioUIFromState();
    logNote('Audio configuration applied.');
  } catch (err) {
    audioConfigDirty = false;
    syncAudioUIFromState();
    logNote('Audio configuration failed: ' + err.message, true);
  } finally {
    audioConfigApplying = false;
  }
}

function scheduleAudioConfigApply() {
  audioConfigDirty = true;
  if (audioConfigApplyTimer) {
    clearTimeout(audioConfigApplyTimer);
  }
  audioConfigApplyTimer = window.setTimeout(() => {
    applyAudioConfig();
  }, 300);
}

async function validateAudioLevel() {
  return testMicrophone();
}

function setAudioLevelStateVisible(visible) {
  if (!audioLevelStateEl) return;
  audioLevelStateEl.classList.toggle('hidden', !visible);
}

async function testMicrophone(seconds = 10) {
  if (micTestRunning) {
    logNote('Mic test is already running.');
    return;
  }
  let stream = null;
  let ctx = null;
  let interval = 0;
  try {
    micTestRunning = true;
    setAudioLevelStateVisible(true);
    if (testMicBtnEl) testMicBtnEl.disabled = true;
    if (micTestMeterFillEl) micTestMeterFillEl.style.width = '0%';
    if (micTestStateEl) {
      micTestStateEl.textContent = 'Starting mic test...';
      micTestStateEl.style.color = '#4fd1c5';
    }
    const deviceId = (audioInputDeviceEl && audioInputDeviceEl.value) ? String(audioInputDeviceEl.value) : '';
    const constraints = {
      audio: deviceId ? { deviceId: { exact: deviceId } } : true,
      video: false
    };
    stream = await navigator.mediaDevices.getUserMedia(constraints);
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
        const widthPct = Math.round(normalized * 100);
        if (micTestMeterFillEl) micTestMeterFillEl.style.width = widthPct + '%';
        if (micTestStateEl) {
          micTestStateEl.textContent = 'Testing microphone... ' + leftSeconds + 's left | level=' + rms.toFixed(3);
          micTestStateEl.style.color = '#4fd1c5';
        }
        if (elapsed >= durationMs) {
          window.clearInterval(interval);
          resolve();
        }
      }, 100);
    });

    const avg = samples > 0 ? (sum / samples) : 0;
    await post('/api/audio/level', { level: peak });
    const min = Number(currentState?.audio?.state?.level_min || 0.02);
    const max = Number(currentState?.audio?.state?.level_max || 0.95);
    const valid = peak >= min && peak <= max;
    if (micTestStateEl) {
      micTestStateEl.textContent = 'Mic test complete. peak=' + peak.toFixed(3) + ' avg=' + avg.toFixed(3) + ' valid=' + valid;
      micTestStateEl.style.color = valid ? '#48bb78' : '#f6ad55';
    }
    logNote('Mic test complete. Peak level: ' + peak.toFixed(3) + ' (valid=' + valid + ').');
  } catch (err) {
    await reportCaptureSource('microphone', 'degraded', 'audio level validation failed');
    if (micTestStateEl) {
      micTestStateEl.textContent = 'Mic test failed: ' + err.message;
      micTestStateEl.style.color = '#f56565';
    }
    logNote('Audio level validation failed: ' + err.message, true);
  } finally {
    if (interval) window.clearInterval(interval);
    if (ctx) {
      try {
        await ctx.close();
      } catch (_) {}
    }
    if (stream) stream.getTracks().forEach(t => t.stop());
    micTestRunning = false;
    setAudioLevelStateVisible(false);
    if (testMicBtnEl) testMicBtnEl.disabled = false;
  }
}

async function applyTranscriptionRuntime() {
  if (sttRuntimeApplyTimer) {
    clearTimeout(sttRuntimeApplyTimer);
    sttRuntimeApplyTimer = 0;
  }
  sttRuntimeApplying = true;
  try {
    if (currentState?.config_locked) throw new Error('Config is locked by policy.');
    const mode = (sttModeEl?.value || '').trim();
    const timeoutSeconds = Number.parseInt(sttTimeoutSecondsEl?.value || '0', 10);
    const payload = {
      mode,
      base_url: (sttBaseURLEl?.value || '').trim(),
      model: currentSTTModelValue(),
      device: (sttDeviceEl?.value || '').trim(),
      compute_type: (sttComputeTypeEl?.value || '').trim(),
      language: (sttLanguageEl?.value || '').trim(),
      local_command: (sttLocalCommandEl?.value || '').trim(),
      timeout_seconds: Number.isFinite(timeoutSeconds) && timeoutSeconds > 0 ? timeoutSeconds : 0
    };
    const data = await post('/api/runtime/transcription', payload);
    if (data && data.runtime_transcription) {
      currentState = currentState || {};
      currentState.runtime_transcription = data.runtime_transcription;
      currentState.transcription_mode = data.runtime_transcription.mode || currentState.transcription_mode;
      currentState.transcription_provider = data.runtime_transcription.provider || currentState.transcription_provider;
      currentState.transcription_endpoint = data.runtime_transcription.endpoint || currentState.transcription_endpoint;
    }
    sttRuntimeDirty = false;
    syncSTTRuntimeUIFromState();
    setUISetting('stt_mode', mode);
    setUISetting('stt_base_url', payload.base_url);
    setUISetting('stt_model', payload.model);
    setUISetting('stt_device', payload.device);
    setUISetting('stt_compute_type', payload.compute_type);
    setUISetting('stt_language', payload.language);
    setUISetting('stt_local_command', payload.local_command);
    setUISetting('stt_timeout_seconds', Number.isFinite(timeoutSeconds) && timeoutSeconds > 0 ? String(timeoutSeconds) : '');
    if (sttHealthStateEl) {
      sttHealthStateEl.textContent = 'Transcription settings saved.';
      sttHealthStateEl.style.color = '#1f8f63';
    }
    logNote('Transcription runtime updated: ' + mode);
  } catch (err) {
    sttRuntimeDirty = false;
    if (sttHealthStateEl) {
      sttHealthStateEl.textContent = 'Transcription settings could not be saved: ' + err.message;
      sttHealthStateEl.style.color = '#c34f4f';
    }
    logNote('Transcription runtime update failed: ' + err.message, true);
  } finally {
    sttRuntimeApplying = false;
  }
}

function scheduleTranscriptionRuntimeApply() {
  sttRuntimeDirty = true;
  syncSTTRuntimeModeUI();
  if (sttHealthStateEl) {
    sttHealthStateEl.textContent = 'Saving transcription settings...';
    sttHealthStateEl.style.color = '#1c7c74';
  }
  if (sttRuntimeApplyTimer) {
    clearTimeout(sttRuntimeApplyTimer);
  }
  sttRuntimeApplyTimer = window.setTimeout(() => {
    applyTranscriptionRuntime();
  }, 350);
}

async function applyReviewMode() {
  try {
    const mode = (reviewModeEl?.value || '').trim();
    await post('/api/session/review-mode', { mode });
    logNote('Review mode updated: ' + (mode || 'general'));
  } catch (err) {
    logNote('Apply review mode failed: ' + err.message, true);
  }
}

async function checkTranscriptionHealth() {
  try {
    if (sttHealthStateEl) {
      sttHealthStateEl.textContent = 'Checking transcription connection...';
      sttHealthStateEl.style.color = '#1c7c74';
    }
    const res = await fetch('/api/runtime/transcription/health', { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    const status = String(data.status || 'unknown');
    const msg = String(data.message || '');
    if (sttHealthStateEl) {
      sttHealthStateEl.textContent = 'Transcription connection: ' + status + (msg ? (' - ' + msg) : '');
      sttHealthStateEl.style.color = status === 'ok' ? '#1f8f63' : '#c34f4f';
    }
    if (data.runtime_transcription) {
      currentState = currentState || {};
      currentState.runtime_transcription = data.runtime_transcription;
      syncSTTRuntimeUIFromState();
    }
  } catch (err) {
    if (sttHealthStateEl) {
      sttHealthStateEl.textContent = 'Connection check failed: ' + err.message;
      sttHealthStateEl.style.color = '#c34f4f';
    }
    logNote('Transcription health check failed: ' + err.message, true);
  }
}

async function refresh() {
  try {
    const res = await fetch('/api/state', { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    currentState = txt ? JSON.parse(txt) : {};
    stateRefreshError = '';
    syncProviderOptionsFromState();
    document.getElementById('capture').textContent = currentState.capture_state;
    document.getElementById('sessionId').textContent = currentState.session?.id || 'none';
    syncSessionDetailInputsFromState();
    syncReplayValueCaptureUI();
    notifySubmitAttemptTransitions(Array.isArray(currentState?.submit_attempts) ? currentState.submit_attempts : []);
    notifySubmitRecoveryNotices(currentState?.submit_recovery_notices);
    renderSubmitAttempts();
    renderSubmitControls();
    const locked = !!currentState.config_locked;
    configLockStatusEl.textContent = locked ? 'locked' : 'unlocked';
    configLockStatusEl.style.color = locked ? '#f6ad55' : '#48bb78';
    applyProfileBtn.disabled = locked;
    applyProfileBtn.title = locked ? 'Configuration updates are locked by policy.' : 'Save settings';
    syncAudioUIFromState();
    syncSTTRuntimeUIFromState();
    syncEnhancementUIFromState();
    renderCaptureGuideStatus();
    renderPlatformRuntimeStatus();
    renderComposerSupportStatus();
    renderExtensionPairings();
    renderSessionTransportControls();
    renderSensitiveCaptureBadges();
    capturePolicyEl.textContent = JSON.stringify({
      capture_scope: currentState.window_scoped ? 'window_only' : 'expanded',
      source_health: currentState.capture_sources || {},
      reduced_capabilities: currentState.reduced_capabilities || [],
      transcription: {
        mode: currentState.transcription_mode,
        provider: currentState.transcription_provider,
        endpoint: currentState.transcription_endpoint || 'local',
        remote_allowed: !!currentState.allow_remote_stt
      },
      submission: {
        remote_allowed: !!currentState.allow_remote_submission,
        adapters: currentState.adapters || []
      },
      retention_defaults: {
        audio: 'not retained by default',
        screenshots: '14 days',
        clips: '7 days',
        transcript_and_structured: '30 days'
      }
    }, null, 2);
    const rc = currentState.runtime_codex || {};
    const preservingRuntimeDraft = codexRuntimeDirty || codexRuntimeApplying;
    codexRuntimeStateEl.textContent = JSON.stringify(rc, null, 2);
    if (agentDefaultProviderEl && !preservingRuntimeDraft) agentDefaultProviderEl.value = rc.default_provider || agentDefaultProviderEl.value || 'codex_cli';
    if (videoModeEl) videoModeEl.value = currentState.video_mode || videoModeEl.value || 'event_triggered';
    if (!preservingRuntimeDraft) codexCliCmdEl.value = rc.cli_adapter_cmd || codexCliCmdEl.value || '';
    if (claudeCliCmdEl && !preservingRuntimeDraft) claudeCliCmdEl.value = rc.claude_cli_adapter_cmd || claudeCliCmdEl.value || '';
    if (opencodeCliCmdEl && !preservingRuntimeDraft) opencodeCliCmdEl.value = rc.opencode_cli_adapter_cmd || opencodeCliCmdEl.value || '';
  } catch (err) {
    handleStateRefreshFailure(err.message || 'State refresh failed.');
  }
  if (!preservingRuntimeDraft) workspaceDirEl.value = rc.codex_workdir || workspaceDirEl.value || '';
  codexWorkdirLabelEl.textContent = workspaceDirEl.value || '(not set)';
  if (!preservingRuntimeDraft) codexOutputDirEl.value = rc.codex_output_dir || codexOutputDirEl.value || '';
  if (!preservingRuntimeDraft) cliTimeoutSecondsEl.value = rc.cli_timeout_seconds || cliTimeoutSecondsEl.value || '600';
  if (claudeCliTimeoutSecondsEl && !preservingRuntimeDraft) claudeCliTimeoutSecondsEl.value = rc.claude_cli_timeout_seconds || claudeCliTimeoutSecondsEl.value || '600';
  if (opencodeCliTimeoutSecondsEl && !preservingRuntimeDraft) opencodeCliTimeoutSecondsEl.value = rc.opencode_cli_timeout_seconds || opencodeCliTimeoutSecondsEl.value || '600';
  if (!preservingRuntimeDraft) submitExecutionModeEl.value = rc.submit_execution_mode || submitExecutionModeEl.value || 'series';
  if (!preservingRuntimeDraft) codexSandboxEl.value = rc.codex_sandbox || '';
  if (!preservingRuntimeDraft) codexApprovalEl.value = rc.codex_approval_policy || '';
  if (!preservingRuntimeDraft && rc.codex_skip_git_repo_check !== undefined && rc.codex_skip_git_repo_check !== null && String(rc.codex_skip_git_repo_check) !== '') {
    codexSkipRepoCheckEl.checked = String(rc.codex_skip_git_repo_check) !== '0';
  }
  if (!preservingRuntimeDraft) codexProfileEl.value = rc.codex_profile || codexProfileEl.value || '';
  if (!preservingRuntimeDraft) codexModelEl.value = rc.codex_model || codexModelEl.value || '';
  if (!preservingRuntimeDraft) codexReasoningEl.value = rc.codex_reasoning_effort || codexReasoningEl.value || '';
  if (codexAPIBaseURLEl && !preservingRuntimeDraft) codexAPIBaseURLEl.value = rc.openai_base_url || codexAPIBaseURLEl.value || '';
  if (codexAPITimeoutSecondsEl && !preservingRuntimeDraft) codexAPITimeoutSecondsEl.value = rc.codex_api_timeout_seconds || codexAPITimeoutSecondsEl.value || '60';
  if (codexAPIOrgEl && !preservingRuntimeDraft) codexAPIOrgEl.value = rc.openai_org_id || codexAPIOrgEl.value || '';
  if (codexAPIProjectEl && !preservingRuntimeDraft) codexAPIProjectEl.value = rc.openai_project_id || codexAPIProjectEl.value || '';
  if (claudeAPIBaseURLEl && !preservingRuntimeDraft) claudeAPIBaseURLEl.value = rc.anthropic_base_url || claudeAPIBaseURLEl.value || '';
  if (claudeAPITimeoutSecondsEl && !preservingRuntimeDraft) claudeAPITimeoutSecondsEl.value = rc.claude_api_timeout_seconds || claudeAPITimeoutSecondsEl.value || '60';
  if (claudeAPIModelEl && !preservingRuntimeDraft) claudeAPIModelEl.value = rc.claude_api_model || claudeAPIModelEl.value || '';
  syncDeliveryPromptUIFromState(rc, preservingRuntimeDraft);
  if (claudeAPIKeyStatusEl) {
    claudeAPIKeyStatusEl.textContent = rc.anthropic_api_key_configured ? 'ANTHROPIC_API_KEY detected for claude_api.' : 'Set ANTHROPIC_API_KEY in the environment before using claude_api.';
  }
  if (!preservingRuntimeDraft) postSubmitRebuildCmdEl.value = rc.post_submit_rebuild_cmd || postSubmitRebuildCmdEl.value || '';
  if (!preservingRuntimeDraft) postSubmitVerifyCmdEl.value = rc.post_submit_verify_cmd || postSubmitVerifyCmdEl.value || '';
  if (!preservingRuntimeDraft) postSubmitTimeoutSecEl.value = rc.post_submit_timeout_seconds || postSubmitTimeoutSecEl.value || '600';
  syncCodexRuntimeModeUI();
  workspaceBrowserStateEl.textContent = JSON.stringify({
    selected_workspace: workspaceDirEl.value || '',
    picker: 'native folder dialog'
  }, null, 2);
  updateWorkspaceModalState();
  if (!workspacePrompted) {
    workspacePrompted = true;
    openWorkspaceModal();
  }
  stateEl.textContent = JSON.stringify(currentState, null, 2);
  refreshActiveSubmitLog();
}

function setSelectOptions(selectEl, values, currentValue, defaultLabel) {
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

function syncProviderOptionsFromState() {
  if (!agentDefaultProviderEl) return;
  const adapters = Array.isArray(currentState?.adapters) ? currentState.adapters : [];
  if (!adapters.length) return;
  const preferred = String(currentState?.runtime_codex?.default_provider || '');
  const selected = String(agentDefaultProviderEl.value || preferred || '');
  agentDefaultProviderEl.innerHTML = '';
  adapters.forEach(name => {
    const value = String(name || '').trim();
    if (!value) return;
    const opt = document.createElement('option');
    opt.value = value;
    opt.textContent = value;
    agentDefaultProviderEl.appendChild(opt);
  });
  if (selected) {
    agentDefaultProviderEl.value = selected;
  }
  if (!agentDefaultProviderEl.value && adapters.length) {
    agentDefaultProviderEl.value = String(adapters[0]);
  }
  renderCaptureAgentNotice();
  syncCodexRuntimeModeUI();
}

function deliveryPromptForProfile(runtimeCodex, profile) {
  const rc = runtimeCodex || {};
  switch (String(profile || '').trim()) {
    case 'draft_plan':
      return String(rc.draft_plan_prompt || '').trim() || defaultDeliveryIntentPrompt('draft_plan');
    case 'create_jira_tickets':
      return String(rc.create_jira_tickets_prompt || '').trim() || defaultDeliveryIntentPrompt('create_jira_tickets');
    default:
      return String(rc.implement_changes_prompt || '').trim() || defaultDeliveryIntentPrompt('implement_changes');
  }
}

function syncDeliveryPromptUIFromState(runtimeCodex, preservingRuntimeDraft = false) {
  if (!deliveryIntentProfileEl || !deliveryInstructionTextEl || preservingRuntimeDraft) return;
  const rc = runtimeCodex || {};
  const profile = String(rc.delivery_intent_profile || deliveryIntentProfileEl.value || 'implement_changes').trim();
  deliveryIntentProfileEl.value = profile === 'draft_plan' || profile === 'create_jira_tickets' ? profile : 'implement_changes';
  deliveryInstructionTextEl.value = deliveryPromptForProfile(rc, deliveryIntentProfileEl.value);
}

function currentDeliveryPromptPayload() {
  const selected = selectedDeliveryIntentProfile();
  const rc = currentState?.runtime_codex || {};
  const selectedText = selectedDeliveryInstructionText();
  return {
    delivery_intent_profile: selected,
    implement_changes_prompt: selected === 'implement_changes' ? selectedText : deliveryPromptForProfile(rc, 'implement_changes'),
    draft_plan_prompt: selected === 'draft_plan' ? selectedText : deliveryPromptForProfile(rc, 'draft_plan'),
    create_jira_tickets_prompt: selected === 'create_jira_tickets' ? selectedText : deliveryPromptForProfile(rc, 'create_jira_tickets'),
  };
}

function selectedProvider() {
  const provider = String(agentDefaultProviderEl?.value || currentState?.runtime_codex?.default_provider || '').trim();
  if (provider) return provider;
  const adapters = Array.isArray(currentState?.adapters) ? currentState.adapters : [];
  if (adapters.includes('codex_cli')) return 'codex_cli';
  if (adapters.includes('cli')) return 'cli';
  if (adapters.length > 0) return String(adapters[0]);
  return 'codex_cli';
}

function renderCaptureAgentNotice() {
  const noticeEl = document.getElementById('captureAgentNotice');
  if (!noticeEl) return;
  const provider = selectedProvider();
  noticeEl.innerHTML = 'Current coding agent: <strong>' + escapePreviewHTML(provider) + '</strong>. Change it in <a href="#" onclick="openAgentSettingsFromNotice(event)">Settings</a> → Agent or in <code>knit.toml</code>.';
}

function defaultDeliveryIntentPrompt(profile) {
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

function selectedDeliveryIntentProfile() {
  const profile = String(deliveryIntentProfileEl?.value || 'implement_changes').trim();
  if (profile === 'draft_plan' || profile === 'create_jira_tickets') return profile;
  return 'implement_changes';
}

function selectedDeliveryIntentLabel() {
  switch (selectedDeliveryIntentProfile()) {
    case 'draft_plan':
      return 'Draft plan';
    case 'create_jira_tickets':
      return 'Create Jira tickets';
    default:
      return 'Implement changes';
  }
}

function selectedDeliveryInstructionText() {
  const profile = selectedDeliveryIntentProfile();
  const text = String(deliveryInstructionTextEl?.value || '').trim();
  return text || defaultDeliveryIntentPrompt(profile);
}

function syncDeliveryIntentPromptText(force = false) {
  if (!deliveryInstructionTextEl) return;
  const template = defaultDeliveryIntentPrompt(selectedDeliveryIntentProfile());
  if (force || !String(deliveryInstructionTextEl.value || '').trim()) {
    deliveryInstructionTextEl.value = template;
  }
}

async function refreshCodexOptions() {
  try {
    codexOptionsAttempted = true;
    setCodexRuntimeStatus('Loading Codex model/reasoning options...');
    const res = await fetch('/api/runtime/codex/options', { headers: authHeaders(false) });
    const txt = await res.text();
    if (!res.ok) throw new Error(txt || ('HTTP ' + res.status));
    const data = txt ? JSON.parse(txt) : {};
    const models = Array.isArray(data.models) ? data.models : [];
    const modelValues = models.map(m => (m && m.model) ? String(m.model) : '').filter(Boolean);
    const reasoningValues = Array.isArray(data.reasoning_efforts) ? data.reasoning_efforts : [];
    const currentModel = (currentState?.runtime_codex?.codex_model || codexModelEl.value || data.default_model || '').trim();
    const currentReasoning = (currentState?.runtime_codex?.codex_reasoning_effort || codexReasoningEl.value || data.default_reasoning || '').trim();
    setSelectOptions(codexModelEl, modelValues, currentModel, 'Use Codex default model');
    setSelectOptions(codexReasoningEl, reasoningValues, currentReasoning, 'Use Codex default reasoning');
    setCodexRuntimeStatus('Loaded ' + modelValues.length + ' models from Codex CLI.');
    codexOptionsLoaded = true;
  } catch (err) {
    setCodexRuntimeStatus('Codex options load failed: ' + err.message, true);
  }
}

function companionSnippet() {
  return "(() => { const s = document.createElement('script'); s.src = '" + location.origin + "/companion.js?token=" + encodeURIComponent(controlToken) + "'; document.head.appendChild(s); })();";
}

function openFloatingComposerPopup() {
  const url = '/floating-composer?token=' + encodeURIComponent(controlToken);
  const width = 430;
  const height = 360;
  const left = Math.max(0, (window.screenX || 0) + ((window.outerWidth || window.innerWidth || 1024) - width - 24));
  const top = Math.max(0, (window.screenY || 0) + 80);
  const features = [
    'popup=yes',
    'resizable=yes',
    'scrollbars=yes',
    'menubar=no',
    'toolbar=no',
    'location=no',
    'status=no',
    'width=' + width,
    'height=' + height,
    'left=' + Math.round(left),
    'top=' + Math.round(top),
  ].join(',');
  const popup = window.open(url, 'knitFloatingComposer', features);
  if (!popup) {
    logNote('Popup blocked. Allow popups for 127.0.0.1 and try again.', true);
    return;
  }
  try { popup.focus(); } catch (_) {}
  window.setTimeout(() => {
    try { popup.focus(); } catch (_) {}
  }, 150);
  logNote('Composer popup opened. Browser security may prevent strict always-on-top behavior.');
}

async function copyCompanionSnippet() {
  closeCaptureSettingsModal();
  const snippet = companionSnippet();
  try {
    await navigator.clipboard.writeText(snippet);
    logNote('Companion snippet copied. Paste/run it in DevTools of the target app tab.');
    showToast('Connect browser link copied');
  } catch {
    try {
      window.prompt('Copy Browser Companion snippet:', snippet);
      logNote('Copy snippet from the prompt and run it in target-tab DevTools.');
      showToast('Connect browser link ready to copy');
    } catch {
      logNote('Clipboard copy failed. Your browser blocked fallback prompt.', true);
      showToast('Could not copy the browser link', true);
    }
  }
}

async function startSession() {
  if (hasLiveSession()) {
    logNote('A live review session is already active. Stop it before starting another one.', true);
    return;
  }
  const targetWindow = String(targetWindowEl?.value || '').trim() || DEFAULT_TARGET_WINDOW;
  const targetURL = String(targetURLEl?.value || '').trim();
  if (targetWindowEl) targetWindowEl.value = targetWindow;
  await post('/api/session/start', { target_window: targetWindow, target_url: targetURL });
  logNote('Session started. Load companion script in target browser window.');
}

async function togglePauseResume() {
  try {
    if (!currentState?.session?.id) throw new Error('Start a session first.');
    const paused = String(currentState?.capture_state || '') === 'paused';
    if (paused) {
      await post('/api/session/resume', {});
      logNote('Capture resumed.');
      return;
    }
    await post('/api/session/pause', {});
    logNote('Capture paused.');
  } catch (err) {
    logNote('Pause/Resume failed: ' + err.message, true);
  }
}

async function stopSession() {
  try {
    if (!currentState?.session?.id) throw new Error('No active session.');
    await post('/api/session/stop', {});
    logNote('Session stopped.');
  } catch (err) {
    logNote('Stop failed: ' + err.message, true);
  }
}

async function deleteSession() {
  try {
    if (!currentState?.session?.id) throw new Error('No active session to delete.');
    const ok = window.confirm('Delete the current session and all of its captured artifacts?');
    if (!ok) return;
    const res = await post('/api/purge/session', {});
    const id = String(res?.session_id || '');
    const deleted = Number(res?.artifacts_deleted || 0);
    logNote('Session deleted' + (id ? (': ' + id) : '') + '. Artifacts removed: ' + deleted + '.');
  } catch (err) {
    logNote('Delete session failed: ' + err.message, true);
  }
}

async function purgeAllData() {
  try {
    const ok = window.confirm('Purge ALL sessions and artifacts on this machine? This cannot be undone.');
    if (!ok) return;
    const res = await post('/api/purge/all', {});
    const deleted = Number(res?.artifacts_deleted || 0);
    logNote('All local session data purged. Artifacts removed: ' + deleted + '.');
  } catch (err) {
    logNote('Purge all data failed: ' + err.message, true);
  }
}

async function enableVisualCapture() {
  try {
    if (!requireCompanionFor('enable visual capture')) {
      await reportCaptureSource('screen', 'degraded', 'browser companion required before visual capture');
      return;
    }
    displayStream = await navigator.mediaDevices.getDisplayMedia({
      video: { frameRate: { ideal: 15, max: 30 } },
      audio: false
    });
    preview.srcObject = displayStream;
    await waitForPreviewReady();
    startClipRecorder();
    await reportCaptureSource('screen', 'available', 'visual capture enabled');
    syncLaserModeForVideo();
    logNote('Visual capture enabled.');
  } catch (err) {
    await reportCaptureSource('screen', 'unavailable', 'visual capture denied or unavailable');
    logNote('Visual capture denied or unavailable: ' + err.message, true);
  }
}

function disableVisualCapture() {
  if (clipRecorder && clipRecorder.state !== 'inactive') {
    clipRecorder.stop();
  }
  stopClipComposition();
  if (displayStream) {
    displayStream.getTracks().forEach(t => t.stop());
  }
  displayStream = null;
  preview.srcObject = null;
  preChunks = [];
  onDemandChunks = [];
  onDemandRecording = false;
  pendingOnDemandClip = null;
  continuousChunks = [];
  frozenFrameBlob = null;
  if (capturePerfStateEl) {
    capturePerfStateEl.textContent = 'capture profile: idle';
  }
  reportCaptureSource('screen', 'degraded', 'visual capture disabled');
  syncLaserModeForVideo();
  logNote('Visual capture disabled.');
}

async function waitForPreviewReady() {
  if (preview.videoWidth > 0 && preview.videoHeight > 0) return;
  await new Promise((resolve, reject) => {
    let done = false;
    const timeout = setTimeout(() => {
      if (done) return;
      done = true;
      cleanup();
      reject(new Error('preview stream did not become ready'));
    }, 2500);
    const onReady = () => {
      if (done) return;
      done = true;
      cleanup();
      resolve();
    };
    const cleanup = () => {
      clearTimeout(timeout);
      preview.removeEventListener('loadedmetadata', onReady);
      preview.removeEventListener('playing', onReady);
    };
    preview.addEventListener('loadedmetadata', onReady);
    preview.addEventListener('playing', onReady);
  });
}

function clipProfileForRuntime() {
  const mode = videoModeEl?.value || 'event_triggered';
  const quality = videoQualityProfileEl?.value || 'balanced';
  const cores = Number(navigator.hardwareConcurrency || 4);
  let fps = 12;
  let bitrate = 1_200_000;
  if (mode === 'continuous') {
    fps = 10;
    bitrate = 900_000;
  }
  if (cores <= 4) {
    fps = Math.min(fps, 8);
    bitrate = Math.min(bitrate, 700_000);
  } else if (cores <= 8) {
    fps = Math.min(fps, 10);
    bitrate = Math.min(bitrate, 1_000_000);
  }
  if (quality === 'smaller') {
    fps = Math.max(6, Math.min(fps, 8));
    bitrate = Math.min(bitrate, 700_000);
  } else if (quality === 'detail') {
    fps = Math.max(fps, mode === 'continuous' ? 10 : 14);
    bitrate = Math.max(bitrate, mode === 'continuous' ? 1_000_000 : 1_600_000);
  }
  return { fps, bitrate };
}

function videoScopeAndRegionForCapture() {
  const mode = screenshotMode?.value || 'full-window';
  if (mode === 'selected-region' && selectionRect && selectionRect.w > 4 && selectionRect.h > 4) {
    const sx = preview.videoWidth / Math.max(preview.clientWidth, 1);
    const sy = preview.videoHeight / Math.max(preview.clientHeight, 1);
    const region = {
      x: Math.max(0, Math.round(selectionRect.x * sx)),
      y: Math.max(0, Math.round(selectionRect.y * sy)),
      w: Math.max(1, Math.round(selectionRect.w * sx)),
      h: Math.max(1, Math.round(selectionRect.h * sy))
    };
    return { scope: 'selected-region', region };
  }
  if (mode === 'pointer-highlighted') {
    return { scope: 'pointer-highlighted', region: null };
  }
  return { scope: 'full-window', region: null };
}

function stopClipComposition() {
  if (clipRenderRAF) {
    cancelAnimationFrame(clipRenderRAF);
    clipRenderRAF = 0;
  }
  if (clipSourceStream) {
    clipSourceStream.getTracks().forEach(t => t.stop());
  }
  if (clipAudioStream) {
    clipAudioStream.getTracks().forEach(t => t.stop());
  }
  clipSourceStream = null;
  clipAudioStream = null;
  clipRenderCanvas = null;
  clipRenderCtx = null;
}

function drawPointerOverlay(ctx, canvasW, canvasH, sourceRect) {
  const ptr = currentState?.pointer_latest;
  if (!ptr) return;
  const px = Number(ptr.x || 0);
  const py = Number(ptr.y || 0);
  const vx = px * (preview.videoWidth / Math.max(preview.clientWidth, 1));
  const vy = py * (preview.videoHeight / Math.max(preview.clientHeight, 1));
  const cx = Math.round((vx - sourceRect.x) * (canvasW / Math.max(sourceRect.w, 1)));
  const cy = Math.round((vy - sourceRect.y) * (canvasH / Math.max(sourceRect.h, 1)));
  if (cx < -40 || cy < -40 || cx > canvasW + 40 || cy > canvasH + 40) return;
  ctx.save();
  ctx.strokeStyle = 'rgba(255, 90, 90, 0.92)';
  ctx.lineWidth = 5;
  ctx.beginPath();
  ctx.arc(cx, cy, 18, 0, Math.PI * 2);
  ctx.stroke();
  ctx.restore();
}

function drawLaserOverlay(ctx, canvasW, canvasH, sourceRect) {
  if (!laserModeEnabledEl?.checked || laserTrail.length < 2) return;
  const toCanvas = (pt) => {
    const vx = Number(pt.x || 0) * (preview.videoWidth / Math.max(preview.clientWidth, 1));
    const vy = Number(pt.y || 0) * (preview.videoHeight / Math.max(preview.clientHeight, 1));
    return {
      x: (vx - sourceRect.x) * (canvasW / Math.max(sourceRect.w, 1)),
      y: (vy - sourceRect.y) * (canvasH / Math.max(sourceRect.h, 1))
    };
  };
  ctx.save();
  ctx.strokeStyle = 'rgba(255, 80, 80, 0.92)';
  ctx.lineWidth = 7;
  ctx.lineCap = 'round';
  ctx.beginPath();
  laserTrail.forEach((pt, idx) => {
    const pos = toCanvas(pt);
    if (idx === 0) ctx.moveTo(pos.x, pos.y); else ctx.lineTo(pos.x, pos.y);
  });
  ctx.stroke();
  const last = toCanvas(laserTrail[laserTrail.length - 1]);
  ctx.strokeStyle = 'rgba(255, 220, 80, 0.96)';
  ctx.lineWidth = 4;
  ctx.beginPath();
  ctx.arc(last.x, last.y, 16, 0, Math.PI * 2);
  ctx.stroke();
  ctx.restore();
}

function startClipRecorder() {
  if (!displayStream) return;
  stopClipComposition();
  preChunks = [];
  clipSubscribers = [];
  onDemandChunks = [];
  continuousChunks = [];
  pendingOnDemandClip = null;
  const profile = clipProfileForRuntime();
  clipProfileState = profile;
  const scope = videoScopeAndRegionForCapture();
  const sourceRect = scope.region || { x: 0, y: 0, w: preview.videoWidth, h: preview.videoHeight };
  clipRenderCanvas = document.createElement('canvas');
  clipRenderCanvas.width = Math.max(1, sourceRect.w);
  clipRenderCanvas.height = Math.max(1, sourceRect.h);
  clipRenderCtx = clipRenderCanvas.getContext('2d', { alpha: false });
  clipPointerOverlayEnabled = true;

  const drawFrame = () => {
    if (!clipRenderCtx || !displayStream) return;
    clipRenderCtx.drawImage(
      preview,
      sourceRect.x,
      sourceRect.y,
      sourceRect.w,
      sourceRect.h,
      0,
      0,
      clipRenderCanvas.width,
      clipRenderCanvas.height
    );
    drawPointerOverlay(clipRenderCtx, clipRenderCanvas.width, clipRenderCanvas.height, sourceRect);
    drawLaserOverlay(clipRenderCtx, clipRenderCanvas.width, clipRenderCanvas.height, sourceRect);
    clipRenderRAF = requestAnimationFrame(drawFrame);
  };
  clipRenderRAF = requestAnimationFrame(drawFrame);

  clipSourceStream = clipRenderCanvas.captureStream(profile.fps);
  let clipHasAudio = false;
  if (clipIncludeAudioEl?.checked) {
    const audioMode = audioModeEl?.value || 'always_on';
    const audioMuted = !!audioMutedEl?.checked;
    const audioPaused = !!audioPausedEl?.checked;
    if (!audioMuted && !audioPaused && (audioMode !== 'push_to_talk' || pttHeld)) {
      const deviceId = (audioInputDeviceEl && audioInputDeviceEl.value) ? String(audioInputDeviceEl.value) : '';
      navigator.mediaDevices.getUserMedia({
        audio: deviceId ? { deviceId: { exact: deviceId } } : true,
        video: false
      }).then(stream => {
        clipAudioStream = stream;
        const audioTrack = stream.getAudioTracks()[0];
        if (audioTrack && clipSourceStream) {
          clipSourceStream.addTrack(audioTrack);
          clipHasAudio = true;
        }
      }).catch((err) => {
        logNote('Clip audio unavailable; proceeding without clip audio: ' + err.message, true);
      });
    } else {
      logNote('Clip audio requested but microphone is paused/muted or push-to-talk is not held.', true);
    }
  }

  try {
    clipRecorder = new MediaRecorder(clipSourceStream, {
      mimeType: 'video/webm;codecs=vp9',
      videoBitsPerSecond: profile.bitrate
    });
  } catch {
    try {
      clipRecorder = new MediaRecorder(clipSourceStream, {
        mimeType: 'video/webm;codecs=vp8',
        videoBitsPerSecond: Math.round(profile.bitrate * 0.85)
      });
    } catch {
      clipRecorder = new MediaRecorder(clipSourceStream);
    }
  }
  clipMimeType = clipRecorder.mimeType || 'video/webm';
  if (capturePerfStateEl) {
    capturePerfStateEl.textContent = 'capture profile: ' + profile.fps + 'fps @ ~' + Math.round(profile.bitrate / 1000) + 'kbps' +
      ' | codec=' + clipMimeType + ' | clip_audio=' + (clipHasAudio ? 'on' : 'off') +
      ' | scope=' + scope.scope;
  }
  clipRecorder.ondataavailable = (evt) => {
    if (!evt.data || evt.data.size === 0) return;
    const item = { ts: Date.now(), blob: evt.data };
    preChunks.push(item);
    const cutoff = Date.now() - 5000;
    preChunks = preChunks.filter(x => x.ts >= cutoff);
    if (onDemandRecording) {
      onDemandChunks.push(item);
    }
    if ((videoModeEl?.value || 'event_triggered') === 'continuous') {
      continuousChunks.push(item);
      const continuousCutoff = Date.now() - 120000;
      continuousChunks = continuousChunks.filter(x => x.ts >= continuousCutoff);
    }
    clipSubscribers.forEach(fn => fn(item));
  };
  clipRecorder.start(1000);
}

function startOnDemandClip() {
  if (!requireCompanionFor('record video clips')) return;
  if (!clipRecorder || clipRecorder.state !== 'recording') {
    logNote('Enable visual capture first for on-demand clips.', true);
    return;
  }
  onDemandRecording = true;
  onDemandChunks = [];
  pendingOnDemandClip = null;
  logNote('On-demand clip recording started.');
}

function stopOnDemandClip() {
  if (!onDemandRecording) {
    logNote('On-demand clip is not active.', true);
    return;
  }
  onDemandRecording = false;
  if (!onDemandChunks.length) {
    logNote('On-demand clip stopped with no data.', true);
    return;
  }
  pendingOnDemandClip = { items: onDemandChunks.slice() };
  logNote('On-demand clip ready for next note submission.');
}

async function captureScreenshotBlob() {
  if (frozenFrameBlob) {
    return frozenFrameBlob;
  }
  if (!isCompanionAttached()) {
    return null;
  }
  if (!displayStream || !preview.videoWidth || !preview.videoHeight) return null;
  const mode = screenshotMode.value || 'full-window';
  const baseCanvas = document.createElement('canvas');
  baseCanvas.width = preview.videoWidth;
  baseCanvas.height = preview.videoHeight;
  const ctx = baseCanvas.getContext('2d');
  ctx.drawImage(preview, 0, 0, baseCanvas.width, baseCanvas.height);

  if (mode === 'pointer-highlighted' && currentState?.pointer_latest) {
    const px = Number(currentState.pointer_latest.x || 0);
    const py = Number(currentState.pointer_latest.y || 0);
    const sx = baseCanvas.width / Math.max(preview.clientWidth, 1);
    const sy = baseCanvas.height / Math.max(preview.clientHeight, 1);
    const x = Math.round(px * sx);
    const y = Math.round(py * sy);
    ctx.beginPath();
    ctx.strokeStyle = '#ff4d4d';
    ctx.lineWidth = 6;
    ctx.arc(x, y, 24, 0, Math.PI * 2);
    ctx.stroke();
  }

  if (laserModeEnabledEl?.checked && laserTrail.length > 1) {
    const sx = baseCanvas.width / Math.max(preview.clientWidth, 1);
    const sy = baseCanvas.height / Math.max(preview.clientHeight, 1);
    ctx.save();
    ctx.strokeStyle = 'rgba(255, 80, 80, 0.92)';
    ctx.lineWidth = 7;
    ctx.lineCap = 'round';
    ctx.beginPath();
    laserTrail.forEach((pt, idx) => {
      const x = Math.round(Number(pt.x || 0) * sx);
      const y = Math.round(Number(pt.y || 0) * sy);
      if (idx === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
    });
    ctx.stroke();
    const last = laserTrail[laserTrail.length - 1];
    const lx = Math.round(Number(last.x || 0) * sx);
    const ly = Math.round(Number(last.y || 0) * sy);
    ctx.strokeStyle = 'rgba(255, 220, 80, 0.96)';
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.arc(lx, ly, 16, 0, Math.PI * 2);
    ctx.stroke();
    ctx.restore();
  }

  if (mode === 'selected-region' && selectionRect) {
    const sx = baseCanvas.width / Math.max(preview.clientWidth, 1);
    const sy = baseCanvas.height / Math.max(preview.clientHeight, 1);
    const x = Math.max(0, Math.round(selectionRect.x * sx));
    const y = Math.max(0, Math.round(selectionRect.y * sy));
    const w = Math.max(1, Math.round(selectionRect.w * sx));
    const h = Math.max(1, Math.round(selectionRect.h * sy));
    const out = document.createElement('canvas');
    out.width = w;
    out.height = h;
    out.getContext('2d').drawImage(baseCanvas, x, y, w, h, 0, 0, w, h);
    return await new Promise(resolve => out.toBlob(resolve, 'image/png', 0.92));
  }

  return await new Promise(resolve => baseCanvas.toBlob(resolve, 'image/png', 0.92));
}

function clearSelection() {
  selectionRect = null;
  selectionStart = null;
  selecting = false;
  logNote('Selection cleared.');
}

async function captureManualScreenshot() {
  try {
    if (!requireCompanionFor('capture snapshots')) return;
    const blob = await captureScreenshotBlob();
    if (!blob) throw new Error('Visual stream not active.');
    manualScreenshotBlob = blob;
    logNote('Manual screenshot captured and queued for next note.');
  } catch (err) {
    logNote('Manual screenshot failed: ' + err.message, true);
  }
}

async function freezeFrame() {
  try {
    if (!requireCompanionFor('capture snapshots')) return;
    const blob = await captureScreenshotBlob();
    if (!blob) throw new Error('Visual stream not active.');
    frozenFrameBlob = blob;
    logNote('Frame frozen for annotation and note capture.');
  } catch (err) {
    logNote('Freeze frame failed: ' + err.message, true);
  }
}

function unfreezeFrame() {
  frozenFrameBlob = null;
  logNote('Frame unfrozen.');
}

async function discardLastNote() {
  try {
    const data = await post('/api/session/feedback/discard-last', {});
    const id = String(data.discarded_event_id || '');
    if (id) {
      logNote('Discarded last note: ' + id);
    } else {
      logNote('No note to discard.');
    }
  } catch (err) {
    logNote('Discard last note failed: ' + err.message, true);
  }
}

async function captureEventClipBlob() {
  if (!requireCompanionFor('record video clips')) return null;
  if (!enableClips.checked || !clipRecorder || clipRecorder.state !== 'recording') return null;
  const scopeInfo = videoScopeAndRegionForCapture();
  const selectedWindow = String(targetWindowEl?.value || '').trim() || (currentState?.session?.target_window || DEFAULT_TARGET_WINDOW);
  const hasAudio = !!(clipAudioStream && clipAudioStream.getAudioTracks().length > 0);
  const baseMeta = {
    codec: clipMimeType || 'video/webm',
    hasAudio,
    pointerOverlay: clipPointerOverlayEnabled,
    scope: scopeInfo.scope,
    region: scopeInfo.region || null,
    window: selectedWindow
  };
  const finalizeClip = (items) => {
    if (!Array.isArray(items) || !items.length) return null;
    const sorted = items.slice().sort((a, b) => Number(a.ts || 0) - Number(b.ts || 0));
    const start = Number(sorted[0].ts || Date.now());
    const end = Number(sorted[sorted.length - 1].ts || start);
    return {
      blob: new Blob(sorted.map(x => x.blob), { type: baseMeta.codec || 'video/webm' }),
      codec: baseMeta.codec,
      hasAudio: baseMeta.hasAudio,
      pointerOverlay: baseMeta.pointerOverlay,
      scope: baseMeta.scope,
      region: baseMeta.region,
      window: baseMeta.window,
      startedAt: new Date(start).toISOString(),
      endedAt: new Date(end).toISOString(),
      durationMS: Math.max(0, end - start)
    };
  };
  const mode = videoModeEl?.value || 'event_triggered';
  if (mode === 'on_demand') {
    if (!pendingOnDemandClip || !pendingOnDemandClip.items?.length) return null;
    const clip = finalizeClip(pendingOnDemandClip.items);
    pendingOnDemandClip = null;
    return clip;
  }
  if (mode === 'continuous') {
    return finalizeClip(continuousChunks);
  }
  const pre = preChunks.slice();
  const post = [];
  const sub = (item) => post.push(item);
  clipSubscribers.push(sub);
  await new Promise(r => setTimeout(r, 5000));
  clipSubscribers = clipSubscribers.filter(fn => fn !== sub);
  return finalizeClip(pre.concat(post));
}

function updateAudioNoteButton(active = false) {
  if (!audioNoteBtnEl) return;
  if (active) {
    audioNoteBtnEl.textContent = '■';
    audioNoteBtnEl.title = 'Stop recording audio note';
    audioNoteBtnEl.setAttribute('aria-label', 'Stop recording audio note');
    renderSubmitControls();
    return;
  }
  audioNoteBtnEl.textContent = '🎙️';
  audioNoteBtnEl.title = 'Record audio note';
  audioNoteBtnEl.setAttribute('aria-label', 'Record audio note');
  renderSubmitControls();
}

function updateVideoNoteButton(active = false) {
  if (!videoNoteBtnEl) return;
  if (active) {
    videoNoteBtnEl.textContent = '■';
    videoNoteBtnEl.title = 'Stop recording video note';
    videoNoteBtnEl.setAttribute('aria-label', 'Stop recording video note');
    renderSubmitControls();
    return;
  }
  videoNoteBtnEl.textContent = '🎥';
  videoNoteBtnEl.title = 'Record video note';
  videoNoteBtnEl.setAttribute('aria-label', 'Record video note');
  renderSubmitControls();
}

function clearAudioNoteCapture() {
  if (audioNoteStream) {
    try {
      audioNoteStream.getTracks().forEach(t => t.stop());
    } catch (_) {}
  }
  audioNoteRecorder = null;
  audioNoteStream = null;
  audioNoteChunks = [];
  audioNoteStopPromise = null;
  updateAudioNoteButton(false);
}

function clearVideoNoteCapture() {
  if (videoNoteRenderRAF) {
    cancelAnimationFrame(videoNoteRenderRAF);
    videoNoteRenderRAF = 0;
  }
  if (videoNoteMicStream) {
    try {
      videoNoteMicStream.getTracks().forEach(t => t.stop());
    } catch (_) {}
  }
  if (videoNoteCanvasStream) {
    try {
      videoNoteCanvasStream.getTracks().forEach(t => t.stop());
    } catch (_) {}
  }
  if (videoNoteCombinedStream) {
    try {
      videoNoteCombinedStream.getTracks().forEach(t => t.stop());
    } catch (_) {}
  }
  videoNoteAudioRecorder = null;
  videoNoteClipRecorder = null;
  videoNoteMicStream = null;
  videoNoteRenderCanvas = null;
  videoNoteRenderCtx = null;
  videoNoteCanvasStream = null;
  videoNoteCombinedStream = null;
  videoNoteAudioChunks = [];
  videoNoteClipChunks = [];
  videoNoteAudioStopPromise = null;
  videoNoteClipStopPromise = null;
  activeVideoNote = null;
  updateVideoNoteButton(false);
}

function createAudioRecorderForStream(stream) {
  try {
    return new MediaRecorder(stream, { mimeType: 'audio/webm' });
  } catch {
    return new MediaRecorder(stream);
  }
}

function createVideoRecorderForStream(stream) {
  try {
    return new MediaRecorder(stream, { mimeType: 'video/webm;codecs=vp9' });
  } catch {
    try {
      return new MediaRecorder(stream, { mimeType: 'video/webm;codecs=vp8' });
    } catch {
      return new MediaRecorder(stream);
    }
  }
}

function getTargetWindowLabel() {
  return String(targetWindowEl?.value || currentState?.session?.target_window || DEFAULT_TARGET_WINDOW).trim() || DEFAULT_TARGET_WINDOW;
}

function isNoteRecordingActive() {
  return !!(audioNoteRecorder || videoNoteClipRecorder || videoNoteFinalizing);
}

async function getAudioCaptureStream() {
  const mode = audioModeEl?.value || 'always_on';
  if (mode === 'push_to_talk' && !pttHeld) {
    throw new Error('Push-to-talk only captures while this Knit tab is focused and Space is held. For cross-tab feedback, switch Audio mode to always_on.');
  }
  if (audioMutedEl?.checked) {
    throw new Error('Audio is muted. Unmute before recording.');
  }
  if (audioPausedEl?.checked) {
    throw new Error('Audio is paused. Resume before recording.');
  }
  const deviceId = (audioInputDeviceEl && audioInputDeviceEl.value) ? String(audioInputDeviceEl.value) : '';
  return navigator.mediaDevices.getUserMedia({
    audio: deviceId ? { deviceId: { exact: deviceId } } : true,
    video: false
  });
}

async function startAudioNoteCapture() {
  try {
    const stream = await getAudioCaptureStream();
    audioNoteStream = stream;
    await reportCaptureSource('microphone', 'available', 'audio capture started');
    let rec;
    try {
      rec = new MediaRecorder(stream, { mimeType: 'audio/webm' });
    } catch {
      rec = new MediaRecorder(stream);
    }
    audioNoteChunks = [];
    rec.ondataavailable = evt => { if (evt.data && evt.data.size > 0) audioNoteChunks.push(evt.data); };
    audioNoteStopPromise = new Promise(resolve => {
      rec.addEventListener('stop', resolve, { once: true });
    });
    audioNoteRecorder = rec;
    rec.start();
    updateAudioNoteButton(true);
    logNote('Recording audio note. Click Stop recording when you are done.');
  } catch (err) {
    clearAudioNoteCapture();
    throw err;
  }
}

async function finishAudioNoteCapture() {
  if (!audioNoteRecorder) {
    throw new Error('Audio note recording is not active.');
  }
  const rec = audioNoteRecorder;
  const stopPromise = audioNoteStopPromise;
  if (rec.state !== 'inactive') {
    rec.stop();
  }
  if (stopPromise) {
    await stopPromise;
  }
  const blob = new Blob(audioNoteChunks, { type: rec.mimeType || 'audio/webm' });
  clearAudioNoteCapture();
  await reportCaptureSource('microphone', 'available', 'audio capture completed');
  return blob;
}

async function startVideoNoteCapture() {
  if (!requireCompanionFor('record video clips')) {
    return null;
  }
  if (!displayStream || !preview.videoWidth || !preview.videoHeight) {
    throw new Error('Enable visual capture first from Settings → Video.');
  }
  try {
    videoNoteMicStream = await getAudioCaptureStream();
    videoNoteRenderCanvas = document.createElement('canvas');
    videoNoteRenderCanvas.width = Math.max(1, preview.videoWidth);
    videoNoteRenderCanvas.height = Math.max(1, preview.videoHeight);
    videoNoteRenderCtx = videoNoteRenderCanvas.getContext('2d', { alpha: false });
    const sourceRect = { x: 0, y: 0, w: preview.videoWidth, h: preview.videoHeight };
    const drawFrame = () => {
      if (!videoNoteRenderCtx || !displayStream) return;
      videoNoteRenderCtx.drawImage(preview, 0, 0, preview.videoWidth, preview.videoHeight);
      drawPointerOverlay(videoNoteRenderCtx, preview.videoWidth, preview.videoHeight, sourceRect);
      drawLaserOverlay(videoNoteRenderCtx, preview.videoWidth, preview.videoHeight, sourceRect);
      videoNoteRenderRAF = requestAnimationFrame(drawFrame);
    };
    videoNoteRenderRAF = requestAnimationFrame(drawFrame);
    videoNoteCanvasStream = videoNoteRenderCanvas.captureStream(12);
    videoNoteCombinedStream = new MediaStream();
    videoNoteCanvasStream.getVideoTracks().forEach(track => videoNoteCombinedStream.addTrack(track));
    videoNoteMicStream.getAudioTracks().forEach(track => videoNoteCombinedStream.addTrack(track));

    const videoTracks = displayStream.getVideoTracks();
    videoTracks.forEach((track) => {
      track.addEventListener('ended', () => {
        if (videoNoteFinalizing || !videoNoteClipRecorder) {
          return;
        }
        logNote('Visual capture ended. Finalizing your video note...');
        window.setTimeout(() => {
          finalizeVideoNoteCapture('visual capture ended').catch(() => {});
        }, 0);
      }, { once: true });
    });

    videoNoteAudioRecorder = createAudioRecorderForStream(videoNoteMicStream);
    videoNoteClipRecorder = createVideoRecorderForStream(videoNoteCombinedStream);
    videoNoteAudioChunks = [];
    videoNoteClipChunks = [];
    videoNoteAudioStopPromise = new Promise(resolve => {
      videoNoteAudioRecorder.addEventListener('stop', resolve, { once: true });
    });
    videoNoteClipStopPromise = new Promise(resolve => {
      videoNoteClipRecorder.addEventListener('stop', resolve, { once: true });
    });
    videoNoteAudioRecorder.ondataavailable = (evt) => {
      if (evt.data && evt.data.size > 0) videoNoteAudioChunks.push(evt.data);
    };
    videoNoteClipRecorder.ondataavailable = (evt) => {
      if (evt.data && evt.data.size > 0) videoNoteClipChunks.push(evt.data);
    };
    activeVideoNote = {
      startedAt: Date.now(),
      window: getTargetWindowLabel(),
      hasAudio: videoNoteCombinedStream.getAudioTracks().length > 0,
    };
    videoNoteAudioRecorder.start();
    videoNoteClipRecorder.start();
    updateVideoNoteButton(true);
    logNote('Recording video note. Click Stop video note when you are done.');
    return true;
  } catch (err) {
    clearVideoNoteCapture();
    throw err;
  }
}

async function finishVideoNoteCapture() {
  if (!videoNoteAudioRecorder || !videoNoteClipRecorder) {
    throw new Error('Video note recording is not active.');
  }
  const audioRecorder = videoNoteAudioRecorder;
  const clipRecorder = videoNoteClipRecorder;
  const audioStopPromise = videoNoteAudioStopPromise;
  const clipStopPromise = videoNoteClipStopPromise;
  const meta = activeVideoNote || { startedAt: Date.now(), window: getTargetWindowLabel(), hasAudio: false };
  if (audioRecorder.state !== 'inactive') {
    audioRecorder.stop();
  }
  if (clipRecorder.state !== 'inactive') {
    clipRecorder.stop();
  }
  await Promise.all([audioStopPromise, clipStopPromise].filter(Boolean));
  const endedAt = Date.now();
  const audioBlob = new Blob(videoNoteAudioChunks, { type: audioRecorder.mimeType || 'audio/webm' });
  const clipBlob = new Blob(videoNoteClipChunks, { type: clipRecorder.mimeType || 'video/webm' });
  clearVideoNoteCapture();
  await reportCaptureSource('microphone', 'available', 'video note capture completed');
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
      durationMS: Math.max(0, endedAt - meta.startedAt),
    },
  };
}

async function submitTextNote() {
  try {
    if (isNoteRecordingActive()) throw new Error('Stop the current recording before adding a written note.');
    if (!currentState?.session?.id) throw new Error('Start a session first.');
    const form = new FormData();
    const transcript = val('transcript').trim();
    form.append('raw_transcript', transcript);
    form.append('normalized', transcript);
    appendEnhancementFields(form);
    const screenshot = manualScreenshotBlob || await captureScreenshotBlob();
    if (screenshot) form.append('screenshot', screenshot, 'frame.png');
    manualScreenshotBlob = null;
    const result = await postForm('/api/session/feedback/note', form);
    if (result && result.command_handled) {
      const cmd = String(result.command_result?.voice_command || 'voice_command');
      if (result.command_result?.freeze_screen) {
        await freezeFrame();
      }
      logNote('Voice command handled: ' + cmd);
      return;
    }
    document.getElementById('transcript').value = '';
    logNote('Text note captured: ' + result.event_id + '. Preview/Submit will auto-prepare the latest snapshot.');
    laserTrail = [];

    if (enableClips.checked) {
      const clip = await captureEventClipBlob();
      if (clip?.blob) {
        const clipForm = new FormData();
        clipForm.append('event_id', result.event_id);
        clipForm.append('clip', clip.blob, 'event.webm');
        appendClipMetadata(clipForm, clip);
        await postForm('/api/session/feedback/clip', clipForm);
        cacheClipForEvent(result.event_id, clip);
        logNote('Text note captured and clip attached: ' + result.event_id);
      }
    }
  } catch (err) {
    logNote('Text note failed: ' + err.message, true);
  }
}

function hasTypedNoteDraft() {
  return String(document.getElementById('transcript')?.value || '').trim().length > 0;
}

async function flushTypedNoteDraft(reason) {
  if (!hasTypedNoteDraft()) {
    return false;
  }
  await submitTextNote();
  if (hasTypedNoteDraft()) {
    throw new Error('The written note could not be added before ' + reason + '.');
  }
  return true;
}

async function submitAudioNote() {
  try {
    if (videoNoteClipRecorder) {
      throw new Error('Stop the current video note before recording audio.');
    }
    if (!audioNoteRecorder) {
      if (!currentState?.session?.id) throw new Error('Start a session first.');
      if (currentState?.transcription_mode === 'remote' && !currentState?.allow_remote_stt) {
        throw new Error('Remote transcription is disabled by policy.');
      }
      await startAudioNoteCapture();
      return;
    }
    const audio = await finishAudioNoteCapture();
    const form = new FormData();
    form.append('audio', audio, 'note.webm');
    appendEnhancementFields(form);
    const screenshot = manualScreenshotBlob || await captureScreenshotBlob();
    if (screenshot) form.append('screenshot', screenshot, 'frame.png');
    manualScreenshotBlob = null;

    const result = await postForm('/api/session/feedback/note', form);
    if (result && result.command_handled) {
      const cmd = String(result.command_result?.voice_command || 'voice_command');
      if (result.command_result?.freeze_screen) {
        await freezeFrame();
      }
      logNote('Voice command handled: ' + cmd);
      return;
    }
    logNote('Audio note captured/transcribed: ' + result.event_id + '. Preview/Submit will auto-prepare the latest snapshot.');
    laserTrail = [];

    if (enableClips.checked) {
      const clip = await captureEventClipBlob();
      if (clip?.blob) {
        const clipForm = new FormData();
        clipForm.append('event_id', result.event_id);
        clipForm.append('clip', clip.blob, 'event.webm');
        appendClipMetadata(clipForm, clip);
        await postForm('/api/session/feedback/clip', clipForm);
        cacheClipForEvent(result.event_id, clip);
        logNote('Audio note captured and clip attached: ' + result.event_id);
      }
    }
  } catch (err) {
    logNote('Audio note failed: ' + err.message, true);
  }
}

async function submitVideoNote() {
  try {
    if (videoNoteFinalizing) {
      throw new Error('Video note is still finalizing.');
    }
    if (audioNoteRecorder) {
      throw new Error('Stop the current audio note before recording video.');
    }
    if (!videoNoteClipRecorder) {
      if (!currentState?.session?.id) throw new Error('Start a session first.');
      if (currentState?.transcription_mode === 'remote' && !currentState?.allow_remote_stt) {
        throw new Error('Remote transcription is disabled by policy.');
      }
      await startVideoNoteCapture();
      return;
    }
    await finalizeVideoNoteCapture('manual stop');
  } catch (err) {
    logNote('Video note failed: ' + err.message, true);
  }
}

async function finalizeVideoNoteCapture(trigger = 'manual stop') {
  if (videoNoteFinalizing) {
    return;
  }
  videoNoteFinalizing = true;
  try {
    const bundle = await finishVideoNoteCapture();
    if (!bundle?.audioBlob || !bundle?.clip?.blob) {
      throw new Error('video note could not be recorded');
    }
    const form = new FormData();
    form.append('audio', bundle.audioBlob, 'video-note.webm');
    appendEnhancementFields(form);
    const screenshot = manualScreenshotBlob || await captureScreenshotBlob();
    if (screenshot) form.append('screenshot', screenshot, 'frame.png');
    manualScreenshotBlob = null;
    const result = await postForm('/api/session/feedback/note', form);
    if (result && result.command_handled) {
      const cmd = String(result.command_result?.voice_command || 'voice_command');
      if (result.command_result?.freeze_screen) {
        await freezeFrame();
      }
      logNote('Voice command handled: ' + cmd);
      return;
    }
    const eventID = String(result?.event_id || '');
    if (!eventID) {
      throw new Error('video note was captured but the event could not be created');
    }
    const clipForm = new FormData();
    clipForm.append('event_id', eventID);
    clipForm.append('clip', bundle.clip.blob, 'video-note.webm');
    appendClipMetadata(clipForm, bundle.clip);
    await postForm('/api/session/feedback/clip', clipForm);
    cacheClipForEvent(eventID, bundle.clip);
    logNote('Video note captured and clip attached: ' + eventID + '.');
  } catch (err) {
    throw new Error(err.message + (trigger ? (' (' + trigger + ')') : ''));
  } finally {
    videoNoteFinalizing = false;
  }
}

function appendEnhancementFields(form) {
  const reviewMode = (reviewModeEl?.value || '').trim();
  const laserEnabled = isVideoCaptureActive() || !!laserModeEnabledEl?.checked;
  if (reviewMode) form.append('review_mode', reviewMode);
  if (laserEnabled) {
    form.append('laser_mode', '1');
    if (laserTrail.length > 0) {
      form.append('laser_path_json', JSON.stringify(laserTrail));
    }
  }
}

function appendClipMetadata(form, clip) {
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
  if (clip.region) {
    if (Number.isFinite(Number(clip.region.x))) form.append('video_region_x', String(Math.round(Number(clip.region.x))));
    if (Number.isFinite(Number(clip.region.y))) form.append('video_region_y', String(Math.round(Number(clip.region.y))));
    if (Number.isFinite(Number(clip.region.w))) form.append('video_region_w', String(Math.round(Number(clip.region.w))));
    if (Number.isFinite(Number(clip.region.h))) form.append('video_region_h', String(Math.round(Number(clip.region.h))));
  }
}

function currentSessionID() {
  return String(currentState?.session?.id || '').trim();
}

function formatMediaSize(bytes) {
  const size = Number(bytes || 0);
  if (!Number.isFinite(size) || size <= 0) return '0 bytes';
  if (size >= (1 << 20)) return (size / (1 << 20)).toFixed(1) + ' MB';
  if (size >= (1 << 10)) return (size / (1 << 10)).toFixed(1) + ' KB';
  return Math.round(size) + ' bytes';
}

function clipCacheMetaFromClip(clip) {
  if (!clip) return {};
  return {
    codec: String(clip.codec || clip.blob?.type || 'video/webm'),
    hasAudio: !!clip.hasAudio,
    pointerOverlay: !!clip.pointerOverlay,
    scope: String(clip.scope || 'window'),
    region: clip.region || null,
    window: String(clip.window || ''),
    startedAt: clip.startedAt || '',
    endedAt: clip.endedAt || '',
    durationMS: Number(clip.durationMS || 0),
  };
}

function cacheClipForEvent(eventID, clip) {
  const id = String(eventID || '').trim();
  if (!id || !clip?.blob) return;
  clipBlobCacheByEventID.set(id, {
    sessionID: currentSessionID(),
    blob: clip.blob,
    meta: clipCacheMetaFromClip(clip),
  });
}

function cachedClipEntryForEvent(eventID) {
  const id = String(eventID || '').trim();
  if (!id) return null;
  const entry = clipBlobCacheByEventID.get(id);
  if (!entry) return null;
  if (entry.sessionID && entry.sessionID !== currentSessionID()) {
    clipBlobCacheByEventID.delete(id);
    return null;
  }
  return entry;
}

async function fetchStoredClipForEvent(eventID) {
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

async function clipEntryForPreviewNote(note) {
  const eventID = String(note?.event_id || '').trim();
  if (!eventID) {
    throw new Error('Preview note is missing its event id.');
  }
  const cached = cachedClipEntryForEvent(eventID);
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
  const stored = await fetchStoredClipForEvent(eventID);
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

async function loadVideoElementForBlob(blob) {
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

function resizedVideoProfiles(width, height) {
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

function createConstrainedVideoRecorder(stream, bitrate) {
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

async function recordReducedVideoProfile(video, profile) {
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(2, profile.width);
  canvas.height = Math.max(2, profile.height);
  const ctx = canvas.getContext('2d', { alpha: false });
  if (!ctx) throw new Error('video resize canvas is unavailable');
  const stream = canvas.captureStream(Math.max(4, Number(profile.fps || 6)));
  const recorder = createConstrainedVideoRecorder(stream, Math.max(120000, Number(profile.bitrate || 300000)));
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

async function shrinkClipBlobToLimit(blob, maxBytes, statusLabel) {
  const { video, url } = await loadVideoElementForBlob(blob);
  try {
    const width = Math.max(2, Number(video.videoWidth || 640));
    const height = Math.max(2, Number(video.videoHeight || 360));
    const profiles = resizedVideoProfiles(width, height);
    let best = null;
    for (const profile of profiles) {
      if (statusLabel) {
        statusLabel('Trying ' + profile.width + '×' + profile.height + ' at ' + profile.fps + 'fps...');
      }
      const candidate = await recordReducedVideoProfile(video, profile);
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

function previewNoteNeedsClipResize(note) {
  const status = String(note?.video_transmission_status || '').trim();
  const size = Number(note?.video_size_bytes || 0);
  const limit = Number(note?.video_send_limit_bytes || 0);
  return !!note?.has_video && ((status === 'omitted_due_to_limit') || (limit > 0 && size > limit));
}

function renderPreviewVideoDecisionActions(note) {
  if (!previewNoteNeedsClipResize(note)) return '';
  const eventID = String(note?.event_id || '').trim();
  if (!eventID) return '';
  const busy = clipResizeInFlight.has(eventID);
  const buttonLabel = busy ? 'Making clip smaller…' : 'Make clip smaller to send';
  const snapshotLabel = previewVideoEventOmitted(eventID)
    ? 'Send clip again'
    : (note?.has_screenshot ? 'Use snapshot instead' : 'Omit clip for this request');
  const helper = '<div class="small" style="margin-top:.35rem;color:#6a7383;">Knit will lower fps, bitrate, and size until the clip fits the inline send limit. The smaller clip may drop clip audio.</div>';
  return '<div class="sub-actions" style="margin-top:.55rem;">' +
    '<button type="button" class="secondary" ' + (busy ? 'disabled ' : '') + 'onclick="fitPreviewClipToSendLimit(\'' + escapePreviewHTML(eventID) + '\')" title="' + escapePreviewHTML(buttonLabel) + '">' + escapePreviewHTML(buttonLabel) + '</button>' +
    '<button type="button" class="secondary" onclick="togglePreviewVideoEventOmission(\'' + escapePreviewHTML(eventID) + '\')" title="' + escapePreviewHTML(snapshotLabel) + '">' + escapePreviewHTML(snapshotLabel) + '</button>' +
    '</div>' + helper;
}

async function fitPreviewClipToSendLimit(eventID) {
  const id = String(eventID || '').trim();
  if (!id) return;
  if (clipResizeInFlight.has(id)) return;
  const note = previewNoteByID(id);
  if (!note) {
    logNote('Clip resize failed: change request not found in the current preview.', true);
    return;
  }
  const limit = Number(note.video_send_limit_bytes || 0);
  if (!Number.isFinite(limit) || limit <= 0) {
    logNote('Clip resize failed: send limit is unavailable for this clip.', true);
    return;
  }
  clipResizeInFlight.add(id);
  if (latestPayloadPreviewData) {
    renderPayloadPreview(latestPayloadPreviewData);
  }
  try {
    showToast('Reducing clip for ' + id + ' to fit the default send limit…');
    logNote('Making the clip for ' + id + ' smaller so it fits the default send limit of ' + formatMediaSize(limit) + '.');
    const clipEntry = await clipEntryForPreviewNote(note);
    const reduced = await shrinkClipBlobToLimit(clipEntry.blob, limit, (message) => {
      setSubmitProgress(true, 'Reducing clip for ' + id + ': ' + message);
    });
    if (!reduced || !reduced.size) {
      throw new Error('Knit could not create a smaller clip in this browser.');
    }
    if (reduced.size > limit) {
      throw new Error('The reduced clip is still ' + formatMediaSize(reduced.size) + '. Use a screenshot instead or allow large inline media.');
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
    form.append('clip', reduced, 'event-fit.webm');
    appendClipMetadata(form, clipMeta);
    await postForm('/api/session/feedback/clip', form);
    cacheClipForEvent(id, { blob: reduced, ...clipMeta });
    setSubmitProgress(false, '');
    showToast('Clip resized for ' + id + ' (' + formatMediaSize(reduced.size) + ').');
    logNote('Clip resized for ' + id + ' (' + formatMediaSize(reduced.size) + '). Preview refreshed with the smaller clip.');
    await previewPayload();
  } catch (err) {
    setSubmitProgress(false, '');
    showToast('Clip resize failed for ' + id + '.', true);
    logNote('Clip resize failed: ' + err.message, true);
  } finally {
    clipResizeInFlight.delete(id);
    if (latestPayloadPreviewData) {
      renderPayloadPreview(latestPayloadPreviewData);
    }
  }
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

function renderPreviewContext(note) {
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

function previewNoteByID(eventID) {
  const notes = Array.isArray(latestPayloadPreviewData?.preview?.notes) ? latestPayloadPreviewData.preview.notes : [];
  return notes.find(note => String(note?.event_id || '') === String(eventID || '')) || null;
}

function syncPreviewSessionState(nextSession) {
  if (!nextSession) return;
  currentState = currentState ? { ...currentState, session: nextSession } : { session: nextSession };
  syncReplayValueCaptureUI();
  if (stateEl) {
    stateEl.textContent = JSON.stringify(currentState, null, 2);
  }
}

function resetPreviewDeliveryOptions() {
  previewDeliveryOptions = { redactReplayValues: false, omitVideoClips: false, omitVideoEventIDs: [] };
}

function syncReplayValueCaptureUI() {
  if (!captureInputValuesToggleEl) return;
  const enabled = !!(currentState?.session?.capture_input_values);
  captureInputValuesToggleEl.checked = enabled;
  captureInputValuesToggleEl.disabled = !currentState?.session?.id;
  captureInputValuesToggleEl.title = enabled
    ? 'Typed values will be included in replay bundles for this session.'
    : 'Typed values stay redacted unless you opt in for this session.';
  renderSensitiveCaptureBadges();
}

async function toggleReplayValueCapture() {
  if (!captureInputValuesToggleEl) return;
  if (!currentState?.session?.id) {
    captureInputValuesToggleEl.checked = false;
    logNote('Start a session before changing replay value capture.', true);
    return;
  }
  try {
    const enabled = !!captureInputValuesToggleEl.checked;
    const data = await post('/api/session/replay/settings', { capture_input_values: enabled });
    syncPreviewSessionState(data?.session);
    logNote(enabled ? 'Replay bundles will include typed values for this session.' : 'Replay bundles will redact typed values for this session.');
    renderSensitiveCaptureBadges();
  } catch (err) {
    captureInputValuesToggleEl.checked = !!(currentState?.session?.capture_input_values);
    logNote('Replay capture update failed: ' + err.message, true);
  }
}

function renderReplayBundle(note) {
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
    parts.push('<details class="console-card" style="margin-top:.65rem;"><summary>Playwright script</summary><pre>' + escapePreviewHTML(script) + '</pre></details>');
  }
  return parts.join('');
}

async function exportReplayBundle(eventID, format) {
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
    logNote('Replay export downloaded: ' + a.download + '.');
  } catch (err) {
    logNote('Replay export failed: ' + err.message, true);
  }
}

async function editPreviewNote(eventID) {
  try {
    if (isNoteRecordingActive()) throw new Error('Stop the current recording before editing a change request.');
    const note = previewNoteByID(eventID);
    if (!note) throw new Error('Change request not found in the current preview.');
    const nextText = window.prompt('Edit change request text:', String(note.text || ''));
    if (nextText === null) return;
    const trimmed = String(nextText || '').trim();
    if (!trimmed) {
      throw new Error('Change request text cannot be empty. Delete the request instead if you want to remove it.');
    }
    const data = await post('/api/session/feedback/update-text', { event_id: eventID, text: trimmed });
    syncPreviewSessionState(data?.session);
    await previewPayload();
    logNote('Change request updated.');
  } catch (err) {
    logNote('Change request update failed: ' + err.message, true);
  }
}

async function deletePreviewNote(eventID) {
  try {
    if (isNoteRecordingActive()) throw new Error('Stop the current recording before deleting a change request.');
    const note = previewNoteByID(eventID);
    if (!note) throw new Error('Change request not found in the current preview.');
    const confirmed = window.confirm('Delete this change request?\n\n' + String(note.text || ''));
    if (!confirmed) return;
    const data = await post('/api/session/feedback/delete', { event_id: eventID });
    syncPreviewSessionState(data?.session);
    const feedbackCount = Array.isArray(currentState?.session?.feedback) ? currentState.session.feedback.length : 0;
    if (feedbackCount > 0) {
      await previewPayload();
    } else {
      latestPayloadPreviewData = null;
      renderPayloadPreview({ preview: { notes: [] } });
    }
    logNote('Change request deleted.');
  } catch (err) {
    logNote('Change request delete failed: ' + err.message, true);
  }
}

function renderPayloadPreview(data) {
  if (!payloadPreviewEl) return;
  latestPayloadPreviewData = data || null;
  const preview = data?.preview;
  const notes = Array.isArray(preview?.notes) ? preview.notes : [];
  if (notes.length === 0) {
    resetPreviewDeliveryOptions();
    payloadPreviewEl.className = 'request-preview empty-tone';
    payloadPreviewEl.textContent = 'Preview the request to inspect what Knit will send.';
    return;
  }
  const provider = String(data?.provider || '');
  const destination = providerDestinationLabel(provider);
  const summary = String(preview?.summary || '');
  const count = Math.max(1, Number(preview?.change_request_count || notes.length));
  const intentLabel = String(preview?.intent_label || 'Implement changes');
  const warnings = Array.isArray(preview?.warnings) ? preview.warnings : [];
  const disclosureBlock = renderDisclosureSummary(preview);
  const oversizedVideoBlock = renderOversizedVideoWarningActions(preview);
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
      '<div class="preview-note-header"><strong>Change request ' + (index + 1) + '</strong><span class="empty-tone">' + escapePreviewHTML(note?.event_id || '') + '</span></div>' +
      (meta.length ? '<div class="preview-note-meta">' + meta.map(item => '<span>' + item + '</span>').join('') + '</div>' : '') +
      '<div class="preview-note-text">' + escapePreviewHTML(note?.text || '') + '</div>' +
      renderPreviewContext(note) +
      renderReplayBundle(note) +
      '<div class="sub-actions">' +
      '<button type="button" class="secondary" onclick="editPreviewNote(\'' + escapePreviewHTML(eventID) + '\')" title="Edit change request text">Edit text</button>' +
      '<button type="button" class="danger" onclick="deletePreviewNote(\'' + escapePreviewHTML(eventID) + '\')" title="Delete change request">Delete</button>' +
      '<button type="button" class="secondary" onclick="exportReplayBundle(\'' + escapePreviewHTML(eventID) + '\', \'json\')" title="Export replay JSON">Export replay JSON</button>' +
      '<button type="button" class="secondary" onclick="exportReplayBundle(\'' + escapePreviewHTML(eventID) + '\', \'playwright\')" title="Export Playwright script">Export Playwright script</button>' +
      '</div>' +
      renderPreviewVideoDecisionActions(note) +
      renderPreviewMedia(note) +
      '</article>';
  }).join('');
  const warningBlock = warnings.length
    ? '<div class="preview-warning-card"><strong>Review media before sending.</strong><div class="preview-note-text">' + warnings.map(item => escapePreviewHTML(item)).join('<br/>') + '</div><div class="small" style="margin-top:.45rem;color:#6a7383;">Choose how to proceed on the affected request: make the clip smaller, rely on a screenshot, or explicitly allow the larger clip.</div></div>'
    : '';
  payloadPreviewEl.className = 'request-preview';
  payloadPreviewEl.innerHTML = '<div class="preview-summary-card">' +
    '<div class="preview-kicker">Ready to send</div>' +
    '<div class="preview-summary-line"><strong>' + count + ' request' + (count === 1 ? '' : 's') + ' prepared</strong><span class="empty-tone">' + escapePreviewHTML(destination) + '</span></div>' +
    '<div class="preview-note-meta"><span>Action: ' + escapePreviewHTML(intentLabel) + '</span></div>' +
    (summary ? '<div class="preview-note-text">' + escapePreviewHTML(summary) + '</div>' : '') +
    '</div>' +
    disclosureBlock +
    oversizedVideoBlock +
    noteCards +
    warningBlock;
}

async function approveSession(silent = false, reason = 'submit') {
  try {
    const data = await post('/api/session/approve', { summary: '' });
    if (!silent) {
      logNote('Prepared ' + (data.change_requests?.length || 0) + ' change requests for ' + reason + '.');
    }
    return data;
  } catch (err) {
    if (!silent) {
      logNote('Prepare failed: ' + err.message, true);
    }
    throw err;
  }
}

async function previewPayload() {
  try {
    if (isNoteRecordingActive()) throw new Error('Stop the current recording before previewing the request.');
    const provider = selectedProvider();
    const intentProfile = selectedDeliveryIntentProfile();
    const instructionText = selectedDeliveryInstructionText();
    if (!currentState?.session?.id) throw new Error('Start a session first.');
    await flushTypedNoteDraft('preview');
    const feedbackCount = Array.isArray(currentState?.session?.feedback) ? currentState.session.feedback.length : 0;
    if (feedbackCount === 0) throw new Error('Capture at least one feedback note first.');
    await approveSession(true, 'preview');
    const data = await post('/api/session/payload/preview', {
      provider,
      intent_profile: intentProfile,
      instruction_text: instructionText,
      allow_large_inline_media: !!allowLargeInlineMediaEl?.checked,
      redact_replay_values: !!previewDeliveryOptions.redactReplayValues,
      omit_video_clips: !!previewDeliveryOptions.omitVideoClips,
      omit_video_event_ids: previewDeliveryOptions.omitVideoEventIDs || []
    });
    renderPayloadPreview(data);
    logNote('Payload preview generated for ' + providerDestinationLabel(provider) + ' with action "' + selectedDeliveryIntentLabel() + '" (snapshot prepared automatically).');
  } catch (err) {
    logNote('Payload preview failed: ' + err.message, true);
  }
}

async function submitSession() {
  const provider = selectedProvider();
  const intentProfile = selectedDeliveryIntentProfile();
  const intentLabel = selectedDeliveryIntentLabel();
  const instructionText = selectedDeliveryInstructionText();
  setSubmitProgress(true, 'Preparing snapshot and queueing submission to ' + providerDestinationLabel(provider) + '...');
  try {
    if (isNoteRecordingActive()) throw new Error('Stop the current recording before sending the request.');
    if (!currentState?.session?.id) throw new Error('Start a session first.');
    await flushTypedNoteDraft('send');
    const feedbackCount = Array.isArray(currentState?.session?.feedback) ? currentState.session.feedback.length : 0;
    if (feedbackCount === 0) throw new Error('Capture at least one feedback note first.');
    await approveSession(true, 'submit');
    const data = await post('/api/session/submit', {
      provider,
      intent_profile: intentProfile,
      instruction_text: instructionText,
      allow_large_inline_media: !!allowLargeInlineMediaEl?.checked,
      redact_replay_values: !!previewDeliveryOptions.redactReplayValues,
      omit_video_clips: !!previewDeliveryOptions.omitVideoClips,
      omit_video_event_ids: previewDeliveryOptions.omitVideoEventIDs || []
    });
    const attemptId = data.attempt_id || '';
    const queuePos = Number(data.queue_position || 0);
    const status = data.status || 'queued';
    if (attemptId) watchedSubmitAttemptIDs.add(String(attemptId));
    setSubmitProgress(false, 'Queued: ' + (attemptId || 'submission') + (queuePos > 0 ? (' (position ' + queuePos + ')') : ''));
    const destination = providerDestinationLabel(provider);
    logNote('Submission queued to ' + destination + ' as ' + (attemptId || 'attempt') + ' (' + status + ', action "' + intentLabel + '").');
    showToast('Request submitted to ' + destination + ' for "' + intentLabel + '"' + (attemptId ? (' as ' + attemptId) : '.'));
    resetPreviewDeliveryOptions();
  } catch (err) {
    const msg = String(err.message || '');
    if (msg.includes('session must be explicitly approved before submission') || msg.includes('session not approved')) {
      setSubmitProgress(false, 'Submission blocked: snapshot preparation failed.');
      logNote('Submit failed while preparing snapshot. Retry Submit.', true);
      return;
    }
    if (msg.includes('over the default send limit')) {
      setSubmitProgress(false, 'Submission blocked until you choose how to handle the large clip.');
      await previewPayload();
      logNote('Send blocked by the inline clip limit. Use “Make clip smaller to send,” “Use screenshot instead,” or allow large inline media.', true);
      return;
    }
    setSubmitProgress(false, 'Submission failed: ' + msg);
    logNote('Submit failed: ' + msg, true);
  }
}

async function openLastLog() {
  try {
    closeCaptureSettingsModal();
    const data = await post('/api/session/open-last-log', {});
    const path = data.path || '';
    setSubmitProgress(false, path ? ('Opened log: ' + path) : 'Opened last log.');
    logNote(path ? ('Opened last log: ' + path) : 'Opened last log.');
  } catch (err) {
    setSubmitProgress(false, 'Open last log failed: ' + err.message);
    logNote('Open last log failed: ' + err.message, true);
  }
}

async function exportConfig() {
  try {
    const res = await fetch('/api/config/export', { headers: authHeaders(false) });
    const data = await res.json();
    configExportEl.textContent = String(data?.config_toml || JSON.stringify(data, null, 2));
    logNote('Config export refreshed.');
  } catch (err) {
    logNote('Config export failed: ' + err.message, true);
  }
}

async function applyProfile() {
  try {
    if (currentState?.config_locked) throw new Error('Config is locked by policy.');
    const profile = configProfileEl.value;
    await post('/api/config/import', { profile });
    await exportConfig();
    logNote('Applied profile: ' + profile);
  } catch (err) {
    logNote('Apply profile failed: ' + err.message, true);
  }
}

async function applyWorkspaceDir() {
  codexWorkdirLabelEl.textContent = workspaceDirEl.value.trim() || '(not set)';
  setUISetting('workspace_dir', workspaceDirEl.value.trim());
  await applyCodexRuntime();
  updateWorkspaceModalState();
  if (!workspaceSelectionRequired) {
    closeWorkspaceModal();
  }
}

async function pickWorkspaceDir() {
  try {
    if (currentState?.config_locked) throw new Error('Config is locked by policy.');
    const picked = await post('/api/fs/pickdir', {});
    const path = (picked && picked.path) ? String(picked.path) : '';
    if (!path) throw new Error('No folder selected.');
    workspaceDirEl.value = path;
    codexWorkdirLabelEl.textContent = path;
    setUISetting('workspace_dir', path);
    workspaceBrowserStateEl.textContent = JSON.stringify({ selected_workspace: path, picker: 'native folder dialog' }, null, 2);
    updateWorkspaceModalState();
    await applyCodexRuntime();
    closeWorkspaceModal();
    logNote('Workspace selected: ' + path);
  } catch (err) {
    logNote('Choose folder failed: ' + err.message, true);
  }
}

async function applyCodexRuntime() {
  if (codexRuntimeApplyTimer) {
    clearTimeout(codexRuntimeApplyTimer);
    codexRuntimeApplyTimer = 0;
  }
  codexRuntimeApplying = true;
  try {
    if (currentState?.config_locked) throw new Error('Config is locked by policy.');
    const timeoutSeconds = readRuntimeSeconds(cliTimeoutSecondsEl, 'Codex CLI timeout');
    const claudeTimeoutSeconds = readRuntimeSeconds(claudeCliTimeoutSecondsEl, 'Claude CLI timeout');
    const opencodeTimeoutSeconds = readRuntimeSeconds(opencodeCliTimeoutSecondsEl, 'OpenCode CLI timeout');
    const codexAPITimeoutSeconds = readRuntimeSeconds(codexAPITimeoutSecondsEl, 'Codex API timeout');
    const claudeAPITimeoutSeconds = readRuntimeSeconds(claudeAPITimeoutSecondsEl, 'Claude API timeout');
    const postTimeoutSeconds = readRuntimeSeconds(postSubmitTimeoutSecEl, 'Post-submit timeout', 7200);
    const deliveryPromptPayload = currentDeliveryPromptPayload();
    const payload = {
      default_provider: selectedProvider(),
      cli_adapter_cmd: readRuntimeSingleLine(codexCliCmdEl, 'Codex CLI command'),
      cli_timeout_seconds: timeoutSeconds,
      claude_cli_adapter_cmd: readRuntimeSingleLine(claudeCliCmdEl, 'Claude CLI command'),
      claude_cli_timeout_seconds: claudeTimeoutSeconds,
      opencode_cli_adapter_cmd: readRuntimeSingleLine(opencodeCliCmdEl, 'OpenCode CLI command'),
      opencode_cli_timeout_seconds: opencodeTimeoutSeconds,
      submit_execution_mode: submitExecutionModeEl.value || 'series',
      codex_workdir: readRuntimeSingleLine(workspaceDirEl, 'Workspace path'),
      codex_output_dir: readRuntimeSingleLine(codexOutputDirEl, 'Output directory', 1024),
      codex_sandbox: codexSandboxEl.value,
      codex_approval_policy: codexApprovalEl.value,
      codex_profile: readRuntimeSingleLine(codexProfileEl, 'Codex profile', 128),
      codex_model: readRuntimeSingleLine(codexModelEl, 'Codex model', 128),
      codex_reasoning_effort: readRuntimeSingleLine(codexReasoningEl, 'Codex reasoning effort', 64),
      openai_base_url: readRuntimeURL(codexAPIBaseURLEl, 'Codex API base URL'),
      codex_api_timeout_seconds: codexAPITimeoutSeconds,
      openai_org_id: readRuntimeSingleLine(codexAPIOrgEl, 'OpenAI org ID', 256),
      openai_project_id: readRuntimeSingleLine(codexAPIProjectEl, 'OpenAI project ID', 256),
      anthropic_base_url: readRuntimeURL(claudeAPIBaseURLEl, 'Claude API base URL'),
      claude_api_timeout_seconds: claudeAPITimeoutSeconds,
      claude_api_model: readRuntimeSingleLine(claudeAPIModelEl, 'Claude API model', 128),
      delivery_intent_profile: deliveryPromptPayload.delivery_intent_profile,
      implement_changes_prompt: deliveryPromptPayload.implement_changes_prompt,
      draft_plan_prompt: deliveryPromptPayload.draft_plan_prompt,
      create_jira_tickets_prompt: deliveryPromptPayload.create_jira_tickets_prompt,
      post_submit_rebuild_cmd: readRuntimeSingleLine(postSubmitRebuildCmdEl, 'Post-submit rebuild command'),
      post_submit_verify_cmd: readRuntimeSingleLine(postSubmitVerifyCmdEl, 'Post-submit verify command'),
      post_submit_timeout_seconds: postTimeoutSeconds,
      codex_skip_git_repo_check: !!codexSkipRepoCheckEl.checked
    };
    const res = await post('/api/runtime/codex', payload);
    codexRuntimeStateEl.textContent = JSON.stringify(res.runtime_codex || payload, null, 2);
    codexWorkdirLabelEl.textContent = payload.codex_workdir || '(not set)';
    codexRuntimeDirty = false;
    setUISetting('default_provider', payload.default_provider || '');
    setUISetting('cli_adapter_cmd', payload.cli_adapter_cmd || '');
    setUISetting('claude_cli_adapter_cmd', payload.claude_cli_adapter_cmd || '');
    setUISetting('opencode_cli_adapter_cmd', payload.opencode_cli_adapter_cmd || '');
    setUISetting('cli_timeout_seconds', Number.isFinite(timeoutSeconds) && timeoutSeconds > 0 ? String(timeoutSeconds) : '');
    setUISetting('claude_cli_timeout_seconds', Number.isFinite(claudeTimeoutSeconds) && claudeTimeoutSeconds > 0 ? String(claudeTimeoutSeconds) : '');
    setUISetting('opencode_cli_timeout_seconds', Number.isFinite(opencodeTimeoutSeconds) && opencodeTimeoutSeconds > 0 ? String(opencodeTimeoutSeconds) : '');
    setUISetting('submit_execution_mode', payload.submit_execution_mode || 'series');
    setUISetting('codex_output_dir', payload.codex_output_dir || '');
    setUISetting('codex_sandbox', payload.codex_sandbox || '');
    setUISetting('codex_approval_policy', payload.codex_approval_policy || '');
    setUISetting('codex_skip_git_repo_check', !!payload.codex_skip_git_repo_check);
    setUISetting('codex_profile', payload.codex_profile || '');
    setUISetting('codex_model', payload.codex_model || '');
    setUISetting('codex_reasoning_effort', payload.codex_reasoning_effort || '');
    setUISetting('openai_base_url', payload.openai_base_url || '');
    setUISetting('codex_api_timeout_seconds', Number.isFinite(codexAPITimeoutSeconds) && codexAPITimeoutSeconds > 0 ? String(codexAPITimeoutSeconds) : '');
    setUISetting('openai_org_id', payload.openai_org_id || '');
    setUISetting('openai_project_id', payload.openai_project_id || '');
    setUISetting('anthropic_base_url', payload.anthropic_base_url || '');
    setUISetting('claude_api_timeout_seconds', Number.isFinite(claudeAPITimeoutSeconds) && claudeAPITimeoutSeconds > 0 ? String(claudeAPITimeoutSeconds) : '');
    setUISetting('claude_api_model', payload.claude_api_model || '');
    setUISetting('delivery_intent_profile', payload.delivery_intent_profile || 'implement_changes');
    setUISetting('delivery_instruction_text', selectedDeliveryInstructionText());
    setUISetting('post_submit_rebuild_cmd', payload.post_submit_rebuild_cmd || '');
    setUISetting('post_submit_verify_cmd', payload.post_submit_verify_cmd || '');
    setUISetting('post_submit_timeout_seconds', Number.isFinite(postTimeoutSeconds) && postTimeoutSeconds > 0 ? String(postTimeoutSeconds) : '');
    setCodexRuntimeStatus('Runtime settings saved automatically.');
    renderCaptureAgentNotice();
    syncCodexRuntimeModeUI();
    logNote('Agent runtime settings updated.');
  } catch (err) {
    codexRuntimeDirty = false;
    setCodexRuntimeStatus('Runtime settings could not be saved: ' + err.message, true);
    logNote('Runtime update failed: ' + err.message, true);
  } finally {
    codexRuntimeApplying = false;
  }
}

preview.addEventListener('mousedown', (e) => {
  if (screenshotMode.value !== 'selected-region') return;
  const rect = preview.getBoundingClientRect();
  selecting = true;
  selectionStart = { x: e.clientX - rect.left, y: e.clientY - rect.top };
  selectionRect = null;
});

preview.addEventListener('mousemove', (e) => {
  if (!selecting || !selectionStart) return;
  const rect = preview.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  selectionRect = {
    x: Math.max(0, Math.min(selectionStart.x, x)),
    y: Math.max(0, Math.min(selectionStart.y, y)),
    w: Math.abs(x - selectionStart.x),
    h: Math.abs(y - selectionStart.y)
  };
});

preview.addEventListener('mousemove', (e) => {
  if (!laserModeEnabledEl?.checked) return;
  const rect = preview.getBoundingClientRect();
  const x = Math.max(0, e.clientX - rect.left);
  const y = Math.max(0, e.clientY - rect.top);
  laserTrail.push({ x, y, t: Date.now() });
  if (laserTrail.length > laserTrailMax) {
    laserTrail = laserTrail.slice(laserTrail.length - laserTrailMax);
  }
});

preview.addEventListener('mouseup', () => {
  if (!selecting) return;
  selecting = false;
  if (selectionRect && selectionRect.w > 4 && selectionRect.h > 4) {
    logNote('Selected screenshot region set.');
  } else {
    selectionRect = null;
  }
});

window.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && captureSettingsModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeCaptureSettingsModal();
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
    closeVideoCaptureModal();
    return;
  }
  if (e.key === 'Escape' && codexRuntimeModalEl?.classList.contains('open')) {
    e.preventDefault();
    closeCodexRuntimeModal();
    return;
  }
  if (e.code === 'Space' && audioModeEl && audioModeEl.value === 'push_to_talk') {
    setPTT(true);
  }
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'S' || e.key === 's')) {
    e.preventDefault();
    captureManualScreenshot();
  }
});

window.addEventListener('keyup', (e) => {
  if (e.code === 'Space') {
    setPTT(false);
  }
});

window.addEventListener('blur', () => {
  setPTT(false);
});

if (audioModeEl) {
  audioModeEl.addEventListener('change', () => {
    if (audioModeEl.value !== 'push_to_talk') {
      setPTT(false);
    }
    scheduleAudioConfigApply();
  });
}
if (audioInputDeviceEl) {
  audioInputDeviceEl.addEventListener('change', scheduleAudioConfigApply);
}
if (audioMutedEl) {
  audioMutedEl.addEventListener('change', scheduleAudioConfigApply);
}
if (audioPausedEl) {
  audioPausedEl.addEventListener('change', scheduleAudioConfigApply);
}
if (sttModeEl) {
  sttModeEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttBaseURLEl) {
  sttBaseURLEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttBaseURLEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttModelEl) {
  sttModelEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttModelEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttFasterWhisperModelEl) {
  sttFasterWhisperModelEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttFasterWhisperModelEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttDeviceEl) {
  sttDeviceEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttDeviceEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttComputeTypeEl) {
  sttComputeTypeEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttComputeTypeEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttLanguageEl) {
  sttLanguageEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttLanguageEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttLocalCommandEl) {
  sttLocalCommandEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttLocalCommandEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (sttTimeoutSecondsEl) {
  sttTimeoutSecondsEl.addEventListener('input', scheduleTranscriptionRuntimeApply);
  sttTimeoutSecondsEl.addEventListener('change', scheduleTranscriptionRuntimeApply);
}
if (agentDefaultProviderEl) {
  agentDefaultProviderEl.addEventListener('change', () => {
    renderCaptureAgentNotice();
    scheduleCodexRuntimeApply();
  });
}
if (codexCliCmdEl) {
  codexCliCmdEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexCliCmdEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (claudeCliCmdEl) {
  claudeCliCmdEl.addEventListener('input', scheduleCodexRuntimeApply);
  claudeCliCmdEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (opencodeCliCmdEl) {
  opencodeCliCmdEl.addEventListener('input', scheduleCodexRuntimeApply);
  opencodeCliCmdEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (cliTimeoutSecondsEl) {
  cliTimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApply);
  cliTimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (claudeCliTimeoutSecondsEl) {
  claudeCliTimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApply);
  claudeCliTimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (opencodeCliTimeoutSecondsEl) {
  opencodeCliTimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApply);
  opencodeCliTimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexOutputDirEl) {
  codexOutputDirEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexOutputDirEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (submitExecutionModeEl) {
  submitExecutionModeEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexSandboxEl) {
  codexSandboxEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexApprovalEl) {
  codexApprovalEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexSkipRepoCheckEl) {
  codexSkipRepoCheckEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexProfileEl) {
  codexProfileEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexProfileEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexModelEl) {
  codexModelEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexReasoningEl) {
  codexReasoningEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexAPIBaseURLEl) {
  codexAPIBaseURLEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexAPIBaseURLEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexAPITimeoutSecondsEl) {
  codexAPITimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexAPITimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexAPIOrgEl) {
  codexAPIOrgEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexAPIOrgEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (codexAPIProjectEl) {
  codexAPIProjectEl.addEventListener('input', scheduleCodexRuntimeApply);
  codexAPIProjectEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (claudeAPIBaseURLEl) {
  claudeAPIBaseURLEl.addEventListener('input', scheduleCodexRuntimeApply);
  claudeAPIBaseURLEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (claudeAPITimeoutSecondsEl) {
  claudeAPITimeoutSecondsEl.addEventListener('input', scheduleCodexRuntimeApply);
  claudeAPITimeoutSecondsEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (claudeAPIModelEl) {
  claudeAPIModelEl.addEventListener('input', scheduleCodexRuntimeApply);
  claudeAPIModelEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (deliveryIntentProfileEl) {
  deliveryIntentProfileEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (deliveryInstructionTextEl) {
  deliveryInstructionTextEl.addEventListener('input', scheduleCodexRuntimeApply);
  deliveryInstructionTextEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (postSubmitRebuildCmdEl) {
  postSubmitRebuildCmdEl.addEventListener('input', scheduleCodexRuntimeApply);
  postSubmitRebuildCmdEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (postSubmitVerifyCmdEl) {
  postSubmitVerifyCmdEl.addEventListener('input', scheduleCodexRuntimeApply);
  postSubmitVerifyCmdEl.addEventListener('change', scheduleCodexRuntimeApply);
}
if (postSubmitTimeoutSecEl) {
  postSubmitTimeoutSecEl.addEventListener('input', scheduleCodexRuntimeApply);
  postSubmitTimeoutSecEl.addEventListener('change', scheduleCodexRuntimeApply);
}

setInterval(refresh, 1500);
initPersistentSettings();
syncDeliveryIntentPromptText(true);
renderCaptureAgentNotice();
syncSTTRuntimeModeUI();
setCaptureGuideSidebarOpen(!hasUISetting('capture_guide_open') || !!uiSettings.capture_guide_open);
refresh();
refreshAudioDevices();
exportConfig();
</script>
</body>
</html>`
