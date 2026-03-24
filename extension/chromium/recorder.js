const sessionStateEl = document.getElementById("sessionState");
const themeToggleBtnEl = document.getElementById("themeToggleBtn");
const headerHelperEl = document.getElementById("headerHelper");
const textNoteWrapEl = document.getElementById("textNoteWrap");
const noteTextEl = document.getElementById("noteText");
const toggleTextBtnEl = document.getElementById("toggleTextBtn");
const snapshotBtnEl = document.getElementById("snapshotBtn");
const audioBtnEl = document.getElementById("audioBtn");
const videoBtnEl = document.getElementById("videoBtn");
const previewBtnEl = document.getElementById("previewBtn");
const submitBtnEl = document.getElementById("submitBtn");
const stopBtnEl = document.getElementById("stopBtn");
const captureStateEl = document.getElementById("captureState");
const videoGuidanceEl = document.getElementById("videoGuidance");
const activityStateEl = document.getElementById("activityState");
const activityStateTextEl = document.getElementById("activityStateText");
const submitNoticeEl = document.getElementById("submitNotice");
const queueIndicatorEl = document.getElementById("queueIndicator");
const statusEl = document.getElementById("status");
const previewEl = document.getElementById("preview");
const previewVideoEl = document.getElementById("previewVideo");

const ICONS = {
  keyboard: `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 18H20"></path><path d="M7 14V6"></path><path d="M12 14V6"></path><path d="M17 14V6"></path><path d="M5 6H19"></path></svg>`,
  pencil: `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 20H21"></path><path d="M16.5 3.5A2.121 2.121 0 0 1 19.5 6.5L7 19L3 20L4 16L16.5 3.5Z"></path></svg>`,
  snapshot: `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 8H8L10 6H14L16 8H20V18H4Z"></path><circle cx="12" cy="13" r="3"></circle></svg>`,
  mic: `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4C10.343 4 9 5.343 9 7V12C9 13.657 10.343 15 12 15C13.657 15 15 13.657 15 12V7C15 5.343 13.657 4 12 4Z"></path><path d="M7 11V12C7 14.761 9.239 17 12 17C14.761 17 17 14.761 17 12V11"></path><path d="M12 17V20"></path><path d="M9 20H15"></path></svg>`,
  video: `<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="6" width="13" height="12" rx="2"></rect><path d="M16 10L21 7V17L16 14"></path></svg>`,
  trash: `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6H21"></path><path d="M8 6V4H16V6"></path><path d="M19 6L18 20H6L5 6"></path><path d="M10 10V16"></path><path d="M14 10V16"></path></svg>`,
  stop: `<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="7" y="7" width="10" height="10"></rect></svg>`,
  close: `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 6L6 18"></path><path d="M6 6L18 18"></path></svg>`
};

const PENDING_SNAPSHOT_KEY = "pendingSnapshotState";
const THEME_STORAGE_KEY = "sidePanelTheme";
const ACTIVE_SESSION_REFRESH_MS = 3000;
const IDLE_SESSION_REFRESH_MS = 5000;
const PREVIEW_EMPTY_HTML = `<div class="preview-empty">Preview updates here automatically as you capture notes, snapshots, audio, and video.</div>`;
const PREVIEW_LOADING_HTML = `<div class="preview-loading"><span class="preview-spinner" aria-hidden="true"></span><span>Loading preview…</span></div>`;

let config = {
  daemonBaseURL: "http://127.0.0.1:7777",
  daemonToken: ""
};
let currentSessionPayload = null;
let textEditorOpen = false;
let requestInFlight = false;
let recordingKind = "";
let pendingSnapshotBlob = null;
let pendingSnapshotContext = null;
let audioNoteRecorder = null;
let audioNoteStream = null;
let audioNoteChunks = [];
let audioNoteStopPromise = null;
let videoNoteAudioRecorder = null;
let videoNoteClipRecorder = null;
let videoNoteMicStream = null;
let videoNoteDisplayStream = null;
let videoNoteAudioChunks = [];
let videoNoteClipChunks = [];
let videoNoteAudioStopPromise = null;
let videoNoteClipStopPromise = null;
let videoNoteStartedAt = 0;
let videoNoteFinalizing = false;
let latestPreviewPayload = null;
let pendingSnapshotPreviewURL = "";
let requestActivityMessage = "";
let currentTheme = "light";
let previewDeliveryOptions = { omitVideoEventIDs: [] };
let watchedSubmitAttemptID = "";
let latestSubmitOutcomeNoticeID = "";
let submitAttemptWatchTimer = 0;
let sessionRefreshTimer = 0;
let videoGuidanceTimer = 0;

async function callBackground(message) {
  return chrome.runtime.sendMessage(message);
}

async function storageGet(keys) {
  return await chrome.storage.local.get(keys);
}

async function storageSet(payload) {
  await chrome.storage.local.set(payload);
}

async function storageRemove(keys) {
  await chrome.storage.local.remove(keys);
}

function normalizeTheme(theme) {
  return String(theme || "").trim().toLowerCase() === "dark" ? "dark" : "light";
}

function applyTheme(theme) {
  currentTheme = normalizeTheme(theme);
  document.documentElement.setAttribute("data-theme", currentTheme);
  if (!themeToggleBtnEl) return;
  const nextTheme = currentTheme === "dark" ? "light" : "dark";
  themeToggleBtnEl.textContent = currentTheme === "dark" ? "☀" : "☾";
  themeToggleBtnEl.title = nextTheme === "dark" ? "Switch to dark theme" : "Switch to light theme";
  themeToggleBtnEl.setAttribute("aria-label", themeToggleBtnEl.title);
}

async function loadTheme() {
  const stored = await storageGet([THEME_STORAGE_KEY]);
  applyTheme(stored?.[THEME_STORAGE_KEY] || "light");
}

async function toggleTheme() {
  const nextTheme = currentTheme === "dark" ? "light" : "dark";
  applyTheme(nextTheme);
  await storageSet({ [THEME_STORAGE_KEY]: nextTheme });
}

async function loadConfig() {
  const stored = await callBackground({ type: "knit:get-config" });
  config = {
    daemonBaseURL: String(stored?.daemonBaseURL || "http://127.0.0.1:7777").trim(),
    daemonToken: String(stored?.daemonToken || "").trim()
  };
}

async function blobToDataURL(blob) {
  return await new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error || new Error("Could not read blob."));
    reader.onload = () => resolve(String(reader.result || ""));
    reader.readAsDataURL(blob);
  });
}

async function dataURLToBlob(dataURL) {
  const res = await fetch(String(dataURL || ""));
  return await res.blob();
}

function setStatus(message, isError = false) {
  const text = String(message || "").trim();
  statusEl.textContent = text;
  statusEl.classList.toggle("hidden", !text);
  statusEl.classList.toggle("error", !!text && !!isError);
}

function setSubmitNotice(message, isError = false) {
  const text = String(message || "").trim();
  submitNoticeEl.textContent = text;
  submitNoticeEl.classList.toggle("hidden", !text);
  submitNoticeEl.classList.toggle("error", !!text && !!isError);
}

function buildMainUIAttemptURL(attemptID) {
  const base = config.daemonBaseURL.replace(/\/+$/, "") + "/";
  const url = new URL(base);
  const id = String(attemptID || "").trim();
  if (id) url.searchParams.set("attempt_id", id);
  return url.toString();
}

