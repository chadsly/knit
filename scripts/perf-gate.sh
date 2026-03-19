#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tracker_max_ns_op="${KNIT_PERF_TRACKER_NS_OP_MAX:-750000}"
server_pointer_max_ns_op="${KNIT_PERF_SERVER_POINTER_NS_OP_MAX:-25000000}"
server_feedback_max_ns_op="${KNIT_PERF_SERVER_FEEDBACK_NS_OP_MAX:-75000000}"
media_compress_max_ns_op="${KNIT_PERF_MEDIA_COMPRESS_NS_OP_MAX:-20000000}"
out_dir="${KNIT_PERF_OUT_DIR:-./dist/perf}"

cd "$ROOT_DIR"
mkdir -p "$out_dir"

extract_ns_op() {
  local bench_name="$1"
  local bench_text="$2"
  printf '%s\n' "$bench_text" | awk -v name="$bench_name" '
    $1 ~ name {
      for (i=1; i<=NF; i++) {
        if ($i ~ /ns\/op$/) {
          print $(i-1);
          exit;
        }
      }
    }'
}

assert_ns_budget() {
  local label="$1"
  local value="$2"
  local max="$3"
  if [[ -z "${value:-}" ]]; then
    echo "[perf-gate] failed to parse benchmark ns/op output for ${label}"
    exit 1
  fi
  if awk -v value="$value" -v max="$max" 'BEGIN { exit !(value > max) }'; then
    echo "[perf-gate] benchmark regression (${label}): ${value} ns/op exceeds max ${max} ns/op"
    exit 1
  fi
  echo "[perf-gate] ${label} passed (${value} ns/op <= ${max} ns/op)"
}

echo "[perf-gate] running tracker benchmark"
tracker_out="$(go test -run '^$' -bench 'BenchmarkTrackerAddAndSnapshot$' -benchmem -count=1 -cpuprofile "$out_dir/tracker.cpuprofile" ./internal/companion)"
echo "$tracker_out"
tracker_ns_op="$(extract_ns_op 'BenchmarkTrackerAddAndSnapshot' "$tracker_out")"
assert_ns_budget "tracker" "$tracker_ns_op" "$tracker_max_ns_op"

echo "[perf-gate] running server capture-path benchmarks"
server_out="$(go test -run '^$' -bench 'BenchmarkServerPointerIngest$|BenchmarkServerFeedbackCapture$' -benchmem -count=1 -cpuprofile "$out_dir/server.cpuprofile" ./internal/server)"
echo "$server_out"
server_pointer_ns_op="$(extract_ns_op 'BenchmarkServerPointerIngest' "$server_out")"
server_feedback_ns_op="$(extract_ns_op 'BenchmarkServerFeedbackCapture' "$server_out")"
assert_ns_budget "server_pointer_ingest" "$server_pointer_ns_op" "$server_pointer_max_ns_op"
assert_ns_budget "server_feedback_capture" "$server_feedback_ns_op" "$server_feedback_max_ns_op"

echo "[perf-gate] running media compression benchmark"
media_out="$(go test -run '^$' -bench 'BenchmarkCompressScreenshot$' -benchmem -count=1 -cpuprofile "$out_dir/media.cpuprofile" ./internal/server)"
echo "$media_out"
media_ns_op="$(extract_ns_op 'BenchmarkCompressScreenshot' "$media_out")"
assert_ns_budget "media_compress" "$media_ns_op" "$media_compress_max_ns_op"

cat > "$out_dir/summary.json" <<EOF
{
  "tracker_ns_op": ${tracker_ns_op},
  "server_pointer_ns_op": ${server_pointer_ns_op},
  "server_feedback_ns_op": ${server_feedback_ns_op},
  "media_compress_ns_op": ${media_ns_op}
}
EOF

echo "[perf-gate] passed"
