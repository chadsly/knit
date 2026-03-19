#!/usr/bin/env bash
set -euo pipefail

# Reads a Knit CLI payload JSON from stdin and invokes local Claude CLI.
# Emits a compact JSON result that Knit's CLI adapter can parse.

work_dir="${KNIT_CODEX_WORKDIR:-$(pwd)}"
run_id="claude-cli-$(date +%s)"
payload_file="$(mktemp)"
prompt_file="$(mktemp)"
output_dir="${KNIT_CODEX_OUTPUT_DIR:-${TMPDIR:-/tmp}}"
mkdir -p "$output_dir"
if [[ -n "${KNIT_CLI_LOG_FILE:-}" ]]; then
  log_file="${KNIT_CLI_LOG_FILE}"
  mkdir -p "$(dirname "$log_file")"
else
  log_file="$(mktemp "$output_dir/knit-claude-${run_id}-XXXX.log")"
fi

cleanup() {
  rm -f "$payload_file" "$prompt_file"
}
trap cleanup EXIT

cat > "$payload_file"

python3 - "$payload_file" "$prompt_file" <<'PY'
import json
import pathlib
import sys

payload_path = pathlib.Path(sys.argv[1])
prompt_path = pathlib.Path(sys.argv[2])
payload = json.loads(payload_path.read_text())
instruction = str(payload.get("instruction_text") or "").strip()
prompt = (
    "You are receiving a Knit CLI payload JSON.\n"
    "Follow the instruction_text exactly and use the package field as the approved source of truth.\n"
    "Do not ignore the selected delivery intent.\n\n"
    f"instruction_text:\n{instruction}\n\n"
    "Knit CLI payload JSON:\n"
    f"{json.dumps(payload, ensure_ascii=True)}\n"
)
prompt_path.write_text(prompt)
PY

run_claude() {
  cd "$work_dir"
  claude -p < "$prompt_file"
}

if [[ -n "${KNIT_CLI_LOG_FILE:-}" ]]; then
  if run_claude 1>&2; then
    printf '{"run_id":"%s","status":"accepted","ref":"%s"}\n' "$run_id" "$log_file"
  else
    status=$?
    echo "claude cli failed (exit $status). See log: $log_file" >&2
    printf '{"run_id":"%s","status":"failed","ref":"%s"}\n' "$run_id" "$log_file"
    exit "$status"
  fi
else
  if run_claude > "$log_file" 2>&1; then
    printf '{"run_id":"%s","status":"accepted","ref":"%s"}\n' "$run_id" "$log_file"
  else
    status=$?
    echo "claude cli failed (exit $status). See log: $log_file" >&2
    tail -n 20 "$log_file" >&2 || true
    printf '{"run_id":"%s","status":"failed","ref":"%s"}\n' "$run_id" "$log_file"
    exit "$status"
  fi
fi