async function openMainUIForAttempt(attemptID) {
  const url = buildMainUIAttemptURL(attemptID);
  if (chrome?.tabs?.create) {
    await chrome.tabs.create({ url });
    return;
  }
  window.open(url, "_blank", "noopener");
}

function isTerminalSubmitStatus(status) {
  const value = String(status || "").trim();
  return value === "submitted" || value === "failed" || value === "canceled";
}

function submitAttemptOutcomeNoticeMessage(attempt) {
  const title = String(attempt?.outcome_title || "").trim();
  const message = String(attempt?.outcome_message || "").trim();
  if (title && message) return `${title}: ${message}`;
  return message || title;
}

function ensureSubmitAttemptWatchTimer() {
  if (submitAttemptWatchTimer || !watchedSubmitAttemptID) return;
  submitAttemptWatchTimer = window.setInterval(() => {
    if (!watchedSubmitAttemptID) {
      window.clearInterval(submitAttemptWatchTimer);
      submitAttemptWatchTimer = 0;
      return;
    }
    refreshSession(true).catch(() => {});
  }, 3000);
}

function stopSubmitAttemptWatchTimer() {
  if (!submitAttemptWatchTimer) return;
  window.clearInterval(submitAttemptWatchTimer);
  submitAttemptWatchTimer = 0;
}

function hasActiveSubmitAttempt(data) {
  const attempts = Array.isArray(data?.submit_attempts) ? data.submit_attempts : [];
  return attempts.some((attempt) => {
    const status = String(attempt?.status || "").trim();
    return status === "queued" || status === "in_progress" || status === "retry_wait" || status === "deferred_offline";
  });
}

function stopSessionRefreshTimer() {
  if (!sessionRefreshTimer) return;
  window.clearTimeout(sessionRefreshTimer);
  sessionRefreshTimer = 0;
}

function scheduleSessionRefresh(data, delayOverride = 0) {
  stopSessionRefreshTimer();
  if (!config.daemonToken) return;
  const delay = delayOverride > 0
    ? delayOverride
    : (hasActiveSubmitAttempt(data) ? ACTIVE_SESSION_REFRESH_MS : IDLE_SESSION_REFRESH_MS);
  sessionRefreshTimer = window.setTimeout(() => {
    sessionRefreshTimer = 0;
    refreshSession(true).catch(() => {});
  }, delay);
}

function maybeShowSubmitAttemptOutcomeNotice(data) {
  const attempts = Array.isArray(data?.submit_attempts) ? data.submit_attempts : [];
  const watched = watchedSubmitAttemptID
    ? attempts.find((attempt) => String(attempt?.attempt_id || "").trim() === watchedSubmitAttemptID) || null
    : null;
  const latestTerminal = watched || attempts.find((attempt) => isTerminalSubmitStatus(attempt?.status)) || null;
  if (!latestTerminal) return;
  const attemptID = String(latestTerminal?.attempt_id || "").trim();
  if (!attemptID || latestSubmitOutcomeNoticeID === attemptID) return;
  if (watchedSubmitAttemptID && attemptID !== watchedSubmitAttemptID) return;
  if (!isTerminalSubmitStatus(latestTerminal?.status)) return;
  const outcome = submitAttemptOutcomeNoticeMessage(latestTerminal);
  if (outcome) {
    setSubmitNotice(outcome, true);
    latestSubmitOutcomeNoticeID = attemptID;
  }
  if (watchedSubmitAttemptID === attemptID) {
    watchedSubmitAttemptID = "";
    stopSubmitAttemptWatchTimer();
  }
}

function renderActivityState() {
  const text = String(requestActivityMessage || "").trim();
  if (!activityStateEl || !activityStateTextEl) return;
  activityStateTextEl.textContent = text || "Working…";
  activityStateEl.classList.toggle("hidden", !text);
}

function renderQueueIndicator() {
  if (!queueIndicatorEl) return;
  const attempts = Array.isArray(currentSessionPayload?.submit_attempts) ? currentSessionPayload.submit_attempts : [];
  const queue = currentSessionPayload?.submit_queue || {};
  const running = Number(queue?.running || 0);
  const queued = Number(queue?.queued || 0);
  const visible = attempts
    .filter((attempt) => {
      const status = String(attempt?.status || "").trim();
      return status === "queued" || status === "in_progress" || status === "retry_wait" || status === "deferred_offline" || status === "submitted";
    })
    .slice(0, 5);
  if (running <= 0 && queued <= 0 && !visible.length) {
    queueIndicatorEl.textContent = "";
    queueIndicatorEl.classList.add("hidden");
    return;
  }
  const summary = `${running} running, ${queued} queued`;
  const rows = visible.length
    ? `<table class="queue-table"><tbody>${visible.map((attempt) => {
        const attemptID = String(attempt?.attempt_id || "").trim();
        const status = queueAttemptStatusLabel(attempt);
        const label = queueAttemptTitle(attempt);
        return `<tr><td><button type="button" class="queue-link" data-attempt-link="${escapeHTML(attemptID)}" title="Open ${escapeHTML(label)} in the main UI">${escapeHTML(label)}</button></td><td>${escapeHTML(status)}</td></tr>`;
      }).join("")}</tbody></table>`
    : "";
  queueIndicatorEl.innerHTML = `<div>Queue: ${escapeHTML(summary)}</div>${rows}`;
  queueIndicatorEl.classList.remove("hidden");
}

function queueAttemptStatusLabel(attempt) {
  const status = String(attempt?.status || "").trim();
  switch (status) {
    case "submitted":
      return "completed";
    case "in_progress":
      return "running";
    case "retry_wait":
      return "retrying";
    case "deferred_offline":
      return "waiting";
    default:
      return status || "recent";
  }
}

function queueAttemptTitle(attempt) {
  const preview = String(attempt?.request_preview || "").trim();
  if (!preview) return "Recent request";
  const compact = preview.replace(/\s+/g, " ").trim();
  if (!compact) return "Recent request";
  const words = compact.split(" ").slice(0, 4).join(" ");
  return words.length < compact.length ? `${words}…` : words;
}

async function consumeSubmitNotice() {
  const data = await callBackground({ type: "knit:consume-submit-notice" });
  setSubmitNotice(String(data?.notice?.message || "").trim());
}

