#!/usr/bin/env bash
set -euo pipefail

public_key="${1:?usage: verify-update-signature.sh <public-key> <checksums.txt> <checksums.sig> [artifact-path]}"
checksums_file="${2:?usage: verify-update-signature.sh <public-key> <checksums.txt> <checksums.sig> [artifact-path]}"
sig_file="${3:?usage: verify-update-signature.sh <public-key> <checksums.txt> <checksums.sig> [artifact-path]}"
artifact_path="${4:-}"

openssl dgst -sha256 -verify "$public_key" -signature "$sig_file" "$checksums_file"

if [[ -n "$artifact_path" ]]; then
  if command -v shasum >/dev/null 2>&1; then
    actual_hash="$(shasum -a 256 "$artifact_path" | awk '{print $1}')"
  else
    actual_hash="$(sha256sum "$artifact_path" | awk '{print $1}')"
  fi
  artifact_base="$(basename "$artifact_path")"
  actual_hash_lower="$(printf '%s' "$actual_hash" | tr '[:upper:]' '[:lower:]')"
  if ! awk -v hash="$actual_hash_lower" -v target="$artifact_base" '
    {
      file=$NF
      sub(/^\*/, "", file)
      n=split(file, parts, "/")
      if (tolower($1) == hash && parts[n] == target) {
        found=1
      }
    }
    END { exit found ? 0 : 1 }' "$checksums_file"; then
    echo "[verify-update-signature] artifact hash not found in checksums for $artifact_base"
    exit 1
  fi
fi

echo "[verify-update-signature] passed"
