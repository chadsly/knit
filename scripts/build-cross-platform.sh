#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/dist}"
VERSION="${VERSION:-dev}"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-0}"

cd "$ROOT_DIR"

mkdir -p "$OUT_DIR/bin"

targets=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

binaries=(
  "daemon:./cmd/daemon"
  "ui:./cmd/ui"
)

hash_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file"
  else
    sha256sum "$file"
  fi
}

echo "Building cross-platform binaries into: $OUT_DIR"
for target in "${targets[@]}"; do
  IFS=/ read -r goos goarch <<< "$target"
  target_dir="$OUT_DIR/bin/${goos}_${goarch}"
  mkdir -p "$target_dir"

  for entry in "${binaries[@]}"; do
    name="${entry%%:*}"
    pkg="${entry##*:}"
    ext=""
    if [[ "$goos" == "windows" ]]; then
      ext=".exe"
    fi
    out="$target_dir/${name}${ext}"
    echo " - $goos/$goarch -> $name"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-buildid= -s -w" -o "$out" "$pkg"
  done
done

if [[ "${INCLUDE_TRAY_LOCAL:-0}" == "1" ]]; then
  host_target="$(go env GOOS)_$(go env GOARCH)"
  mkdir -p "$OUT_DIR/bin/$host_target"
  tray_out="$OUT_DIR/bin/$host_target/tray"
  if [[ "$(go env GOOS)" == "windows" ]]; then
    tray_out="${tray_out}.exe"
  fi
  echo " - $(go env GOOS)/$(go env GOARCH) -> tray (local only, CGO-enabled)"
  CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags="-buildid= -s -w" -o "$tray_out" ./cmd/tray
fi

checksum_file="$OUT_DIR/checksums.txt"
rm -f "$checksum_file"
while IFS= read -r -d '' file; do
  hash_file "$file" >> "$checksum_file"
done < <(find "$OUT_DIR/bin" -type f -print0 | sort -z)

manifest_file="$OUT_DIR/build-manifest.json"
{
  echo "{"
  echo "  \"version\": \"${VERSION}\","
  echo "  \"source_date_epoch\": \"${SOURCE_DATE_EPOCH}\","
  echo "  \"go_version\": \"$(go version | sed 's/"/\\"/g')\","
  echo "  \"targets\": ["
  first_target=1
  for target in "${targets[@]}"; do
    if [[ $first_target -eq 0 ]]; then
      echo ","
    fi
    first_target=0
    printf '    "%s"' "$target"
  done
  echo
  echo "  ],"
  echo "  \"artifacts\": ["
  first_artifact=1
  while read -r sum file; do
    [[ -z "${sum:-}" ]] && continue
    [[ -z "${file:-}" ]] && continue
    if [[ $first_artifact -eq 0 ]]; then
      echo ","
    fi
    first_artifact=0
    rel="${file#$OUT_DIR/}"
    printf '    {"sha256":"%s","path":"%s"}' "$sum" "$rel"
  done < "$checksum_file"
  echo
  echo "  ]"
  echo "}"
} > "$manifest_file"

echo "Build complete. Version: $VERSION"
echo "Checksums: $checksum_file"
echo "Build manifest: $manifest_file"
