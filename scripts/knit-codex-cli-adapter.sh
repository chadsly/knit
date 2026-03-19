#!/usr/bin/env bash
set -euo pipefail

# Reads a Knit CLI payload JSON from stdin and invokes local Codex CLI.
# Emits a compact JSON result that Knit's CLI adapter can parse.

work_dir="${KNIT_CODEX_WORKDIR:-$(pwd)}"
run_id="codex-cli-$(date +%s)"
skip_git_repo_check="${KNIT_CODEX_SKIP_GIT_REPO_CHECK:-1}"
sandbox_mode="${KNIT_CODEX_SANDBOX:-}"
approval_policy="${KNIT_CODEX_APPROVAL_POLICY:-}"
profile_name="${KNIT_CODEX_PROFILE:-}"
model_name="${KNIT_CODEX_MODEL:-}"
reasoning_effort="${KNIT_CODEX_REASONING_EFFORT:-}"

payload_file="$(mktemp)"
prompt_file="$(mktemp)"
output_dir="${KNIT_CODEX_OUTPUT_DIR:-${TMPDIR:-/tmp}}"
mkdir -p "$output_dir"
if [[ -n "${KNIT_CLI_LOG_FILE:-}" ]]; then
  log_file="${KNIT_CLI_LOG_FILE}"
  mkdir -p "$(dirname "$log_file")"
else
  log_file="$(mktemp "$output_dir/knit-codex-${run_id}-XXXX.log")"
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

cmd=(codex)
if [[ -n "$approval_policy" ]]; then
  cmd+=(-a "$approval_policy")
fi
if [[ -n "$profile_name" ]]; then
  cmd+=(-p "$profile_name")
fi
if [[ -n "$model_name" ]]; then
  cmd+=(-m "$model_name")
fi
if [[ -n "$reasoning_effort" ]]; then
  cmd+=(-c "model_reasoning_effort=\"$reasoning_effort\"")
fi
cmd+=(exec -C "$work_dir")
if [[ "$skip_git_repo_check" == "1" || "$skip_git_repo_check" == "true" ]]; then
  cmd+=(--skip-git-repo-check)
fi
if [[ -n "$sandbox_mode" ]]; then
  cmd+=(--sandbox "$sandbox_mode")
fi
cmd+=(-)

if [[ -n "${KNIT_CLI_LOG_FILE:-}" ]]; then
  run_codex() {
    "${cmd[@]}" < "$prompt_file" 1>&2
  }
else
  run_codex() {
    "${cmd[@]}" < "$prompt_file" > "$log_file" 2>&1
  }
fi

if run_codex; then
  printf '{"run_id":"%s","status":"accepted","ref":"%s"}\n' "$run_id" "$log_file"
else
  status=$?
  echo "codex exec failed (exit $status). See log: $log_file" >&2
  if [[ -z "${KNIT_CLI_LOG_FILE:-}" ]]; then
    tail -n 20 "$log_file" >&2 || true
  fi
  printf '{"run_id":"%s","status":"failed","ref":"%s"}\n' "$run_id" "$log_file"
  exit "$status"
fi