function escapeHTML(value) {
  return String(value || "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function formatMediaSize(bytes) {
  const value = Number(bytes || 0);
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size >= 10 || unitIndex === 0 ? Math.round(size) : size.toFixed(1)} ${units[unitIndex]}`;
}

function authHeaders(isMutation) {
  const headers = {
    Authorization: "Bearer " + config.daemonToken
  };
  if (isMutation) {
    headers["X-Knit-Nonce"] = (self.crypto && self.crypto.randomUUID) ? self.crypto.randomUUID() : String(Date.now());
    headers["X-Knit-Timestamp"] = String(Date.now());
  }
  return headers;
}

async function request(path, options = {}) {
  const url = config.daemonBaseURL.replace(/\/+$/, "") + path;
  const res = await fetch(url, options);
  const text = await res.text();
  if (!res.ok) {
    throw new Error(text || ("HTTP " + res.status));
  }
  return text ? JSON.parse(text) : {};
}

async function postJSON(path, payload) {
  return request(path, {
    method: "POST",
    headers: { ...authHeaders(true), "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
}

async function postForm(path, form) {
  return request(path, {
    method: "POST",
    headers: authHeaders(true),
    body: form
  });
}

function setRequestInFlight(active) {
  requestInFlight = !!active;
  if (!requestInFlight) {
    requestActivityMessage = "";
  }
  renderActivityState();
  renderComposerControls();
}

function setRequestActivity(message) {
  requestActivityMessage = String(message || "").trim();
  renderActivityState();
}

function setIconButton(btn, iconMarkup, title, options = {}) {
  if (!btn) return;
  btn.innerHTML = iconMarkup;
  setButtonTooltip(btn, title);
  btn.disabled = !!options.disabled;
  btn.classList.toggle("active", !!options.active);
}

function setButtonTooltip(btn, tooltip) {
  if (!btn) return;
  const text = String(tooltip || "").trim();
  btn.setAttribute("aria-label", text);
  btn.removeAttribute("title");
  const wrap = btn.closest(".tooltip-wrap");
  if (wrap) {
    wrap.dataset.tooltip = text;
  }
}

function typedNoteValue() {
  return String(noteTextEl?.value || "").trim();
}

function hasPendingSnapshot() {
  return !!pendingSnapshotBlob;
}

function queuedSnapshotFilename() {
  return "extension-frame.png";
}

function revokePendingSnapshotPreviewURL() {
  if (!pendingSnapshotPreviewURL) return;
  URL.revokeObjectURL(pendingSnapshotPreviewURL);
  pendingSnapshotPreviewURL = "";
}

function queuedSnapshotPreviewURL() {
  if (!pendingSnapshotBlob) {
    revokePendingSnapshotPreviewURL();
    return "";
  }
  if (!pendingSnapshotPreviewURL) {
    pendingSnapshotPreviewURL = URL.createObjectURL(pendingSnapshotBlob);
  }
  return pendingSnapshotPreviewURL;
}

function previewMediaHTML(src, caption, alt) {
  if (!src) return "";
  return `<figure class="preview-media"><figcaption>${escapeHTML(caption)}</figcaption><img src="${escapeHTML(src)}" alt="${escapeHTML(alt)}" /></figure>`;
}

function previewVideoHTML(src, caption) {
  if (!src) return "";
  return `<figure class="preview-media"><figcaption>${escapeHTML(caption)}</figcaption><div class="preview-media-frame"><div class="preview-loading centered" data-preview-video-loading><span class="preview-spinner" aria-hidden="true"></span><span>Loading video preview…</span></div><video controls playsinline preload="metadata" src="${escapeHTML(src)}" data-preview-video></video></div></figure>`;
}

function setPreviewLoading(message = "Loading preview…") {
  previewEl.innerHTML = `<div class="preview-loading"><span class="preview-spinner" aria-hidden="true"></span><span>${escapeHTML(message)}</span></div>`;
}

function hidePreviewVideoSpinner(videoEl) {
  if (!videoEl) return;
  const frame = videoEl.closest(".preview-media-frame");
  const loading = frame?.querySelector("[data-preview-video-loading]");
  if (loading) {
    loading.classList.add("hidden");
  }
}

function showPreviewVideoError(videoEl) {
  if (!videoEl) return;
  const frame = videoEl.closest(".preview-media-frame");
  const loading = frame?.querySelector("[data-preview-video-loading]");
  if (loading) {
    loading.classList.remove("hidden");
    loading.innerHTML = `<span>Video preview could not be loaded.</span>`;
  }
}

function hydratePreviewMediaLoadState() {
  previewEl.querySelectorAll("[data-preview-video]").forEach((videoEl) => {
    if (videoEl.readyState >= 1) {
      hidePreviewVideoSpinner(videoEl);
    }
    videoEl.addEventListener("loadedmetadata", () => hidePreviewVideoSpinner(videoEl), { once: true });
    videoEl.addEventListener("canplay", () => hidePreviewVideoSpinner(videoEl), { once: true });
    videoEl.addEventListener("error", () => showPreviewVideoError(videoEl), { once: true });
  });
}

function previewNoteNeedsVideoDecision(note) {
  const status = String(note?.video_transmission_status || "").trim();
  const size = Number(note?.video_size_bytes || 0);
  const limit = Number(note?.video_send_limit_bytes || 0);
  return !!note?.has_video && (status === "omitted_due_to_limit" || (limit > 0 && size > limit));
}

function previewVideoEventOmitted(eventID) {
  const id = String(eventID || "").trim();
  return !!id && Array.isArray(previewDeliveryOptions.omitVideoEventIDs) && previewDeliveryOptions.omitVideoEventIDs.includes(id);
}

function setPreviewVideoEventOmitted(eventID, omitted) {
  const id = String(eventID || "").trim();
  const current = Array.isArray(previewDeliveryOptions.omitVideoEventIDs) ? previewDeliveryOptions.omitVideoEventIDs.slice() : [];
  const next = current.filter((item) => item !== id);
  if (omitted && id) {
    next.push(id);
  }
  previewDeliveryOptions.omitVideoEventIDs = next;
}

function resetPreviewDeliveryOptions() {
  previewDeliveryOptions = { omitVideoEventIDs: [] };
}

function syncPreviewDeliveryOptions(preview) {
  const validIDs = new Set((Array.isArray(preview?.notes) ? preview.notes : []).map((note) => String(note?.event_id || "").trim()).filter(Boolean));
  previewDeliveryOptions.omitVideoEventIDs = (Array.isArray(previewDeliveryOptions.omitVideoEventIDs) ? previewDeliveryOptions.omitVideoEventIDs : []).filter((id) => validIDs.has(id));
}

function renderOversizedVideoWarningActions(preview) {
  const notes = (Array.isArray(preview?.notes) ? preview.notes : []).filter((note) => previewNoteNeedsVideoDecision(note));
  if (!notes.length) return "";
  const items = notes.map((note) => {
    const eventID = String(note?.event_id || "").trim();
    const useSnapshotLabel = previewVideoEventOmitted(eventID)
      ? "Send clip again"
      : (note?.has_screenshot ? "Use snapshot instead" : "Omit clip for this request");
    const transmissionNote = String(note?.video_transmission_note || "This clip is too large to send with the current inline media setting.").trim();
    return `<section class="preview-note"><div class="preview-note-header"><div class="preview-note-title">${escapeHTML(eventID || "Queued request")}</div></div><div class="preview-note-text">${escapeHTML(transmissionNote)}</div><div class="preview-note-meta">${escapeHTML(formatMediaSize(note?.video_size_bytes || 0))} over ${escapeHTML(formatMediaSize(note?.video_send_limit_bytes || 0))}</div><div class="mini-toolbar"><button type="button" data-preview-action="toggle-video-omit" data-event-id="${escapeHTML(eventID)}">${escapeHTML(useSnapshotLabel)}</button><button type="button" class="danger" data-preview-action="delete" data-event-id="${escapeHTML(eventID)}">Delete request</button></div></section>`;
  }).join("");
  return `<div class="preview-warning-card"><strong>Large clip needs a decision</strong><div class="preview-note-text">Choose how to handle the affected request before you submit.</div>${items}</div>`;
}

function renderPreviewSurface() {
  const preview = latestPreviewPayload || null;
  const notes = Array.isArray(preview?.notes) ? preview.notes : [];
  if (!preview && !hasPendingSnapshot()) {
    previewEl.innerHTML = PREVIEW_EMPTY_HTML;
    return;
  }
  if (!preview && hasPendingSnapshot()) {
    previewEl.innerHTML = `<div class="preview-summary"><strong>Queued snapshot ready</strong></div>${previewMediaHTML(queuedSnapshotPreviewURL(), "Queued snapshot", "Queued snapshot for the next note")}`;
    return;
  }
  syncPreviewDeliveryOptions(preview);
  const summary = escapeHTML(preview?.summary || "Ready to send");
  const oversizedVideoBlock = renderOversizedVideoWarningActions(preview);
  const noteCards = notes.length ? notes.map((note, index) => {
    const eventID = String(note?.event_id || "").trim();
    const dom = note?.dom_summary ? `<div class="preview-note-meta">Target: ${escapeHTML(note.dom_summary)}</div>` : "";
    const media = [
      note?.has_screenshot ? "Snapshot" : "",
      note?.has_audio ? "Audio" : "",
      note?.has_video ? "Video" : ""
    ].filter(Boolean).join(", ");
    const mediaLine = media ? `<div class="preview-note-meta">Media: ${escapeHTML(media)}</div>` : "";
    const screenshot = note?.screenshot_data_url
      ? previewMediaHTML(note.screenshot_data_url, "Snapshot", "Captured snapshot for request preview")
      : "";
    const video = note?.video_data_url
      ? previewVideoHTML(note.video_data_url, "Clip")
      : "";
    const transmissionNote = previewNoteNeedsVideoDecision(note)
      ? `<div class="preview-note-meta">${escapeHTML(String(note?.video_transmission_note || "This clip is too large to send inline.").trim())}</div>`
      : "";
    const deleteAction = eventID
      ? `<button type="button" class="preview-note-action" data-preview-action="delete" data-event-id="${escapeHTML(eventID)}" title="Remove queued request" aria-label="Remove queued request">${ICONS.trash}</button>`
      : "";
    return `<section class="preview-note"><div class="preview-note-header"><div class="preview-note-title">Request ${index + 1}</div>${deleteAction}</div><div class="preview-note-text">${escapeHTML(note?.text || "")}</div>${dom}${mediaLine}${transmissionNote}${screenshot}${video}</section>`;
  }).join("") : `<div class="preview-empty">No notes captured yet. Add a note or snapshot first.</div>`;
  const queuedSnapshotFallback = !notes.some((note) => !!note?.screenshot_data_url) && hasPendingSnapshot()
    ? previewMediaHTML(queuedSnapshotPreviewURL(), "Queued snapshot", "Queued snapshot for the next note")
    : "";
  previewEl.innerHTML = `<div class="preview-summary"><strong>${summary}</strong></div>${oversizedVideoBlock}${queuedSnapshotFallback}${noteCards}`;
  hydratePreviewMediaLoadState();
}

function renderCaptureState() {
  if (!captureStateEl) return;
  if (hasPendingSnapshot()) {
    captureStateEl.textContent = "Snapshot queued. Add a typed note, voice note, or tab recording next.";
    captureStateEl.classList.add("ready");
    captureStateEl.classList.remove("hidden");
    return;
  }
  captureStateEl.textContent = "";
  captureStateEl.classList.remove("ready");
  captureStateEl.classList.add("hidden");
}

function formatElapsedClock(ms) {
  const totalSeconds = Math.max(0, Math.round(Number(ms || 0) / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

function stopVideoGuidanceTimer() {
  if (!videoGuidanceTimer) return;
  window.clearInterval(videoGuidanceTimer);
  videoGuidanceTimer = 0;
}

function ensureVideoGuidanceTimer() {
  if (videoGuidanceTimer || recordingKind !== "video") return;
  videoGuidanceTimer = window.setInterval(() => {
    renderVideoGuidance();
  }, 1000);
}

function renderVideoGuidance() {
  if (!videoGuidanceEl) return;
  if (recordingKind !== "video" && !videoNoteFinalizing) {
    videoGuidanceEl.textContent = "";
    videoGuidanceEl.classList.add("hidden");
    videoGuidanceEl.classList.remove("active");
    stopVideoGuidanceTimer();
    return;
  }
  let text = "";
  let active = false;
  if (recordingKind === "video") {
    text = `Recording video note: ${formatElapsedClock(Date.now() - videoNoteStartedAt)}. Aim for 5-20 seconds so the clip is more likely to fit send limits.`;
    active = true;
    ensureVideoGuidanceTimer();
  } else if (videoNoteFinalizing) {
    text = "Finalizing video note…";
    active = true;
    stopVideoGuidanceTimer();
  } else {
    stopVideoGuidanceTimer();
  }
  videoGuidanceEl.textContent = text;
  videoGuidanceEl.classList.toggle("hidden", !text);
  videoGuidanceEl.classList.toggle("active", active);
}

function renderHeaderHelper() {
  if (!headerHelperEl) return;
  const paired = !!config.daemonToken;
  headerHelperEl.textContent = paired
    ? ""
    : "Pair this browser from the popup first to connect it to the main Knit UI.";
  headerHelperEl.classList.toggle("hidden", paired);
}

function renderSessionState() {
  const hasSession = !!currentSessionPayload?.session?.id;
  sessionStateEl.textContent = hasSession ? "Session active" : "Session idle";
  sessionStateEl.classList.toggle("idle", !hasSession);
}

function renderTextEditorState() {
  textNoteWrapEl.classList.toggle("hidden", !textEditorOpen);
  const hasSession = !!currentSessionPayload?.session?.id;
  const saveReady = textEditorOpen && !!typedNoteValue();
  setIconButton(
    toggleTextBtnEl,
    textEditorOpen ? ICONS.keyboard : ICONS.pencil,
    !hasSession
      ? "Start a review session from the main UI first."
      : saveReady
      ? (hasPendingSnapshot() ? "Save typed note with the queued snapshot." : "Save typed note.")
      : (textEditorOpen ? "Hide typed note field." : "Type note. Show or hide the typed note field."),
    { disabled: !hasSession || requestInFlight || !!recordingKind || videoNoteFinalizing, active: hasSession && (textEditorOpen || saveReady) }
  );
}

function renderComposerControls() {
  const hasSession = !!currentSessionPayload?.session?.id;
  const busy = requestInFlight || videoNoteFinalizing;
  setIconButton(
    snapshotBtnEl,
    ICONS.snapshot,
    !hasSession
      ? "Start a review session from the main UI first."
      : hasPendingSnapshot()
      ? "Snapshot queued. Capture again to replace it before you submit the next note."
      : "Capture snapshot. Queue the current tab for your next typed or voice note.",
    { disabled: !hasSession || busy || !!recordingKind, active: hasPendingSnapshot() }
  );
  setIconButton(
    audioBtnEl,
    recordingKind === "audio" ? ICONS.stop : ICONS.mic,
    !hasSession
      ? "Start a review session from the main UI first."
      : recordingKind === "audio"
      ? "Stop recording audio note."
      : (hasPendingSnapshot() ? "Record a voice note. The queued snapshot will be attached." : "Record a voice note for the current page."),
    { disabled: !hasSession || busy || recordingKind === "video", active: recordingKind === "audio" }
  );
  setIconButton(
    videoBtnEl,
    recordingKind === "video" ? ICONS.stop : ICONS.video,
    !hasSession
      ? "Start a review session from the main UI first."
      : recordingKind === "video"
      ? "Stop recording the current tab."
      : (hasPendingSnapshot() ? "Record the current tab with voice. The queued snapshot will be attached. Aim for 5-20 seconds." : "Record the current tab with voice. Aim for 5-20 seconds."),
    { disabled: !hasSession || busy || recordingKind === "audio", active: recordingKind === "video" }
  );
  setIconButton(
    stopBtnEl,
    ICONS.close,
    "Clear session",
    { disabled: busy || !!recordingKind || !hasSession }
  );
  previewBtnEl.disabled = busy || !!recordingKind || !hasSession;
  submitBtnEl.disabled = busy || !!recordingKind || !hasSession;
  setButtonTooltip(
    previewBtnEl,
    !hasSession
      ? "Start a review session from the main UI first."
      : (busy || !!recordingKind || videoNoteFinalizing)
        ? "Finish the current recording or request first."
        : "Preview queued requests before submitting."
  );
  setButtonTooltip(
    submitBtnEl,
    !hasSession
      ? "Start a review session from the main UI first."
      : (busy || !!recordingKind || videoNoteFinalizing)
        ? "Finish the current recording or request first."
        : "Submit the current package to the daemon queue."
  );
  renderCaptureState();
  renderVideoGuidance();
  renderHeaderHelper();
  renderSessionState();
  renderTextEditorState();
}

function renderSession(data) {
  currentSessionPayload = data || null;
  renderComposerControls();
  renderQueueIndicator();
}

function clearPendingSnapshot() {
  pendingSnapshotBlob = null;
  pendingSnapshotContext = null;
  revokePendingSnapshotPreviewURL();
  storageRemove([PENDING_SNAPSHOT_KEY]).catch(() => {});
  renderCaptureState();
  renderPreviewSurface();
}

async function savePendingSnapshotState(blob, context) {
  pendingSnapshotBlob = blob;
  pendingSnapshotContext = context;
  revokePendingSnapshotPreviewURL();
  const dataURL = await blobToDataURL(blob);
  await storageSet({
    [PENDING_SNAPSHOT_KEY]: {
      dataURL,
      context
    }
  });
  renderCaptureState();
  renderPreviewSurface();
}

async function restorePendingSnapshotState() {
  const items = await storageGet([PENDING_SNAPSHOT_KEY]);
  const saved = items?.[PENDING_SNAPSHOT_KEY] || null;
  if (!saved?.dataURL) {
    pendingSnapshotBlob = null;
    pendingSnapshotContext = null;
    renderCaptureState();
    renderPreviewSurface();
    return;
  }
  pendingSnapshotBlob = await dataURLToBlob(saved.dataURL);
  pendingSnapshotContext = saved.context || null;
  revokePendingSnapshotPreviewURL();
  renderCaptureState();
  renderPreviewSurface();
}

async function refreshSession(preserveStatus = false) {
  if (!config.daemonToken) {
    stopSessionRefreshTimer();
    currentSessionPayload = null;
    latestPreviewPayload = null;
    resetPreviewDeliveryOptions();
    clearPendingSnapshot();
    renderComposerControls();
    if (!preserveStatus) {
      setStatus("Pair the extension from the popup first.", true);
    }
    renderPreviewSurface();
    return;
  }
  try {
    const data = await request("/api/extension/session", { headers: authHeaders(false) });
    renderSession(data);
    maybeShowSubmitAttemptOutcomeNotice(data);
    scheduleSessionRefresh(data);
    if (!data?.session?.id) {
      resetPreviewDeliveryOptions();
      clearPendingSnapshot();
    }
    if (!preserveStatus) {
      setStatus(data?.session?.id ? "" : "No active review session. Start one from the main Knit UI first.");
    }
  } catch (err) {
    scheduleSessionRefresh(null, IDLE_SESSION_REFRESH_MS);
    currentSessionPayload = null;
    resetPreviewDeliveryOptions();
    clearPendingSnapshot();
    renderComposerControls();
    renderQueueIndicator();
    if (!preserveStatus) {
      setStatus("Daemon connection failed: " + err.message, true);
    }
  }
}

async function refreshPreviewAuto(options = {}) {
  const preserveStatus = !!options.preserveStatus;
  try {
    await preview(true);
  } catch (err) {
    if (!preserveStatus) {
      setStatus("Preview could not be refreshed: " + err.message, true);
    }
  }
}

async function activeTab() {
  const tabs = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
  if (!tabs.length) {
    throw new Error("No active browser tab found.");
  }
  return tabs[0];
}

async function collectPageContext(tabId) {
  const [{ result }] = await chrome.scripting.executeScript({
    target: { tabId },
    func: () => {
      const active = document.activeElement;
      const attrs = active && active.attributes ? Object.fromEntries(Array.from(active.attributes).map((item) => [item.name, item.value]).slice(0, 12)) : {};
      const selector = active && active.id ? `#${active.id}` : (active ? active.tagName.toLowerCase() : "");
      return {
        title: document.title || "",
        url: location.href,
        route: location.pathname || "",
        selection: String(window.getSelection ? window.getSelection() : "").trim(),
        dom: active ? {
          tag: active.tagName.toLowerCase(),
          id: active.id || "",
          test_id: active.getAttribute("data-testid") || active.getAttribute("data-test-id") || "",
          label: active.getAttribute("aria-label") || active.getAttribute("name") || "",
          role: active.getAttribute("role") || "",
          selector,
          text_preview: String(active.textContent || "").trim().slice(0, 160),
          attributes: attrs
        } : null
      };
    }
  });
  return result || {};
}

