const DEFAULT_DAEMON_BASE_URL = "http://127.0.0.1:7777";
const DEFAULT_ACTION_TITLE = "Knit";
const SUBMIT_NOTICE_KEY = "lastSubmitNotice";
const SIDE_PANEL_PATH = "recorder.html";
const BOUND_TAB_ID_KEY = "composerBoundTabId";
const BOUND_WINDOW_ID_KEY = "composerBoundWindowId";

async function applySidePanelBinding() {
  if (!chrome.sidePanel?.setOptions) return;
  await chrome.sidePanel.setOptions({ path: SIDE_PANEL_PATH, enabled: false });
  const items = await chrome.storage.local.get([BOUND_TAB_ID_KEY]);
  const tabId = Number(items?.[BOUND_TAB_ID_KEY]);
  if (!Number.isInteger(tabId)) return;
  try {
    await chrome.sidePanel.setOptions({ tabId, path: SIDE_PANEL_PATH, enabled: true });
  } catch (_) {
    await chrome.storage.local.remove([BOUND_TAB_ID_KEY, BOUND_WINDOW_ID_KEY]);
  }
}

async function bindSidePanelToTab(tabId, windowId) {
  await chrome.storage.local.set({
    [BOUND_TAB_ID_KEY]: tabId,
    [BOUND_WINDOW_ID_KEY]: windowId
  });
  await applySidePanelBinding();
}

async function clearSidePanelBinding() {
  await chrome.storage.local.remove([BOUND_TAB_ID_KEY, BOUND_WINDOW_ID_KEY]);
  await applySidePanelBinding();
}

function normalizeSubmitNotice(payload) {
  const attemptID = String(payload?.attemptID || "").trim();
  const provider = String(payload?.provider || "").trim();
  const message = String(payload?.message || "Request submitted to daemon queue.").trim();
  return {
    message,
    attemptID,
    provider,
    createdAt: new Date().toISOString()
  };
}

function setSubmitBadge(message) {
  const title = String(message || DEFAULT_ACTION_TITLE).trim() || DEFAULT_ACTION_TITLE;
  chrome.action.setBadgeBackgroundColor({ color: "#0e766e" });
  if (chrome.action.setBadgeTextColor) {
    chrome.action.setBadgeTextColor({ color: "#ffffff" });
  }
  chrome.action.setBadgeText({ text: "1" });
  chrome.action.setTitle({ title });
}

function clearSubmitBadge() {
  chrome.action.setBadgeText({ text: "" });
  chrome.action.setTitle({ title: DEFAULT_ACTION_TITLE });
}

chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.get(["daemonBaseURL"], (items) => {
    if (!items || !items.daemonBaseURL) {
      chrome.storage.local.set({ daemonBaseURL: DEFAULT_DAEMON_BASE_URL });
    }
  });
  clearSubmitBadge();
  void applySidePanelBinding();
});

chrome.runtime.onStartup.addListener(() => {
  void applySidePanelBinding();
});

chrome.tabs.onRemoved.addListener((tabId) => {
  chrome.storage.local.get([BOUND_TAB_ID_KEY], (items) => {
    if (Number(items?.[BOUND_TAB_ID_KEY]) === tabId) {
      void clearSidePanelBinding();
    }
  });
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message && message.type === "knit:get-config") {
    chrome.storage.local.get(["daemonBaseURL", "daemonToken"], (items) => sendResponse(items || {}));
    return true;
  }
  if (message && message.type === "knit:set-config") {
    chrome.storage.local.set(message.payload || {}, () => sendResponse({ ok: true }));
    return true;
  }
  if (message && message.type === "knit:bind-side-panel") {
    const tabId = Number(message?.payload?.tabId);
    const windowId = Number(message?.payload?.windowId);
    void bindSidePanelToTab(tabId, windowId).then(() => sendResponse({ ok: true }));
    return true;
  }
  if (message && message.type === "knit:clear-side-panel-binding") {
    void clearSidePanelBinding().then(() => sendResponse({ ok: true }));
    return true;
  }
  if (message && message.type === "knit:notify-submit") {
    const notice = normalizeSubmitNotice(message.payload || {});
    chrome.storage.local.set({ [SUBMIT_NOTICE_KEY]: notice }, () => {
      setSubmitBadge(notice.message);
      sendResponse({ ok: true, notice });
    });
    return true;
  }
  if (message && message.type === "knit:consume-submit-notice") {
    chrome.storage.local.get([SUBMIT_NOTICE_KEY], (items) => {
      const notice = items?.[SUBMIT_NOTICE_KEY] || null;
      chrome.storage.local.remove([SUBMIT_NOTICE_KEY], () => {
        clearSubmitBadge();
        sendResponse({ ok: true, notice });
      });
    });
    return true;
  }
  return false;
});
