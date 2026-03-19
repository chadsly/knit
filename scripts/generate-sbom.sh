#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_FILE="${1:-./dist/sbom.spdx.json}"
OUT_DIR="$(dirname "$OUT_FILE")"
mkdir -p "$OUT_DIR"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-0}"

cd "$ROOT_DIR"

if command -v syft >/dev/null 2>&1; then
  syft "dir:." -o "spdx-json=$OUT_FILE"
  echo "SBOM generated with syft: $OUT_FILE"
  exit 0
fi

{
  echo "{"
  echo "  \"spdxVersion\": \"SPDX-2.3\","
  echo "  \"name\": \"knit-fallback-sbom\","
  echo "  \"documentNamespace\": \"https://knit.local/sbom/${SOURCE_DATE_EPOCH}\","
  echo "  \"creationInfo\": {"
  echo "    \"created\": \"${SOURCE_DATE_EPOCH}\","
  echo "    \"creators\": [\"Tool: scripts/generate-sbom.sh (fallback)\"]"
  echo "  },"
  echo "  \"comment\": \"Fallback SBOM generated from go module list because syft is unavailable.\","
  echo "  \"packages\": ["
  first=1
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    if [[ $first -eq 0 ]]; then
      echo ","
    fi
    first=0
    escaped="${line//\"/\\\"}"
    echo -n "    {\"name\":\"$escaped\"}"
  done < <(go list -m all)
  echo
  echo "  ]"
  echo "}"
} >"$OUT_FILE"

echo "Fallback SBOM metadata generated: $OUT_FILE"