async function currentSessionBundle() {
  const current = await request("/api/extension/session", { headers: authHeaders(false) });
  renderSession(current);
  if (!current?.session?.id) {
    throw new Error("Start a session first.");
  }
  const tab = await activeTab();
  const context = await collectPageContext(tab.id);
  return { current, tab, context };
}

async function postCurrentBrowserPointer(sessionID, tab, context) {
  await postJSON("/api/companion/pointer", {
    session_id: sessionID,
    x: 0,
    y: 0,
    event_type: "extension_context",
    window: context.title || tab.title || "Browser Review",
    url: context.url || tab.url || "",
    route: context.route || "",
    target_tag: context.dom?.tag || "",
    target_id: context.dom?.id || "",
    target_test_id: context.dom?.test_id || "",
    target_role: context.dom?.role || "",
    target_label: context.dom?.label || "",
    target_selector: context.dom?.selector || "",
    dom: context.dom || null
  });
}

async function postQueuedSnapshotPointerOrCurrent() {
  if (pendingSnapshotContext?.sessionID && pendingSnapshotContext?.tab && pendingSnapshotContext?.context) {
    await postCurrentBrowserPointer(
      pendingSnapshotContext.sessionID,
      pendingSnapshotContext.tab,
      pendingSnapshotContext.context
    );
    return;
  }
  const { current, tab, context } = await currentSessionBundle();
  await postCurrentBrowserPointer(current.session.id, tab, context);
}

