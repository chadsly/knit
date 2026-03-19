package server

const docsBrowserHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Knit Docs</title>
  <style>
    :root {
      --bg: #f5efe5;
      --bg-accent: #fff9f1;
      --panel: rgba(255, 252, 247, 0.94);
      --panel-solid: #fffdf9;
      --text: #1a2431;
      --muted: #697487;
      --accent: #1c7c74;
      --accent-strong: #155b56;
      --accent-soft: #dff2ee;
      --line: #e5dccf;
      --line-strong: #d8cab7;
      --shadow: 0 22px 58px rgba(27, 39, 59, 0.10);
      --code-bg: #efe7da;
      --sidebar-width: 290px;
      --outline-width: 240px;
    }
    html[data-theme="dark"] {
      --bg: #0f1722;
      --bg-accent: #121d2b;
      --panel: rgba(18, 28, 42, 0.94);
      --panel-solid: #142033;
      --text: #eef4fb;
      --muted: #aab7ca;
      --accent: #62c6bc;
      --accent-strong: #8de0d7;
      --accent-soft: rgba(98, 198, 188, 0.16);
      --line: rgba(157, 178, 205, 0.18);
      --line-strong: rgba(157, 178, 205, 0.28);
      --shadow: 0 26px 66px rgba(3, 8, 15, 0.44);
      --code-bg: #192536;
    }
    * { box-sizing: border-box; }
    html { scroll-behavior: smooth; }
    body {
      margin: 0;
      font-family: "Inter", "SF Pro Text", "Segoe UI", sans-serif;
      background:
        radial-gradient(1200px 640px at 6% -12%, rgba(28, 124, 116, 0.10), transparent 55%),
        radial-gradient(900px 540px at 96% 0%, rgba(217, 189, 146, 0.16), transparent 44%),
        linear-gradient(180deg, var(--bg-accent) 0%, var(--bg) 100%);
      color: var(--text);
      line-height: 1.65;
    }
    .shell {
      max-width: 1500px;
      margin: 0 auto;
      padding: 1rem 1rem 1.5rem;
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: .95rem 1rem;
      box-shadow: var(--shadow);
      backdrop-filter: blur(14px);
      margin-bottom: 1rem;
    }
    .topbar-main {
      display: flex;
      flex-direction: column;
      gap: .2rem;
      min-width: 0;
    }
    .topbar-kicker {
      display: inline-flex;
      align-items: center;
      gap: .45rem;
      align-self: flex-start;
      font-size: .76rem;
      font-weight: 700;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent-strong);
      background: var(--accent-soft);
      border-radius: 999px;
      padding: .34rem .68rem;
    }
    .topbar-title {
      font-size: 1.5rem;
      line-height: 1.1;
      letter-spacing: -0.03em;
      font-weight: 780;
    }
    .topbar-copy {
      color: var(--muted);
      font-size: .95rem;
      max-width: 62ch;
    }
    .topbar-actions {
      display: flex;
      align-items: center;
      gap: .6rem;
      flex-wrap: wrap;
    }
    .layout {
      display: grid;
      grid-template-columns: var(--sidebar-width) minmax(0, 1fr) var(--outline-width);
      gap: 1rem;
      align-items: start;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(14px);
      min-width: 0;
    }
    .sidebar,
    .outline {
      position: sticky;
      top: 1rem;
      padding: 1rem;
    }
    .sidebar-title,
    .outline-title {
      font-size: .92rem;
      font-weight: 760;
      margin-bottom: .15rem;
    }
    .sidebar-copy,
    .outline-copy {
      color: var(--muted);
      font-size: .88rem;
      margin-bottom: .85rem;
    }
    .doc-list,
    .outline-list {
      display: grid;
      gap: .7rem;
    }
    .doc-card {
      appearance: none;
      width: 100%;
      text-align: left;
      border: 1px solid var(--line);
      border-radius: 18px;
      background: var(--panel-solid);
      color: var(--text);
      padding: .95rem;
      cursor: pointer;
      transition: transform .14s ease, border-color .14s ease, box-shadow .14s ease, background .14s ease;
    }
    .doc-card:hover {
      transform: translateY(-1px);
      border-color: var(--accent);
      box-shadow: 0 14px 28px rgba(24, 35, 48, 0.10);
    }
    .doc-card.active {
      border-color: var(--accent);
      background: linear-gradient(180deg, var(--panel-solid), var(--accent-soft));
    }
    .doc-card-title {
      display: block;
      font-weight: 760;
      font-size: 1rem;
      line-height: 1.25;
    }
    .article {
      padding: 1.2rem 1.2rem 1.6rem;
    }
    .article-shell {
      max-width: 76ch;
      margin: 0 auto;
    }
    .article-intro {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      margin-bottom: 1.2rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--line);
    }
    .article-kicker {
      display: inline-flex;
      align-items: center;
      gap: .35rem;
      font-size: .75rem;
      font-weight: 760;
      letter-spacing: .08em;
      text-transform: uppercase;
      color: var(--accent-strong);
      background: var(--accent-soft);
      border-radius: 999px;
      padding: .32rem .62rem;
      margin-bottom: .6rem;
    }
    .article-description {
      color: var(--muted);
      margin-top: 0;
      max-width: 62ch;
      font-size: .98rem;
    }
    .article-meta {
      display: inline-flex;
      align-items: center;
      gap: .45rem;
      flex-wrap: wrap;
      margin-top: .7rem;
      color: var(--muted);
      font-size: .84rem;
    }
    .article-meta code {
      background: var(--code-bg);
      border-radius: 8px;
      padding: .16rem .4rem;
      font-family: ui-monospace, "SFMono-Regular", Consolas, monospace;
    }
    .article-actions {
      display: flex;
      gap: .55rem;
      flex-wrap: wrap;
      flex: 0 0 auto;
    }
    .article-body {
      font-size: 1rem;
    }
    .article-body h1,
    .article-body h2,
    .article-body h3,
    .article-body h4 {
      scroll-margin-top: 92px;
      line-height: 1.14;
      letter-spacing: -0.025em;
      margin: 1.4rem 0 .55rem;
    }
    .article-body h1:first-child,
    .article-body h2:first-child,
    .article-body h3:first-child,
    .article-body h4:first-child {
      margin-top: 0;
    }
    .article-body h1 {
      font-size: clamp(2.2rem, 3vw, 3rem);
      margin-bottom: .7rem;
    }
    .article-body h2 {
      font-size: 1.7rem;
    }
    .article-body h3 {
      font-size: 1.3rem;
    }
    .article-body p {
      margin: .85rem 0;
      color: var(--text);
    }
    .article-body ul,
    .article-body ol {
      margin: .7rem 0 1rem 1.25rem;
      padding: 0;
    }
    .article-body li {
      margin: .42rem 0;
    }
    .article-body code {
      font-family: ui-monospace, "SFMono-Regular", Consolas, monospace;
      background: var(--code-bg);
      border-radius: 8px;
      padding: .12rem .36rem;
      font-size: .92em;
    }
    .article-body pre {
      margin: 1rem 0 1.2rem;
      background: var(--code-bg);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 1rem 1.05rem;
      overflow: auto;
      max-width: 100%;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-word;
      font-size: .92rem;
      line-height: 1.55;
    }
    .article-body pre code {
      background: transparent;
      padding: 0;
      border-radius: 0;
    }
    .doc-image {
      margin: 1rem 0 1.2rem;
    }
    .doc-image img {
      display: block;
      width: min(100%, 520px);
      max-width: 100%;
      margin: 0 auto;
      border-radius: 18px;
      border: 1px solid var(--line);
      background: var(--panel-solid);
      box-shadow: 0 14px 32px rgba(24, 35, 48, 0.10);
    }
    .doc-image figcaption {
      color: var(--muted);
      font-size: .85rem;
      margin-top: .45rem;
      text-align: center;
    }
    .doc-tabs {
      margin: 1rem 0 1.2rem;
      border: 1px solid var(--line);
      border-radius: 18px;
      background: linear-gradient(180deg, var(--panel-solid), var(--panel));
      overflow: hidden;
    }
    .doc-tab-list {
      display: flex;
      gap: .45rem;
      flex-wrap: wrap;
      padding: .7rem;
      border-bottom: 1px solid var(--line);
      background: rgba(0, 0, 0, 0.02);
    }
    .doc-tab {
      padding: .55rem .8rem;
      border-radius: 999px;
      border: 1px solid var(--line);
      background: var(--panel-solid);
      color: var(--muted);
      font-size: .86rem;
      font-weight: 720;
      box-shadow: none;
    }
    .doc-tab:hover {
      box-shadow: none;
    }
    .doc-tab.active {
      background: var(--accent);
      border-color: var(--accent);
      color: #fff;
    }
    .doc-tab-panels {
      padding: 1rem;
    }
    .doc-tab-panel {
      display: none;
    }
    .doc-tab-panel.active {
      display: block;
    }
    .doc-tab-panel > *:first-child {
      margin-top: 0;
    }
    .doc-tab-panel > *:last-child {
      margin-bottom: 0;
    }
    .outline-list a {
      display: block;
      color: var(--muted);
      text-decoration: none;
      border-radius: 12px;
      padding: .42rem .5rem;
      transition: background .14s ease, color .14s ease;
      font-size: .88rem;
      line-height: 1.35;
    }
    .outline-list a:hover,
    .outline-list a.active {
      background: var(--accent-soft);
      color: var(--text);
    }
    .outline-list a.level-3,
    .outline-list a.level-4,
    .outline-list a.level-5,
    .outline-list a.level-6 {
      padding-left: 1rem;
    }
    .outline-empty,
    .article-empty,
    .loading {
      color: var(--muted);
      font-size: .93rem;
    }
    button, a.button-link {
      appearance: none;
      border: 1px solid var(--line-strong);
      border-radius: 16px;
      padding: .72rem .95rem;
      background: var(--panel-solid);
      color: var(--text);
      text-decoration: none;
      font-weight: 700;
      font-size: .92rem;
      cursor: pointer;
      transition: transform .14s ease, border-color .14s ease, box-shadow .14s ease, background .14s ease;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: .42rem;
    }
    button:hover, a.button-link:hover {
      transform: translateY(-1px);
      border-color: var(--accent);
      box-shadow: 0 12px 24px rgba(24, 35, 48, 0.10);
    }
    button.primary {
      background: var(--accent);
      border-color: var(--accent);
      color: #fff;
    }
    button.primary:hover {
      background: var(--accent-strong);
      border-color: var(--accent-strong);
    }
    button.icon-only {
      width: 3rem;
      height: 3rem;
      padding: 0;
      border-radius: 999px;
    }
    @media (max-width: 1260px) {
      .layout {
        grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
      }
      .outline {
        position: static;
        grid-column: 1 / -1;
      }
    }
    @media (max-width: 920px) {
      .layout {
        grid-template-columns: 1fr;
      }
      .sidebar,
      .outline {
        position: static;
      }
      .article-intro,
      .topbar {
        flex-direction: column;
        align-items: stretch;
      }
      .topbar-actions,
      .article-actions {
        width: 100%;
      }
      .topbar-actions > *,
      .article-actions > * {
        flex: 1 1 180px;
      }
    }
    @media (max-width: 720px) {
      .shell {
        padding: .75rem;
      }
      .panel,
      .topbar {
        border-radius: 20px;
      }
      .article {
        padding: 1rem;
      }
      .article-body h1 {
        font-size: 2rem;
      }
      button, a.button-link {
        width: 100%;
      }
      .topbar-actions {
        gap: .5rem;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <div class="topbar-main">
        <div class="topbar-kicker">Knit Docs</div>
        <div class="topbar-title">Operator guides for the local review workflow.</div>
        <div class="topbar-copy">Choose a guide from the library, skim the current document outline, and read in a focused column without raw filesystem noise crowding the page.</div>
      </div>
      <div class="topbar-actions">
        <button id="docsThemeToggleBtn" class="icon-only" type="button" title="Toggle theme" aria-label="Toggle theme">☾</button>
      </div>
    </header>

    <main class="layout">
      <aside class="panel sidebar">
        <div class="sidebar-title">Docs</div>
        <div class="sidebar-copy">Switch between guides here. The selected document stays in the URL so you can reopen it directly.</div>
        <div id="docsCatalog" class="doc-list">
          <div class="loading">Loading docs...</div>
        </div>
      </aside>

      <article class="panel article">
        <div class="article-shell">
          <div class="article-intro">
            <div>
              <div id="docsArticleKicker" class="article-kicker">Current doc</div>
              <p id="docsArticleDescription" class="article-description">Resolving the local docs catalog.</p>
              <div class="article-meta">
                <span>Source</span>
                <code id="docsArticleSource">loading...</code>
              </div>
            </div>
            <div class="article-actions">
              <button id="docsOpenCurrentTabBtn" type="button">↗ Open this doc in a new tab</button>
            </div>
          </div>
          <div id="docsArticleBody" class="article-body">
            <div class="loading">Loading docs...</div>
          </div>
        </div>
      </article>

      <aside class="panel outline">
        <div class="outline-title">On this page</div>
        <div class="outline-copy">Jump directly to the current document sections.</div>
        <nav id="docsOutline" class="outline-list">
          <div class="loading">Loading outline...</div>
        </nav>
      </aside>
    </main>
  </div>

  <script>
const defaultControlToken = '__KNIT_TOKEN__';
const docsCatalogEl = document.getElementById('docsCatalog');
const docsOutlineEl = document.getElementById('docsOutline');
const docsArticleKickerEl = document.getElementById('docsArticleKicker');
const docsArticleDescriptionEl = document.getElementById('docsArticleDescription');
const docsArticleSourceEl = document.getElementById('docsArticleSource');
const docsArticleBodyEl = document.getElementById('docsArticleBody');
const docsOpenCurrentTabBtnEl = document.getElementById('docsOpenCurrentTabBtn');
const docsThemeToggleBtnEl = document.getElementById('docsThemeToggleBtn');
const docsURLState = new URL(window.location.href);
const docsToken = docsURLState.searchParams.get('token') || defaultControlToken;
let docsCatalog = [];
let docsCurrentKey = '';
let docsCurrentHeadings = [];

function docsAuthHeaders() {
  return docsToken ? { 'X-Knit-Token': docsToken } : {};
}

function docsPageURL(name) {
  const url = new URL(window.location.origin + '/docs');
  if (docsToken) url.searchParams.set('token', docsToken);
  if (name) url.searchParams.set('name', name);
  return url.pathname + url.search;
}

function escapeHTML(value) {
  return String(value || '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function docsApplyTheme(theme) {
  const normalized = theme === 'dark' ? 'dark' : 'light';
  document.documentElement.setAttribute('data-theme', normalized);
  if (docsThemeToggleBtnEl) {
    docsThemeToggleBtnEl.textContent = normalized === 'dark' ? '☀' : '☾';
    docsThemeToggleBtnEl.title = normalized === 'dark' ? 'Switch to light theme' : 'Switch to dark theme';
    docsThemeToggleBtnEl.setAttribute('aria-label', docsThemeToggleBtnEl.title);
  }
}

function docsSavedTheme() {
  try {
    const raw = window.localStorage.getItem('knit_ui_settings_v1');
    const parsed = raw ? JSON.parse(raw) : {};
    const candidate = String(parsed?.theme || '').trim().toLowerCase();
    if (candidate === 'dark' || candidate === 'light') return candidate;
  } catch (_) {}
  return 'light';
}

function docsToggleTheme() {
  const next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
  docsApplyTheme(next);
  try {
    const raw = window.localStorage.getItem('knit_ui_settings_v1');
    const parsed = raw ? JSON.parse(raw) : {};
    parsed.theme = next;
    window.localStorage.setItem('knit_ui_settings_v1', JSON.stringify(parsed));
  } catch (_) {}
}

function docsRenderCatalog() {
  if (!docsCatalogEl) return;
  if (!Array.isArray(docsCatalog) || docsCatalog.length === 0) {
    docsCatalogEl.innerHTML = '<div class="article-empty">No local docs were found for this runtime.</div>';
    return;
  }
  docsCatalogEl.innerHTML = docsCatalog.map(function(doc) {
    const key = String(doc?.key || '');
    const active = key === docsCurrentKey ? ' active' : '';
    return '<button type="button" class="doc-card' + active + '" data-doc-key="' + escapeHTML(key) + '">' +
      '<span class="doc-card-title">' + escapeHTML(String(doc?.label || key || 'Doc')) + '</span>' +
    '</button>';
  }).join('');
  Array.from(docsCatalogEl.querySelectorAll('[data-doc-key]')).forEach(function(node) {
    node.addEventListener('click', function() {
      const key = String(node.getAttribute('data-doc-key') || '').trim();
      if (key) loadDoc(key, true);
    });
  });
}

function docsRenderOutline() {
  if (!docsOutlineEl) return;
  const sections = Array.isArray(docsCurrentHeadings) ? docsCurrentHeadings.filter(function(item) {
    const level = Number(item?.level || 0);
    return level >= 2 && level <= 4 && String(item?.id || '').trim() !== '';
  }) : [];
  if (sections.length === 0) {
    docsOutlineEl.innerHTML = '<div class="outline-empty">This document does not have section headings yet.</div>';
    return;
  }
  docsOutlineEl.innerHTML = sections.map(function(item) {
    const id = escapeHTML(String(item.id || ''));
    const text = escapeHTML(String(item.text || 'Section'));
    const level = Math.min(6, Math.max(2, Number(item.level || 2)));
    return '<a href="#' + id + '" class="level-' + level + '">' + text + '</a>';
  }).join('');
}

function initDocTabs() {
  Array.from(docsArticleBodyEl.querySelectorAll('[data-doc-tabs]')).forEach(function(group) {
    const buttons = Array.from(group.querySelectorAll('[data-doc-tab]'));
    const panels = Array.from(group.querySelectorAll('.doc-tab-panel'));
    if (buttons.length === 0 || panels.length === 0) return;
    const activate = function(targetID) {
      buttons.forEach(function(button, idx) {
        const isActive = String(button.getAttribute('data-doc-tab') || '') === targetID;
        button.classList.toggle('active', isActive);
        button.setAttribute('aria-selected', isActive ? 'true' : 'false');
        button.setAttribute('tabindex', isActive ? '0' : '-1');
        if (isActive) button.dataset.docTabIndex = String(idx);
      });
      panels.forEach(function(panel) {
        const isActive = panel.id === targetID;
        panel.classList.toggle('active', isActive);
        panel.hidden = !isActive;
      });
    };
    buttons.forEach(function(button) {
      button.addEventListener('click', function() {
        activate(String(button.getAttribute('data-doc-tab') || ''));
      });
    });
    activate(String(buttons[0].getAttribute('data-doc-tab') || ''));
  });
}

function docsUpdateURL(key) {
  docsCurrentKey = key;
  window.history.replaceState({}, '', docsPageURL(key));
  docsRenderCatalog();
}

async function loadDocsCatalog() {
  const res = await fetch('/api/docs/catalog', { headers: docsAuthHeaders() });
  const text = await res.text();
  if (!res.ok) throw new Error(text || ('HTTP ' + res.status));
  const data = text ? JSON.parse(text) : {};
  docsCatalog = Array.isArray(data.docs) ? data.docs : [];
  docsRenderCatalog();
}

async function loadDoc(key, pushState) {
  docsArticleKickerEl.textContent = 'Current doc';
  docsArticleDescriptionEl.textContent = 'Fetching the selected document.';
  docsArticleSourceEl.textContent = 'loading...';
  docsArticleBodyEl.innerHTML = '<div class="loading">Loading docs...</div>';
  docsCurrentHeadings = [];
  docsRenderOutline();
  try {
    const res = await fetch('/api/docs/view?name=' + encodeURIComponent(String(key || '').trim()), { headers: docsAuthHeaders() });
    const text = await res.text();
    if (!res.ok) throw new Error(text || ('HTTP ' + res.status));
    const data = text ? JSON.parse(text) : {};
    const headings = Array.isArray(data.headings) ? data.headings : [];
    docsCurrentHeadings = headings;
    docsArticleKickerEl.textContent = String(data.label || data.name || 'Current doc');
    docsArticleDescriptionEl.textContent = String(data.description || '');
    docsArticleSourceEl.textContent = String(data.name || '');
    docsArticleBodyEl.innerHTML = String(data.content_html || '<div class="article-empty">Document is empty.</div>');
    initDocTabs();
    docsRenderOutline();
    document.title = docsArticleKickerEl.textContent + ' · Knit Docs';
    if (pushState !== false) docsUpdateURL(String(data.key || key || ''));
  } catch (err) {
    docsCurrentHeadings = [];
    docsRenderOutline();
    docsArticleKickerEl.textContent = 'Docs unavailable';
    docsArticleDescriptionEl.textContent = 'The requested local doc could not be opened.';
    docsArticleSourceEl.textContent = 'Unavailable';
    docsArticleBodyEl.innerHTML = '<div class="article-empty">' + escapeHTML(String(err?.message || err || 'Could not load document.')) + '</div>';
  }
}

function docsHighlightActiveOutline() {
  if (!docsOutlineEl) return;
  const headings = Array.from(docsArticleBodyEl.querySelectorAll('h2[id], h3[id], h4[id]'));
  if (headings.length === 0) return;
  let activeID = headings[0].id;
  for (const heading of headings) {
    const rect = heading.getBoundingClientRect();
    if (rect.top <= 140) activeID = heading.id;
  }
  Array.from(docsOutlineEl.querySelectorAll('a')).forEach(function(node) {
    const href = String(node.getAttribute('href') || '');
    node.classList.toggle('active', href === ('#' + activeID));
  });
}

async function initDocsBrowser() {
  docsApplyTheme(docsSavedTheme());
  await loadDocsCatalog();
  const requested = String(docsURLState.searchParams.get('name') || '').trim();
  const initial = requested || String((docsCatalog[0] && docsCatalog[0].key) || '');
  if (initial) {
    await loadDoc(initial, true);
    docsHighlightActiveOutline();
  } else {
    docsArticleKickerEl.textContent = 'Docs unavailable';
    docsArticleDescriptionEl.textContent = 'This runtime could not locate the local docs directory.';
    docsArticleSourceEl.textContent = 'Unavailable';
    docsArticleBodyEl.innerHTML = '<div class="article-empty">No docs are available to render.</div>';
  }
}

if (docsOpenCurrentTabBtnEl) {
  docsOpenCurrentTabBtnEl.addEventListener('click', function() {
    const tab = window.open(docsPageURL(docsCurrentKey), '_blank', 'noopener');
    try { if (tab) tab.focus(); } catch (_) {}
  });
}
if (docsThemeToggleBtnEl) {
  docsThemeToggleBtnEl.addEventListener('click', docsToggleTheme);
}
window.addEventListener('scroll', docsHighlightActiveOutline, { passive: true });
initDocsBrowser().catch(function(err) {
  docsArticleKickerEl.textContent = 'Docs unavailable';
  docsArticleDescriptionEl.textContent = 'The docs browser failed to initialize.';
  docsArticleSourceEl.textContent = 'Unavailable';
  docsArticleBodyEl.innerHTML = '<div class="article-empty">' + escapeHTML(String(err?.message || err || 'Could not initialize docs browser.')) + '</div>';
});
  </script>
</body>
</html>
`
