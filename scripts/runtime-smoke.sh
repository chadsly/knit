#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${1:?usage: runtime-smoke.sh <dist-dir>}"
DIST_DIR="$(cd "$DIST_DIR" && pwd)"
HOST_TARGET="$(go env GOOS)_$(go env GOARCH)"
BIN_DIR="$DIST_DIR/bin/$HOST_TARGET"
DAEMON_BIN="$BIN_DIR/daemon"
UI_BIN="$BIN_DIR/ui"

if [[ "$(go env GOOS)" == "windows" ]]; then
  DAEMON_BIN="${DAEMON_BIN}.exe"
  UI_BIN="${UI_BIN}.exe"
fi

if [[ ! -x "$DAEMON_BIN" ]]; then
  echo "[runtime-smoke] missing daemon binary: $DAEMON_BIN"
  exit 1
fi
if [[ ! -x "$UI_BIN" ]]; then
  echo "[runtime-smoke] missing ui binary: $UI_BIN"
  exit 1
fi

ADDR="${KNIT_SMOKE_ADDR:-127.0.0.1:17777}"
TOKEN="smoke-token"
DATA_DIR="$(mktemp -d "${TMPDIR:-/tmp}/knit-smoke.XXXXXX")"
LOG_FILE="$DATA_DIR/daemon.log"
CONFIG_PATH="$DATA_DIR/knit.toml"
ENCRYPTION_KEY_B64="$(python3 - <<'PY'
import base64
print(base64.b64encode(b"\x11" * 32).decode())
PY
)"

cleanup() {
  if [[ -n "${daemon_pid:-}" ]]; then
    kill "$daemon_pid" 2>/dev/null || true
    wait "$daemon_pid" 2>/dev/null || true
  fi
  rm -rf "$DATA_DIR"
}
trap cleanup EXIT

KNIT_ADDR="$ADDR" \
KNIT_CONTROL_TOKEN="$TOKEN" \
KNIT_DATA_DIR="$DATA_DIR" \
KNIT_SQLITE_PATH="$DATA_DIR/knit.db" \
KNIT_CONFIG_PATH="$CONFIG_PATH" \
KNIT_ENCRYPTION_KEY_B64="$ENCRYPTION_KEY_B64" \
"$DAEMON_BIN" >"$LOG_FILE" 2>&1 &
daemon_pid=$!

health_url="http://$ADDR/healthz"
state_url="http://$ADDR/api/state"
floating_url="http://$ADDR/floating-composer"
root_url="http://$ADDR/"

json_post() {
  local url="$1"
  local payload="$2"
  curl -fsS -H "X-Knit-Token: $TOKEN" -H "Content-Type: application/json" -H "X-Knit-Nonce: smoke-$RANDOM-$RANDOM" -H "X-Knit-Timestamp: $(python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
)" -d "$payload" "$url"
}

for _ in $(seq 1 50); do
  if curl -fsS "$health_url" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done

health_payload="$(curl -fsS "$health_url")"
state_payload="$(curl -fsS -H "X-Knit-Token: $TOKEN" "$state_url")"
root_payload="$(curl -fsS "$root_url")"
floating_status="$(curl -s -o /dev/null -w '%{http_code}' "$floating_url")"
floating_auth_status="$(curl -s -o "$DATA_DIR/floating.html" -w '%{http_code}' -H "X-Knit-Token: $TOKEN" "$floating_url")"
floating_payload="$(cat "$DATA_DIR/floating.html")"
ui_output="$("$UI_BIN")"

start_payload="$(json_post "http://$ADDR/api/session/start" '{"target_window":"Browser Preview","target_url":"https://example.com/app"}')"
pause_payload="$(json_post "http://$ADDR/api/session/pause" '{}')"
paused_state="$(curl -fsS -H "X-Knit-Token: $TOKEN" "$state_url")"
resume_payload="$(json_post "http://$ADDR/api/session/resume" '{}')"
feedback_payload="$(json_post "http://$ADDR/api/session/feedback" '{"raw_transcript":"Smoke note","normalized":"Smoke note","pointer_x":12,"pointer_y":16,"window":"Browser Preview"}')"
approve_payload="$(json_post "http://$ADDR/api/session/approve" '{"summary":"Smoke summary"}')"
preview_payload="$(json_post "http://$ADDR/api/session/payload/preview" '{"provider":"cli"}')"
submit_payload="$(json_post "http://$ADDR/api/session/submit" '{"provider":"cli"}')"

attempt_id="$(python3 - "$submit_payload" <<'PY'
import json
import sys
print(json.loads(sys.argv[1])["attempt_id"])
PY
)"

attempt_status=""
for _ in $(seq 1 50); do
  polled_state="$(curl -fsS -H "X-Knit-Token: $TOKEN" "$state_url")"
  attempt_status="$(python3 - "$polled_state" "$attempt_id" <<'PY'
import json
import sys
state = json.loads(sys.argv[1])
attempt_id = sys.argv[2]
for attempt in state.get("submit_attempts", []):
    if attempt.get("attempt_id") == attempt_id:
        print(attempt.get("status", ""))
        break
else:
    print("")
PY
)"
  if [[ "$attempt_status" == "submitted" ]]; then
    break
  fi
  sleep 0.2
done

stop_payload="$(json_post "http://$ADDR/api/session/stop" '{}')"
final_state="$(curl -fsS -H "X-Knit-Token: $TOKEN" "$state_url")"
history_payload="$(curl -fsS -H "X-Knit-Token: $TOKEN" "http://$ADDR/api/session/history")"