async function captureVisibleTabBlob(windowId) {
  const dataURL = await chrome.tabs.captureVisibleTab(windowId, { format: "png" });
  const res = await fetch(dataURL);
  return await res.blob();
}

function createAudioRecorderForStream(stream) {
  try {
    return new MediaRecorder(stream, { mimeType: "audio/webm" });
  } catch (_) {
    return new MediaRecorder(stream);
  }
}

function createVideoRecorderForStream(stream) {
  const options = [
    { mimeType: "video/webm;codecs=vp9" },
    { mimeType: "video/webm;codecs=vp8" },
    { mimeType: "video/webm" }
  ];
  for (const option of options) {
    try {
      if (!MediaRecorder.isTypeSupported || MediaRecorder.isTypeSupported(option.mimeType)) {
        return new MediaRecorder(stream, option);
      }
    } catch (_) {}
  }
  return new MediaRecorder(stream);
}

function stopStream(stream) {
  if (!stream) return;
  try {
    stream.getTracks().forEach((track) => track.stop());
  } catch (_) {}
}

function normalizeCaptureError(err, kind) {
  const name = String(err?.name || "").trim();
  const message = String(err?.message || "").trim();
  const lowered = message.toLowerCase();
  if (name === "NotAllowedError" || lowered.includes("permission dismissed") || lowered.includes("permission denied")) {
    if (kind === "video") {
      return "Tab or microphone access was dismissed. Try the video button again and allow access.";
    }
    return "Microphone access was dismissed. Try the microphone button again and allow access.";
  }
  return message || (kind === "video" ? "Video recording failed." : "Audio recording failed.");
}

