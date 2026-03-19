#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_FILE="${1:-./dist/dependency-scan.json}"
OUT_DIR="$(dirname "$OUT_FILE")"
mkdir -p "$OUT_DIR"

cd "$ROOT_DIR"

if command -v govulncheck >/dev/null 2>&1; then
  govulncheck -json ./... > "$OUT_FILE"
  echo "dependency scan generated with govulncheck: $OUT_FILE"
  exit 0
fi

echo "govulncheck not found; running go list as baseline dependency inventory check."
go list ./... >/dev/null
cat > "$OUT_FILE" <<EOF
{
  "scanner": "fallback",
  "status": "go-list-only",
  "comment": "govulncheck unavailable; module/package inventory resolved successfully as baseline validation."
}
EOF
echo "fallback dependency scan report generated: $OUT_FILE"