health_file="$DATA_DIR/health.json"
state_file="$DATA_DIR/state.json"
root_file="$DATA_DIR/root.html"
ui_file="$DATA_DIR/ui.txt"
floating_status_file="$DATA_DIR/floating_status.txt"
floating_auth_status_file="$DATA_DIR/floating_auth_status.txt"
floating_file="$DATA_DIR/floating.html"
start_file="$DATA_DIR/start.json"
pause_file="$DATA_DIR/pause.json"
paused_state_file="$DATA_DIR/paused_state.json"
resume_file="$DATA_DIR/resume.json"
feedback_file="$DATA_DIR/feedback.json"
approve_file="$DATA_DIR/approve.json"
preview_file="$DATA_DIR/preview.json"
submit_file="$DATA_DIR/submit.json"
attempt_status_file="$DATA_DIR/attempt_status.txt"
stop_file="$DATA_DIR/stop.json"
final_state_file="$DATA_DIR/final_state.json"
history_file="$DATA_DIR/history.json"

printf '%s' "$health_payload" > "$health_file"
printf '%s' "$state_payload" > "$state_file"
printf '%s' "$root_payload" > "$root_file"
printf '%s' "$ui_output" > "$ui_file"
printf '%s' "$floating_status" > "$floating_status_file"
printf '%s' "$floating_auth_status" > "$floating_auth_status_file"
printf '%s' "$floating_payload" > "$floating_file"
printf '%s' "$start_payload" > "$start_file"
printf '%s' "$pause_payload" > "$pause_file"
printf '%s' "$paused_state" > "$paused_state_file"
printf '%s' "$resume_payload" > "$resume_file"
printf '%s' "$feedback_payload" > "$feedback_file"
printf '%s' "$approve_payload" > "$approve_file"
printf '%s' "$preview_payload" > "$preview_file"
printf '%s' "$submit_payload" > "$submit_file"
printf '%s' "$attempt_status" > "$attempt_status_file"
printf '%s' "$stop_payload" > "$stop_file"
printf '%s' "$final_state" > "$final_state_file"
printf '%s' "$history_payload" > "$history_file"

python3 - "$HOST_TARGET" "$ADDR" "$health_file" "$state_file" "$root_file" "$ui_file" "$floating_status_file" "$floating_auth_status_file" "$floating_file" "$start_file" "$pause_file" "$paused_state_file" "$resume_file" "$feedback_file" "$approve_file" "$preview_file" "$submit_file" "$attempt_status_file" "$stop_file" "$final_state_file" "$history_file" <<'PY'
import json
import sys

(
    host_target,
    addr,
    health_path,
    state_path,
    root_path,
    ui_path,
    floating_status_path,
    floating_auth_status_path,
    floating_path,
    start_path,
    pause_path,
    paused_state_path,
    resume_path,
    feedback_path,
    approve_path,
    preview_path,
    submit_path,
    attempt_status_path,
    stop_path,
    final_state_path,
    history_path,
) = sys.argv[1:]
health = json.loads(open(health_path, encoding="utf-8").read())
state = json.loads(open(state_path, encoding="utf-8").read())
root_raw = open(root_path, encoding="utf-8").read()
ui_output = open(ui_path, encoding="utf-8").read()
floating_status = open(floating_status_path, encoding="utf-8").read()
floating_auth_status = open(floating_auth_status_path, encoding="utf-8").read()
floating_payload = open(floating_path, encoding="utf-8").read()
start = json.loads(open(start_path, encoding="utf-8").read())
pause = json.loads(open(pause_path, encoding="utf-8").read())
paused_state = json.loads(open(paused_state_path, encoding="utf-8").read())
resume = json.loads(open(resume_path, encoding="utf-8").read())
feedback = json.loads(open(feedback_path, encoding="utf-8").read())
approve = json.loads(open(approve_path, encoding="utf-8").read())
preview = json.loads(open(preview_path, encoding="utf-8").read())
submit = json.loads(open(submit_path, encoding="utf-8").read())
attempt_status = open(attempt_status_path, encoding="utf-8").read()
stop = json.loads(open(stop_path, encoding="utf-8").read())
final_state = json.loads(open(final_state_path, encoding="utf-8").read())
history = json.loads(open(history_path, encoding="utf-8").read())

assert health["ok"] is True
assert state["runtime_platform"]["host_target"] == host_target
assert state["runtime_platform"]["supported"] is True
assert state["platform_profile"]["supported"] is True
assert "runtime_summary" in state["runtime_platform"]
assert "Knit Daemon" in root_raw
assert floating_status == "401"
assert floating_auth_status == "200"
assert "Compact Composer" in floating_payload
assert f"http://{addr}" in ui_output or "http://127.0.0.1:7777" in ui_output
assert start["id"]
assert pause["status"] == "paused"
assert paused_state["capture_state"] == "paused"
assert resume["status"] == "active"
assert feedback["feedback"]
assert approve["summary"] == "Smoke summary"
assert preview["provider"] == "codex_cli"
assert submit["status"] in ("queued", "running", "submitted")
assert attempt_status in ("queued", "in_progress", "submitted")
assert stop["status"] == "stopped"
assert final_state["capture_state"] == "inactive"
assert history and history[0]["status"] in ("submitted", "stopped")
PY

echo "[runtime-smoke] passed for $HOST_TARGET"