function clearAudioNoteCapture() {
  stopStream(audioNoteStream);
  audioNoteStream = null;
  audioNoteRecorder = null;
  audioNoteChunks = [];
  audioNoteStopPromise = null;
  if (recordingKind === "audio") {
    recordingKind = "";
  }
  renderComposerControls();
}

function clearVideoNoteCapture() {
  stopStream(videoNoteMicStream);
  stopStream(videoNoteDisplayStream);
  videoNoteMicStream = null;
  videoNoteDisplayStream = null;
  videoNoteAudioRecorder = null;
  videoNoteClipRecorder = null;
  videoNoteAudioChunks = [];
  videoNoteClipChunks = [];
  videoNoteAudioStopPromise = null;
  videoNoteClipStopPromise = null;
  videoNoteStartedAt = 0;
  if (previewVideoEl) {
    previewVideoEl.pause();
    previewVideoEl.srcObject = null;
    previewVideoEl.classList.add("hidden");
  }
  if (recordingKind === "video") {
    recordingKind = "";
  }
  renderComposerControls();
}

function toggleTextEditor() {
  if (textEditorOpen && typedNoteValue()) {
    submitTypedNote().catch((err) => setStatus(err.message, true));
    return;
  }
  textEditorOpen = !textEditorOpen;
  renderTextEditorState();
  if (textEditorOpen) {
    window.setTimeout(() => {
      try {
        noteTextEl?.focus();
      } catch (_) {}
    }, 0);
  }
}

async function submitTypedNote(options = {}) {
  const skipAutoPreview = !!options.skipAutoPreview;
  const note = typedNoteValue();
  if (!note) {
    textEditorOpen = true;
    renderTextEditorState();
    try {
      noteTextEl.focus();
    } catch (_) {}
    throw new Error("Type a note first.");
  }
  setRequestInFlight(true);
  setRequestActivity("Saving typed note…");
  let usedQueuedSnapshot = false;
  try {
    previewSubmittedMessage = "";
    const form = new FormData();
    if (hasPendingSnapshot()) {
      usedQueuedSnapshot = true;
      await postQueuedSnapshotPointerOrCurrent();
      form.append("screenshot", pendingSnapshotBlob, queuedSnapshotFilename());
    } else {
      const { current, tab, context } = await currentSessionBundle();
      await postCurrentBrowserPointer(current.session.id, tab, context);
    }
    form.append("raw_transcript", note);
    form.append("normalized", note);
    const result = await postForm("/api/session/feedback/note", form);
    clearPendingSnapshot();
    noteTextEl.value = "";
    textEditorOpen = false;
    renderTextEditorState();
    await refreshSession(true);
    if (!skipAutoPreview) {
      await refreshPreviewAuto({ preserveStatus: true });
    } else {
      renderPreviewSurface();
    }
    setStatus((usedQueuedSnapshot ? "Typed note captured with snapshot: " : "Typed note captured: ") + String(result?.event_id || "ready for preview") + ".");
  } finally {
    setRequestInFlight(false);
  }
}

async function submitSnapshotNote() {
  setRequestInFlight(true);
  setRequestActivity("Capturing snapshot…");
  try {
    previewSubmittedMessage = "";
    const { current, tab, context } = await currentSessionBundle();
    const screenshot = await captureVisibleTabBlob(tab.windowId);
    const snapshotContext = {
      sessionID: current.session.id,
      tab: {
        title: tab.title || "",
        url: tab.url || ""
      },
      context
    };
    await savePendingSnapshotState(screenshot, snapshotContext);
    textEditorOpen = true;
    renderComposerControls();
    renderTextEditorState();
    try {
      noteTextEl.focus();
    } catch (_) {}
    setStatus("Snapshot queued. Type a note and press Cmd/Ctrl+Enter, or record audio or video next.");
  } finally {
    setRequestInFlight(false);
  }
}

async function startAudioNoteCapture() {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
  const recorder = createAudioRecorderForStream(stream);
  audioNoteStream = stream;
  audioNoteChunks = [];
  audioNoteStopPromise = new Promise((resolve) => {
    recorder.addEventListener("stop", resolve, { once: true });
  });
  recorder.ondataavailable = (evt) => {
    if (evt.data && evt.data.size > 0) {
      audioNoteChunks.push(evt.data);
    }
  };
  audioNoteRecorder = recorder;
  recordingKind = "audio";
  recorder.start();
  renderComposerControls();
  setStatus("Recording audio note. Click the microphone again to stop.");
}

async function finishAudioNoteCapture() {
  if (!audioNoteRecorder) {
    throw new Error("Audio note recording is not active.");
  }
  const recorder = audioNoteRecorder;
  const stopPromise = audioNoteStopPromise;
  if (recorder.state !== "inactive") {
    recorder.stop();
  }
  if (stopPromise) {
    await stopPromise;
  }
  const blob = new Blob(audioNoteChunks, { type: recorder.mimeType || "audio/webm" });
  clearAudioNoteCapture();
  return blob;
}

async function handleAudioNote() {
  if (recordingKind === "video" || videoNoteFinalizing) {
    throw new Error("Stop the current tab recording before starting audio.");
  }
  if (!audioNoteRecorder) {
    setRequestInFlight(true);
    setRequestActivity("Starting microphone…");
    try {
      previewSubmittedMessage = "";
      await currentSessionBundle();
      await startAudioNoteCapture();
    } finally {
      setRequestInFlight(false);
    }
    return;
  }
  setRequestInFlight(true);
  setRequestActivity("Saving audio note…");
  try {
    const audio = await finishAudioNoteCapture();
    const form = new FormData();
    const usedQueuedSnapshot = hasPendingSnapshot();
    if (usedQueuedSnapshot) {
      await postQueuedSnapshotPointerOrCurrent();
      form.append("screenshot", pendingSnapshotBlob, queuedSnapshotFilename());
    } else {
      const { current, tab, context } = await currentSessionBundle();
      await postCurrentBrowserPointer(current.session.id, tab, context);
    }
    form.append("audio", audio, "extension-note.webm");
    const result = await postForm("/api/session/feedback/note", form);
    clearPendingSnapshot();
    await refreshSession(true);
    await refreshPreviewAuto({ preserveStatus: true });
    setStatus((usedQueuedSnapshot ? "Audio note captured with snapshot: " : "Audio note captured: ") + String(result?.event_id || "ready for preview") + ".");
  } finally {
    setRequestInFlight(false);
  }
}

