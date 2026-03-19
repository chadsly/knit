const daemonBaseURLEl = document.getElementById("daemonBaseURL");
const pairingCodeEl = document.getElementById("pairingCode");
const pairBtnEl = document.getElementById("pairBtn");
const pairingCardEl = document.getElementById("pairingCard");
const launcherCardEl = document.getElementById("launcherCard");
const openComposerBtnEl = document.getElementById("openComposerBtn");
const unlinkBtnEl = document.getElementById("unlinkBtn");
const submitNoticeEl = document.getElementById("submitNotice");
const statusEl = document.getElementById("status");

let config = {
  daemonBaseURL: "http://127.0.0.1:7777",
  daemonToken: ""
};
let requestInFlight = false;

async function callBackground(message) {
  return chrome.runtime.sendMessage(message);
}

async function loadConfig() {
  const stored = await callBackground({ type: "knit:get-config" });
  config = {
    daemonBaseURL: String(stored?.daemonBaseURL || "http://127.0.0.1:7777").trim(),
    daemonToken: String(stored?.daemonToken || "").trim()
  };
  daemonBaseURLEl.value = config.daemonBaseURL;
}

async function saveConfig(patch) {
  config = { ...config, ...patch };
  await callBackground({ type: "knit:set-config", payload: config });
}

function setStatus(message, isError = false) {
  statusEl.textContent = String(message || "").trim();
  statusEl.classList.toggle("error", !!isError);
}

function setSubmitNotice(message) {
  const text = String(message || "").trim();
  submitNoticeEl.textContent = text;
  submitNoticeEl.classList.toggle("hidden", !text);
}

async function consumeSubmitNotice() {
  const data = await callBackground({ type: "knit:consume-submit-notice" });
  setSubmitNotice(String(data?.notice?.message || "").trim());
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

function renderPopupState() {
  const paired = !!config.daemonToken;
  pairingCardEl.classList.toggle("hidden", paired);
  launcherCardEl.classList.toggle("hidden", !paired);
  pairBtnEl.disabled = requestInFlight;
  openComposerBtnEl.disabled = requestInFlight || !paired;
  unlinkBtnEl.disabled = requestInFlight || !paired;
}

async function activeTab() {
  const tabs = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
  if (!tabs.length) {
    throw new Error("No active browser tab found.");
  }
  return tabs[0];
}

async function openComposerPanel() {
  const tab = await activeTab();
  if (!Number.isInteger(tab.id)) {
    throw new Error("The current tab could not be bound to the browser composer.");
  }
  await callBackground({
    type: "knit:bind-side-panel",
    payload: {
      tabId: tab.id,
      windowId: tab.windowId
    }
  });
  await chrome.sidePanel.open({ tabId: tab.id });
  setStatus("The browser composer opened in the extension side panel for this tab.");
}

async function refreshConnectionStatus() {
  renderPopupState();
  if (!config.daemonToken) {
    setSubmitNotice("");
    setStatus("Enter the pairing code from the main Knit UI to connect this browser.");
    return;
  }
  try {
    await request("/api/extension/session", { headers: authHeaders(false) });
    setStatus("Ready. Open the browser composer from the extension side panel.");
  } catch (err) {
    setStatus("Daemon connection failed: " + err.message, true);
  }
}

async function pairExtension() {
  const daemonBaseURL = String(daemonBaseURLEl.value || "").trim() || "http://127.0.0.1:7777";
  const pairingCode = String(pairingCodeEl.value || "").trim().toUpperCase();
  if (!pairingCode) {
    setStatus("Enter the pairing code from the main Knit UI first.", true);
    return;
  }
  requestInFlight = true;
  renderPopupState();
  try {
    await saveConfig({ daemonBaseURL });
    const data = await request("/api/extension/pair/complete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        pairing_code: pairingCode,
        name: "Chromium Side Panel",
        browser: "chromium",
        platform: navigator.platform || "unknown"
      })
    });
    await saveConfig({
      daemonBaseURL,
      daemonToken: String(data?.token || "").trim()
    });
    pairingCodeEl.value = "";
    await consumeSubmitNotice();
    await refreshConnectionStatus();
    await openComposerPanel();
    setStatus("Extension paired. Opening the browser composer in the side panel.");
  } catch (err) {
    setStatus("Pairing failed: " + err.message, true);
  } finally {
    requestInFlight = false;
    renderPopupState();
  }
}

async function unpair() {
  requestInFlight = true;
  renderPopupState();
  try {
    await callBackground({ type: "knit:clear-side-panel-binding" });
    await saveConfig({ daemonToken: "" });
    setSubmitNotice("");
    setStatus("Extension unpaired. Pair again from the main Knit UI when needed.");
    await refreshConnectionStatus();
  } finally {
    requestInFlight = false;
    renderPopupState();
  }
}

pairBtnEl.addEventListener("click", () => pairExtension().catch((err) => setStatus(err.message, true)));
openComposerBtnEl.addEventListener("click", () => openComposerPanel().catch((err) => setStatus(err.message, true)));
unlinkBtnEl.addEventListener("click", () => unpair().catch((err) => setStatus(err.message, true)));

loadConfig()
  .then(consumeSubmitNotice)
  .then(refreshConnectionStatus)
  .catch((err) => setStatus(err.message, true));
