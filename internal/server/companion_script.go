package server

const companionJS = `(function () {
  const controlToken = '__KNIT_TOKEN__';
  const script = document.currentScript;
  const origin = script ? new URL(script.src).origin : 'http://127.0.0.1:7777';
  const stateURL = origin + '/api/state';
  const pointerURL = origin + '/api/companion/pointer';
  const maxContextItems = 12;
  const contextSyncMS = 1500;
  const stateSyncMS = 4000;

  if (typeof window.__knitCompanionStop === 'function') {
    try { window.__knitCompanionStop(); } catch (_) {}
  }

  let sessionId = '';
  let captureInputValues = false;
  let lastX = 0;
  let lastY = 0;
  let hoverStart = Date.now();
  let lastMoveSent = 0;
  let lastContextSent = 0;
  let lastStateSync = 0;
  let dragging = false;
  let lastResyncAttempt = 0;
  let moveThrottleMS = 33;
  let lastTarget = null;
  let contextTimer = 0;

  const recentConsole = [];
  const recentNetwork = [];
  const originalConsole = {};
  const baseFetch = typeof window.fetch === 'function' ? window.fetch.bind(window) : null;
  const originalFetch = baseFetch;
  const originalXHROpen = XMLHttpRequest.prototype.open;
  const originalXHRSend = XMLHttpRequest.prototype.send;

  async function bootstrap() {
    try {
      const state = await fetchStateSummary();
      applyState(state);
      if (!sessionId) {
        console.warn('[knit companion] no active session');
        return;
      }
      bind();
      console.info('[knit companion] attached to session', sessionId);
    } catch (err) {
      console.error('[knit companion] bootstrap failed', err);
    }
  }

  async function fetchStateSummary() {
    if (!baseFetch) return null;
    const response = await baseFetch(stateURL, { mode: 'cors', headers: { 'X-Knit-Token': controlToken } });
    lastStateSync = Date.now();
    return response.json();
  }

  function applyState(state) {
    const hz = Number(state && state.pointer_sample_hz || 30);
    if (Number.isFinite(hz) && hz > 0) {
      moveThrottleMS = Math.max(8, Math.round(1000 / hz));
    }
    sessionId = state && state.session && state.session.id ? String(state.session.id) : '';
    captureInputValues = !!(state && state.session && state.session.capture_input_values);
  }

  function bind() {
    document.addEventListener('mousemove', onMove, true);
    document.addEventListener('click', onClick, true);
    document.addEventListener('mousedown', onMouseDown, true);
    document.addEventListener('mouseup', onMouseUp, true);
    document.addEventListener('wheel', onWheel, { passive: true, capture: true });
    document.addEventListener('keydown', onKeyDown, true);
    document.addEventListener('keyup', onKeyUp, true);
    document.addEventListener('focus', onFocus, true);
    document.addEventListener('blur', onBlur, true);
    document.addEventListener('input', onInput, true);
    document.addEventListener('change', onChange, true);
    document.addEventListener('submit', onSubmit, true);
    patchConsole();
    patchFetch();
    patchXHR();
    contextTimer = window.setInterval(() => {
      if (!sessionId) return;
      if ((Date.now() - lastStateSync) >= stateSyncMS) {
        refreshState();
      }
      send('context_sync', lastX, lastY, Date.now() - hoverStart, lastTarget, {}, true);
    }, contextSyncMS);
    window.__knitCompanionStop = function () {
      document.removeEventListener('mousemove', onMove, true);
      document.removeEventListener('click', onClick, true);
      document.removeEventListener('mousedown', onMouseDown, true);
      document.removeEventListener('mouseup', onMouseUp, true);
      document.removeEventListener('wheel', onWheel, true);
      document.removeEventListener('keydown', onKeyDown, true);
      document.removeEventListener('keyup', onKeyUp, true);
      document.removeEventListener('focus', onFocus, true);
      document.removeEventListener('blur', onBlur, true);
      document.removeEventListener('input', onInput, true);
      document.removeEventListener('change', onChange, true);
      document.removeEventListener('submit', onSubmit, true);
      if (contextTimer) {
        clearInterval(contextTimer);
        contextTimer = 0;
      }
      restoreConsole();
      restoreFetch();
      restoreXHR();
      console.info('[knit companion] detached');
    };
  }

  function onMove(e) {
    const now = Date.now();
    lastTarget = e.target || lastTarget;
    if (e.clientX !== lastX || e.clientY !== lastY) {
      hoverStart = now;
      lastX = e.clientX;
      lastY = e.clientY;
    }
    if (now - lastMoveSent < moveThrottleMS) return;
    lastMoveSent = now;
    send('move', e.clientX, e.clientY, now - hoverStart, e.target);
    if (dragging) {
      send('drag_move', e.clientX, e.clientY, now - hoverStart, e.target, { mouse_button: Number(e.button || 0) });
    }
  }

  function onClick(e) {
    lastTarget = e.target || lastTarget;
    send('click', e.clientX, e.clientY, Date.now() - hoverStart, e.target, {
      mouse_button: Number(e.button || 0),
      click_count: Number(e.detail || 1),
      modifiers: eventModifiers(e)
    }, true);
  }

  function onMouseDown(e) {
    dragging = true;
    lastTarget = e.target || lastTarget;
    send('drag_start', e.clientX, e.clientY, Date.now() - hoverStart, e.target, {
      mouse_button: Number(e.button || 0),
      modifiers: eventModifiers(e)
    }, true);
  }

  function onMouseUp(e) {
    lastTarget = e.target || lastTarget;
    if (dragging) {
      send('drag_end', e.clientX, e.clientY, Date.now() - hoverStart, e.target, {
        mouse_button: Number(e.button || 0),
        modifiers: eventModifiers(e)
      }, true);
    }
    dragging = false;
  }

  function onWheel(e) {
    lastTarget = e.target || lastTarget;
    send('scroll', e.clientX, e.clientY, Date.now() - hoverStart, e.target, {
      scroll_dx: Number.isFinite(e.deltaX) ? e.deltaX : 0,
      scroll_dy: Number.isFinite(e.deltaY) ? e.deltaY : 0,
      modifiers: eventModifiers(e)
    }, true);
  }

  function onKeyDown(e) {
    lastTarget = e.target || lastTarget;
    send('keydown', lastX, lastY, Date.now() - hoverStart, e.target, {
      key: String(e.key || ''),
      code: String(e.code || ''),
      modifiers: eventModifiers(e),
      input_type: elementInputType(e.target)
    }, true);
  }

  function onKeyUp(e) {
    lastTarget = e.target || lastTarget;
    send('keyup', lastX, lastY, Date.now() - hoverStart, e.target, {
      key: String(e.key || ''),
      code: String(e.code || ''),
      modifiers: eventModifiers(e),
      input_type: elementInputType(e.target)
    }, true);
  }

  function onFocus(e) {
    lastTarget = e.target || lastTarget;
    send('focus', lastX, lastY, Date.now() - hoverStart, e.target, {
      input_type: elementInputType(e.target)
    }, true);
  }

  function onBlur(e) {
    lastTarget = e.target || lastTarget;
    send('blur', lastX, lastY, Date.now() - hoverStart, e.target, {
      input_type: elementInputType(e.target)
    }, true);
  }

  function onInput(e) {
    lastTarget = e.target || lastTarget;
    const valueInfo = inputValuePayload(e.target);
    send('input', lastX, lastY, Date.now() - hoverStart, e.target, {
      input_type: elementInputType(e.target),
      value: valueInfo.value,
      value_captured: valueInfo.valueCaptured,
      value_redacted: valueInfo.valueRedacted
    }, true);
  }

  function onChange(e) {
    lastTarget = e.target || lastTarget;
    const valueInfo = inputValuePayload(e.target);
    send('change', lastX, lastY, Date.now() - hoverStart, e.target, {
      input_type: elementInputType(e.target),
      value: valueInfo.value,
      value_captured: valueInfo.valueCaptured,
      value_redacted: valueInfo.valueRedacted
    }, true);
  }

  function onSubmit(e) {
    lastTarget = e.target || lastTarget;
    send('submit', lastX, lastY, Date.now() - hoverStart, e.target, {
      input_type: 'form'
    }, true);
  }

  function shouldIncludeContext(eventType, now, forceContext) {
    return !!forceContext || eventType !== 'move' || (now - lastContextSent) >= contextSyncMS;
  }

  function send(eventType, x, y, hoverDurationMS, target, extra, forceContext) {
    if (!sessionId || !baseFetch) return;
    const el = target && target.tagName ? target : null;
    const now = Date.now();
    const payload = Object.assign({
      session_id: sessionId,
      x,
      y,
      hover_duration_ms: hoverDurationMS,
      event_type: eventType,
      window: document.title || 'browser-window',
      url: location.href,
      route: location.pathname,
      target_tag: el ? el.tagName.toLowerCase() : '',
      target_id: el ? (el.id || '') : '',
      target_test_id: el ? ((el.getAttribute && el.getAttribute('data-testid')) || '') : '',
      target_role: el ? ((el.getAttribute && el.getAttribute('role')) || '') : '',
      target_label: elementLabel(el),
      target_selector: el ? cssSelector(el) : '',
      timestamp: new Date().toISOString()
    }, extra || {});
    if (shouldIncludeContext(eventType, now, forceContext)) {
      lastContextSent = now;
      const dom = inspectElement(el || lastTarget);
      if (dom) payload.dom = dom;
      if (recentConsole.length) payload.console = recentConsole.slice(-8);
      if (recentNetwork.length) payload.network = recentNetwork.slice(-8);
    }
    baseFetch(pointerURL, {
      method: 'POST',
      mode: 'cors',
      headers: {
        'Content-Type': 'application/json',
        'X-Knit-Token': controlToken,
        'X-Knit-Nonce': (self.crypto && self.crypto.randomUUID ? self.crypto.randomUUID() : String(Date.now()) + Math.random()),
        'X-Knit-Timestamp': String(Date.now())
      },
      body: JSON.stringify(payload),
      keepalive: true
    }).then((resp) => {
      if (resp.status === 400) {
        maybeResyncSession();
      }
    }).catch(() => {});
  }

  function eventModifiers(evt) {
    if (!evt) return [];
    const modifiers = [];
    if (evt.altKey) modifiers.push('alt');
    if (evt.ctrlKey) modifiers.push('control');
    if (evt.metaKey) modifiers.push('meta');
    if (evt.shiftKey) modifiers.push('shift');
    return modifiers;
  }

  function elementLabel(el) {
    if (!el || !el.tagName) return '';
    return ((el.getAttribute && el.getAttribute('aria-label')) || el.innerText || '').trim().slice(0, 120);
  }

  function elementInputType(el) {
    if (!el || !el.tagName) return '';
    if (el.isContentEditable) return 'contenteditable';
    const tag = String(el.tagName || '').toLowerCase();
    if (tag === 'textarea') return 'textarea';
    if (tag === 'select') return 'select';
    if (tag === 'form') return 'form';
    if (tag === 'input') return String((el.getAttribute && el.getAttribute('type')) || 'text').toLowerCase();
    return tag;
  }

  function classifySensitiveField(el) {
    if (!el || !el.tagName) return '';
    const haystack = [
      el.id || '',
      el.name || '',
      (el.getAttribute && el.getAttribute('autocomplete')) || '',
      (el.getAttribute && el.getAttribute('placeholder')) || '',
      (el.getAttribute && el.getAttribute('aria-label')) || ''
    ].join(' ').toLowerCase();
    const inputType = elementInputType(el);
    if (inputType === 'password' || inputType === 'hidden' || inputType === 'file') return 'password';
    if (/(password|passcode|secret|otp|one[-_ ]?time|mfa|2fa|pin|cvv|cvc|ssn)/.test(haystack)) return 'password';
    if (/(token|credential|auth|api[-_ ]?key|bearer|access[-_ ]?key|secret[-_ ]?key|client[-_ ]?secret)/.test(haystack)) return 'token';
    if (/(card|credit|debit|amex|visa|mastercard|discover|pan)/.test(haystack)) return 'card';
    return '';
  }

  function maskTokenValue(value) {
    const text = String(value || '').trim();
    if (!text) return '';
    if (text.length <= 8) return '[masked token]';
    return text.slice(0, 4) + '...' + text.slice(-4);
  }

  function maskCardValue(value) {
    const raw = String(value || '');
    const digits = raw.replace(/\D/g, '');
    if (!digits) return '';
    const last4 = digits.slice(-4);
    return '**** **** **** ' + last4;
  }

  function inputValuePayload(el) {
    const result = { value: '', valueCaptured: false, valueRedacted: false };
    if (!el || !el.tagName) return result;
    if (!captureInputValues) {
      result.valueRedacted = true;
      return result;
    }
    const sensitivity = classifySensitiveField(el);
    if (sensitivity === 'password') {
      result.valueRedacted = true;
      return result;
    }
    try {
      let value = '';
      if (el.isContentEditable) {
        value = String(el.textContent || '');
      } else if (typeof el.value === 'string') {
        value = el.value;
      } else if (el.tagName && String(el.tagName).toLowerCase() === 'select') {
        value = String(el.value || '');
      }
      value = String(value || '').replace(/\s+/g, ' ').trim();
      if (!value) return result;
      if (sensitivity === 'token') {
        result.value = maskTokenValue(value);
      } else if (sensitivity === 'card') {
        result.value = maskCardValue(value);
      } else {
        result.value = value.slice(0, 400);
      }
      result.valueCaptured = true;
      return result;
    } catch (_) {
      result.valueRedacted = true;
      return result;
    }
  }

  function pushBounded(list, entry) {
    list.push(entry);
    if (list.length > maxContextItems) {
      list.splice(0, list.length - maxContextItems);
    }
  }

  function truncate(value, max) {
    const text = String(value || '').replace(/\s+/g, ' ').trim();
    if (text.length <= max) return text;
    return text.slice(0, Math.max(0, max - 1)) + '…';
  }

  function sanitizeURL(raw) {
    try {
      const parsed = new URL(String(raw || ''), location.href);
      parsed.username = '';
      parsed.password = '';
      parsed.search = '';
      parsed.hash = '';
      return parsed.toString();
    } catch (_) {
      return truncate(raw, 320);
    }
  }

  function formatConsoleArg(value) {
    try {
      if (value instanceof Error) {
        return truncate(value.stack || value.message || String(value), 240);
      }
      if (typeof value === 'string') return truncate(value, 240);
      return truncate(JSON.stringify(value), 240);
    } catch (_) {
      return truncate(String(value), 240);
    }
  }

  function captureConsole(level, args) {
    const message = truncate(args.map(formatConsoleArg).filter(Boolean).join(' '), 400);
    if (!message || message.indexOf('[knit companion]') >= 0) return;
    pushBounded(recentConsole, {
      level: truncate(level, 16),
      message,
      timestamp: new Date().toISOString()
    });
  }

  function patchConsole() {
    ['log', 'info', 'warn', 'error', 'debug'].forEach((level) => {
      if (typeof console[level] !== 'function') return;
      if (originalConsole[level]) return;
      originalConsole[level] = console[level].bind(console);
      console[level] = function (...args) {
        captureConsole(level, args);
        return originalConsole[level](...args);
      };
    });
  }

  function restoreConsole() {
    Object.keys(originalConsole).forEach((level) => {
      console[level] = originalConsole[level];
      delete originalConsole[level];
    });
  }

  function recordNetwork(kind, method, rawURL, status, ok, durationMS) {
    const sanitizedURL = sanitizeURL(rawURL);
    if (isCompanionURL(sanitizedURL)) return;
    pushBounded(recentNetwork, {
      kind: truncate(kind, 16),
      method: truncate((method || 'GET').toUpperCase(), 16),
      url: sanitizedURL,
      status: Number.isFinite(status) ? status : 0,
      ok: !!ok,
      duration_ms: Math.max(0, Math.round(Number(durationMS || 0))),
      timestamp: new Date().toISOString()
    });
  }

  function isCompanionURL(rawURL) {
    const value = String(rawURL || '');
    return value.indexOf(pointerURL) === 0 || value.indexOf(stateURL) === 0;
  }

  async function wrappedFetch(input, init) {
    const startedAt = Date.now();
    const method = init && init.method ? init.method : (input && input.method ? input.method : 'GET');
    const rawURL = typeof input === 'string' ? input : (input && input.url ? input.url : '');
    try {
      const response = await originalFetch(input, init);
      recordNetwork('fetch', method, rawURL, response.status, response.ok, Date.now() - startedAt);
      return response;
    } catch (err) {
      recordNetwork('fetch', method, rawURL, 0, false, Date.now() - startedAt);
      throw err;
    }
  }

  function patchFetch() {
    if (originalFetch && window.fetch !== wrappedFetch) {
      window.fetch = wrappedFetch;
    }
  }

  function restoreFetch() {
    if (originalFetch) {
      window.fetch = originalFetch;
    }
  }

  function wrappedXHROpen(method, url) {
    this.__knitMethod = method;
    this.__knitURL = url;
    return originalXHROpen.apply(this, arguments);
  }

  function wrappedXHRSend() {
    const startedAt = Date.now();
    const xhr = this;
    const finalize = function () {
      const status = Number(xhr.status || 0);
      recordNetwork('xhr', xhr.__knitMethod || 'GET', xhr.__knitURL || '', status, status >= 200 && status < 400, Date.now() - startedAt);
      xhr.removeEventListener('loadend', finalize);
      xhr.removeEventListener('error', finalize);
      xhr.removeEventListener('abort', finalize);
    };
    xhr.addEventListener('loadend', finalize);
    xhr.addEventListener('error', finalize);
    xhr.addEventListener('abort', finalize);
    return originalXHRSend.apply(xhr, arguments);
  }

  function patchXHR() {
    if (XMLHttpRequest.prototype.open !== wrappedXHROpen) {
      XMLHttpRequest.prototype.open = wrappedXHROpen;
      XMLHttpRequest.prototype.send = wrappedXHRSend;
    }
  }

  function restoreXHR() {
    XMLHttpRequest.prototype.open = originalXHROpen;
    XMLHttpRequest.prototype.send = originalXHRSend;
  }

  function inspectElement(el) {
    try {
      if (!el || !el.tagName) return null;
      const attrs = {};
      ['role', 'type', 'name', 'href', 'src', 'placeholder', 'aria-label'].forEach((key) => {
        const value = el.getAttribute && el.getAttribute(key);
        if (value) attrs[key] = truncate(value, 160);
      });
      return {
        tag: truncate(el.tagName.toLowerCase(), 32),
        id: truncate(el.id || '', 120),
        test_id: truncate((el.getAttribute && el.getAttribute('data-testid')) || '', 120),
        role: truncate((el.getAttribute && el.getAttribute('role')) || '', 60),
        label: truncate(elementLabel(el), 200),
        selector: truncate(cssSelector(el), 240),
        text_preview: truncate((el.textContent || '').trim(), 240),
        outer_html: truncate(((el.outerHTML || '').replace(/\s+/g, ' ')).trim(), 400),
        attributes: attrs
      };
    } catch (_) {
      return null;
    }
  }

  function refreshState() {
    fetchStateSummary()
      .then(applyState)
      .catch(() => {});
  }

  function maybeResyncSession() {
    const now = Date.now();
    if (now - lastResyncAttempt < 1000) return;
    lastResyncAttempt = now;
    refreshState();
  }

  function cssSelector(el) {
    try {
      if (!el || !el.tagName) return '';
      if (el.id) return '#' + el.id;
      const parts = [];
      let curr = el;
      let depth = 0;
      while (curr && curr.tagName && depth < 4) {
        let part = curr.tagName.toLowerCase();
        const testID = curr.getAttribute && curr.getAttribute('data-testid');
        if (testID) {
          part += '[data-testid="' + testID + '"]';
          parts.unshift(part);
          break;
        }
        const parent = curr.parentElement;
        if (parent) {
          const siblings = Array.from(parent.children).filter(x => x.tagName === curr.tagName);
          if (siblings.length > 1) {
            const idx = siblings.indexOf(curr) + 1;
            part += ':nth-of-type(' + idx + ')';
          }
        }
        parts.unshift(part);
        curr = curr.parentElement;
        depth += 1;
      }
      return parts.join(' > ');
    } catch (_) {
      return '';
    }
  }

  bootstrap();
})();`