async function captureCurrentTabStream(streamID) {
  const id = String(streamID || "").trim();
  if (!id) {
    throw new Error("Current-tab capture is not available for this recording.");
  }
  return await navigator.mediaDevices.getUserMedia({
    audio: {
      mandatory: {
        chromeMediaSource: "tab",
        chromeMediaSourceId: id
      }
    },
    video: {
      mandatory: {
        chromeMediaSource: "tab",
        chromeMediaSourceId: id,
        maxFrameRate: 18
      }
    }
  });
}

async function startVideoNoteCapture() {
  const tab = await activeTab();
  const tabStreamID = await chrome.tabCapture.getMediaStreamId({ targetTabId: tab.id });
  try {
    videoNoteDisplayStream = await captureCurrentTabStream(tabStreamID);
    videoNoteMicStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
    const clipRecorder = createVideoRecorderForStream(videoNoteDisplayStream);
    const audioRecorder = createAudioRecorderForStream(videoNoteMicStream);
    videoNoteClipRecorder = clipRecorder;
    videoNoteAudioRecorder = audioRecorder;
    videoNoteClipChunks = [];
    videoNoteAudioChunks = [];
    videoNoteStartedAt = Date.now();
    videoNoteClipStopPromise = new Promise((resolve) => {
      clipRecorder.addEventListener("stop", resolve, { once: true });
    });
    videoNoteAudioStopPromise = new Promise((resolve) => {
      audioRecorder.addEventListener("stop", resolve, { once: true });
    });
    clipRecorder.ondataavailable = (evt) => {
      if (evt.data && evt.data.size > 0) {
        videoNoteClipChunks.push(evt.data);
      }
    };
    audioRecorder.ondataavailable = (evt) => {
      if (evt.data && evt.data.size > 0) {
        videoNoteAudioChunks.push(evt.data);
      }
    };
    const [videoTrack] = videoNoteDisplayStream.getVideoTracks();
    if (videoTrack) {
      videoTrack.addEventListener("ended", () => {
        if (recordingKind === "video" && !videoNoteFinalizing) {
          finalizeVideoNoteCapture("tab recording ended").catch((err) => setStatus(normalizeCaptureError(err, "video"), true));
        }
      }, { once: true });
    }
    previewVideoEl.srcObject = videoNoteDisplayStream;
    previewVideoEl.classList.remove("hidden");
    recordingKind = "video";
    clipRecorder.start();
    audioRecorder.start();
    renderComposerControls();
    setStatus("Recording the current tab. Click the video button again to stop.");
  } catch (err) {
    clearVideoNoteCapture();
    throw err;
  }
}

async function finishVideoNoteCapture() {
  if (!videoNoteAudioRecorder || !videoNoteClipRecorder) {
    throw new Error("Tab recording is not active.");
  }
  const audioRecorder = videoNoteAudioRecorder;
  const clipRecorder = videoNoteClipRecorder;
  const audioStopPromise = videoNoteAudioStopPromise;
  const clipStopPromise = videoNoteClipStopPromise;
  if (audioRecorder.state !== "inactive") {
    audioRecorder.stop();
  }
  if (clipRecorder.state !== "inactive") {
    clipRecorder.stop();
  }
  await Promise.all([audioStopPromise, clipStopPromise].filter(Boolean));
  const audioBlob = new Blob(videoNoteAudioChunks, { type: audioRecorder.mimeType || "audio/webm" });
  const clipBlob = new Blob(videoNoteClipChunks, { type: clipRecorder.mimeType || "video/webm" });
  const clip = {
    blob: clipBlob,
    codec: clipRecorder.mimeType || "video/webm",
    hasAudio: false,
    scope: "tab",
    window: "Browser tab",
    durationMS: Math.max(0, Date.now() - videoNoteStartedAt)
  };
  clearVideoNoteCapture();
  return { audioBlob, clip };
}

function appendClipMetadata(form, clip) {
  if (!clip) return;
  if (clip.codec) form.append("video_codec", String(clip.codec));
  if (clip.scope) form.append("video_scope", String(clip.scope));
  if (clip.window) form.append("video_window", String(clip.window));
  if (Number.isFinite(Number(clip.durationMS)) && Number(clip.durationMS) > 0) {
    form.append("video_duration_ms", String(Math.round(Number(clip.durationMS))));
  }
  form.append("video_has_audio", clip.hasAudio ? "1" : "0");
}

async function finalizeVideoNoteCapture(trigger = "manual stop") {
  if (videoNoteFinalizing) {
    return;
  }
  videoNoteFinalizing = true;
  setRequestActivity("Saving tab recording…");
  renderComposerControls();
  setRequestInFlight(true);
  try {
    const bundle = await finishVideoNoteCapture();
    if (!bundle?.audioBlob || !bundle?.clip?.blob?.size) {
      throw new Error("Video note could not be recorded.");
    }
    const form = new FormData();
    const usedQueuedSnapshot = hasPendingSnapshot();
    if (usedQueuedSnapshot) {
      await postQueuedSnapshotPointerOrCurrent();
      form.append("screenshot", pendingSnapshotBlob, queuedSnapshotFilename());
    } else {
      const { current, tab, context } = await currentSessionBundle();
      await postCurrentBrowserPointer(current.session.id, tab, context);
    }
    form.append("audio", bundle.audioBlob, "extension-video-note.webm");
    const note = await postForm("/api/session/feedback/note", form);
    const eventID = String(note?.event_id || "");
    if (!eventID) {
      throw new Error("Video note was captured but the event could not be created.");
    }
    const clipForm = new FormData();
    clipForm.append("event_id", eventID);
    clipForm.append("clip", bundle.clip.blob, "extension-video-clip.webm");
    appendClipMetadata(clipForm, bundle.clip);
    await postForm("/api/session/feedback/clip", clipForm);
    clearPendingSnapshot();
    await refreshSession(true);
    await refreshPreviewAuto({ preserveStatus: true });
    setStatus((usedQueuedSnapshot ? "Video note captured with snapshot: " : "Video note captured: ") + eventID + " (" + trigger + ").");
  } finally {
    videoNoteFinalizing = false;
    setRequestInFlight(false);
    renderComposerControls();
  }
}

async function handleVideoNote() {
  if (recordingKind === "audio") {
    throw new Error("Stop the current audio note before starting a tab recording.");
  }
  if (!videoNoteClipRecorder) {
    setRequestInFlight(true);
    setRequestActivity("Starting tab recording…");
    try {
      previewSubmittedMessage = "";
      await currentSessionBundle();
      await startVideoNoteCapture();
    } finally {
      setRequestInFlight(false);
    }
    return;
  }
  await finalizeVideoNoteCapture("manual stop");
}

