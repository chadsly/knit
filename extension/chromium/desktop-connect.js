// One explicit handoff, reusing the daemon's short-lived pairing flow.
globalThis.KnitDesktopConnect = {
  parse(raw) {
    if (typeof raw !== "string" || raw.length > 12000 || !raw.startsWith("knit-connect:")) throw Error("Paste the connection code from Connect browser in Codex.");
    let data;
    try { data = JSON.parse(decodeURIComponent(raw.slice(13))); } catch { throw Error("This connection code is damaged. Copy a new one from Codex."); }
    const url = new URL(data.daemon_url);
    if (data.version !== 1 || url.protocol !== "http:" || !["127.0.0.1", "localhost"].includes(url.hostname) || url.username || url.password || url.search || url.hash || url.pathname !== "/") throw Error("Only a local Knit connection is allowed.");
    if (!/^[A-Z0-9]{4,16}$/.test(data.pairing_code || "") || typeof data.binding_id !== "string" || !data.binding_id || data.binding_id.length > 100) throw Error("This connection code is incomplete.");
    if (!Number.isFinite(Date.parse(data.expires_at)) || Date.parse(data.expires_at) <= Date.now()) throw Error("This code expired. Click Connect browser in Codex for a new one.");
    return {...data, daemon_url:url.origin};
  },
  async connect(raw, fetcher = fetch) {
    const data = this.parse(raw);
    async function request(path, token, body) {
      const headers = {};
      if (token) headers.Authorization = "Bearer " + token;
      if (body) {
        headers["Content-Type"] = "application/json";
        headers["X-Knit-Nonce"] = crypto.randomUUID();
        headers["X-Knit-Timestamp"] = String(Date.now());
      }
      const response = await fetcher(data.daemon_url + path,{method:body ? "POST" : "GET",headers,body:body ? JSON.stringify(body) : undefined,redirect:"error",signal:AbortSignal.timeout(10000)});
      if (!response.ok) throw Error(response.status === 401 ? "Pairing was rejected. Get a fresh connection code from Codex." : "Knit could not complete the connection. Your previous connection is unchanged.");
      return response.json();
    }
    const paired = await request("/api/extension/pair/complete",null,{pairing_code:data.pairing_code,name:"Codex browser feedback",browser:"chromium"});
    if (!paired.token) throw Error("Knit did not return a browser credential.");
    let state = await request("/api/extension/session",paired.token);
    const binding = state.desktop?.bindings?.find(item => item.id === data.binding_id);
    if (!binding) throw Error("This conversation is disconnected. Reconnect it in Codex.");
    if (!state.session || state.session.status === "stopped") {
      await request("/api/session/start",paired.token,{target_window:binding.label});
      state = await request("/api/extension/session",paired.token);
    }
    return {daemonBaseURL:data.daemon_url,daemonToken:paired.token,desktopDestinationID:binding.id,label:binding.label};
  }
};