async function approveSession(silent = false, reason = "preview") {
  const summary = typedNoteValue();
  await postJSON("/api/session/approve", { summary });
  await refreshSession(true);
  if (!silent) {
    setStatus("Package approved for " + reason + ".");
  }
}

async function ensureFeedbackPresent() {
  const feedback = Array.isArray(currentSessionPayload?.session?.feedback) ? currentSessionPayload.session.feedback : [];
  if (feedback.length > 0) {
    return;
  }
  await refreshSession(true);
  const nextFeedback = Array.isArray(currentSessionPayload?.session?.feedback) ? currentSessionPayload.session.feedback : [];
  if (nextFeedback.length > 0) {
    return;
  }
  throw new Error("Capture at least one note first.");
}

async function preview(silent = false) {
  setPreviewLoading("Loading preview…");
  try {
    if (!silent) {
      setRequestInFlight(true);
      setRequestActivity("Refreshing preview…");
    }
    if (textEditorOpen && typedNoteValue()) {
      await submitTypedNote({ skipAutoPreview: true });
    }
    await ensureFeedbackPresent();
    await approveSession(true, "preview");
    const data = await postJSON("/api/session/payload/preview", {
      provider: "",
      omit_video_event_ids: previewDeliveryOptions.omitVideoEventIDs || []
    });
    latestPreviewPayload = data?.preview || {};
    renderPreviewSurface();
    if (!silent) {
      setStatus("Preview refreshed.");
    }
  } catch (err) {
    renderPreviewSurface();
    throw err;
  } finally {
    if (!silent) {
      setRequestInFlight(false);
    }
  }
}

async function deletePreviewNote(eventID) {
  const trimmedEventID = String(eventID || "").trim();
  if (!trimmedEventID) {
    throw new Error("Preview request could not be removed because it has no event id.");
  }
  setRequestInFlight(true);
  setRequestActivity("Removing request…");
  setPreviewLoading("Removing request…");
  try {
    await postJSON("/api/session/feedback/delete", { event_id: trimmedEventID });
    setPreviewVideoEventOmitted(trimmedEventID, false);
    await refreshSession(true);
    await preview(true);
    setStatus("Queued request removed.");
  } finally {
    setRequestInFlight(false);
  }
}

async function togglePreviewVideoEventOmission(eventID) {
  const id = String(eventID || "").trim();
  if (!id) return;
  setPreviewVideoEventOmitted(id, !previewVideoEventOmitted(id));
  await preview(true);
  setStatus(previewVideoEventOmitted(id) ? "Large clip will be omitted for this request." : "Large clip will be sent again for this request.");
}

async function submit() {
  setRequestInFlight(true);
  setRequestActivity("Submitting request…");
  try {
    if (textEditorOpen && typedNoteValue()) {
      await submitTypedNote({ skipAutoPreview: true });
    }
    await ensureFeedbackPresent();
    await approveSession(true, "submit");
    const data = await postJSON("/api/session/submit", {
      provider: "",
      omit_video_event_ids: previewDeliveryOptions.omitVideoEventIDs || []
    });
    latestPreviewPayload = null;
    resetPreviewDeliveryOptions();
    noteTextEl.value = "";
    textEditorOpen = false;
    renderTextEditorState();
    clearPendingSnapshot();
    renderPreviewSurface();
    await refreshSession(true);
    const provider = String(data?.provider || "default adapter").trim() || "default adapter";
    const attemptID = String(data?.attempt_id || "").trim();
    watchedSubmitAttemptID = attemptID;
    if (attemptID) ensureSubmitAttemptWatchTimer();
    const message = `Request queued via ${provider}${attemptID ? ` (${attemptID})` : ""}.`;
    setStatus("");
    setSubmitNotice(message);
    await callBackground({
      type: "knit:notify-submit",
      payload: {
        message,
        attemptID,
        provider
      }
    });
  } catch (err) {
    const message = String(err?.message || "").trim();
    if (message.includes("over the default send limit") || message.includes("explicit decision")) {
      await preview(true).catch(() => {});
      setStatus("Submission blocked until you choose how to handle the large clip in preview.", true);
      return;
    }
    throw err;
  } finally {
    setRequestInFlight(false);
  }
}

async function stopSession() {
  await postJSON("/api/session/stop", {});
  latestPreviewPayload = null;
  resetPreviewDeliveryOptions();
  clearPendingSnapshot();
  noteTextEl.value = "";
  textEditorOpen = false;
  renderTextEditorState();
  renderPreviewSurface();
  await refreshSession(true);
  setStatus("Session stopped.");
}

noteTextEl.addEventListener("input", renderTextEditorState);
noteTextEl.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
    event.preventDefault();
    submitTypedNote().catch((err) => setStatus(err.message, true));
  }
});
toggleTextBtnEl.addEventListener("click", toggleTextEditor);
snapshotBtnEl.addEventListener("click", () => submitSnapshotNote().catch((err) => setStatus(err.message, true)));
audioBtnEl.addEventListener("click", () => handleAudioNote().catch((err) => setStatus(normalizeCaptureError(err, "audio"), true)));
videoBtnEl.addEventListener("click", () => handleVideoNote().catch((err) => setStatus(normalizeCaptureError(err, "video"), true)));
previewBtnEl.addEventListener("click", () => preview().catch((err) => setStatus(err.message, true)));
submitBtnEl.addEventListener("click", () => submit().catch((err) => setStatus(err.message, true)));
stopBtnEl.addEventListener("click", () => stopSession().catch((err) => setStatus(err.message, true)));
themeToggleBtnEl?.addEventListener("click", () => toggleTheme().catch((err) => setStatus(err.message, true)));
previewEl.addEventListener("click", (event) => {
  const button = event.target instanceof Element ? event.target.closest("[data-preview-action]") : null;
  if (!button) return;
  const action = String(button.getAttribute("data-preview-action") || "").trim();
  const eventID = String(button.getAttribute("data-event-id") || "").trim();
  if (action === "delete") {
    deletePreviewNote(eventID).catch((err) => setStatus(err.message, true));
    return;
  }
  if (action === "toggle-video-omit") {
    togglePreviewVideoEventOmission(eventID).catch((err) => setStatus(err.message, true));
  }
});
queueIndicatorEl?.addEventListener("click", (event) => {
  const button = event.target instanceof Element ? event.target.closest("[data-attempt-link]") : null;
  if (!button) return;
  const attemptID = String(button.getAttribute("data-attempt-link") || "").trim();
  if (!attemptID) return;
  openMainUIForAttempt(attemptID).catch((err) => setStatus(err.message, true));
});

window.addEventListener("beforeunload", () => {
  stopSubmitAttemptWatchTimer();
  stopSessionRefreshTimer();
  stopVideoGuidanceTimer();
  stopStream(videoNoteMicStream);
  stopStream(videoNoteDisplayStream);
  stopStream(audioNoteStream);
});

renderComposerControls();
loadTheme()
  .then(() => loadConfig())
  .then(restorePendingSnapshotState)
  .then(() => refreshSession())
  .then(() => currentSessionPayload?.session?.id ? refreshPreviewAuto({ preserveStatus: true }) : null)
  .then(consumeSubmitNotice)
  .catch((err) => setStatus(err.message, true));
