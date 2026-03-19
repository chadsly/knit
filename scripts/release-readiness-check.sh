#!/usr/bin/env bash
set -euo pipefail

pkg_dir="${1:-./dist-ci/packages}"
checksums_file="$pkg_dir/checksums.txt"
sig_file="$pkg_dir/checksums.sig"
build_manifest="$pkg_dir/build-manifest.json"
release_manifest="$pkg_dir/release-manifest.json"
release_manifest_sig="$pkg_dir/release-manifest.sig"
sbom_file="$pkg_dir/sbom.spdx.json"
dependency_scan_file="$pkg_dir/dependency-scan.json"

hash_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    sha256sum "$file" | awk '{print $1}'
  fi
}

if [[ ! -f "$checksums_file" ]]; then
  echo "[release-readiness] missing checksums file: $checksums_file"
  exit 1
fi
if [[ ! -f "$build_manifest" ]]; then
  echo "[release-readiness] missing build manifest: $build_manifest"
  exit 1
fi
if [[ ! -f "$release_manifest" ]]; then
  echo "[release-readiness] missing release manifest: $release_manifest"
  exit 1
fi

echo "[release-readiness] verifying packaged artifact list"
while read -r sum file; do
  [[ -z "${sum:-}" ]] && continue
  [[ -z "${file:-}" ]] && continue
  if [[ "$file" = /* ]]; then
    artifact_path="$file"
  else
    artifact_path="$pkg_dir/$file"
  fi
  if [[ ! -f "$artifact_path" ]]; then
    echo "[release-readiness] missing artifact referenced by checksums: $artifact_path"
    exit 1
  fi
  actual_hash="$(hash_file "$artifact_path")"
  if [[ "$(printf '%s' "$actual_hash" | tr '[:upper:]' '[:lower:]')" != "$(printf '%s' "$sum" | tr '[:upper:]' '[:lower:]')" ]]; then
    echo "[release-readiness] checksum mismatch for $artifact_path"
    exit 1
  fi
done < "$checksums_file"

if [[ -n "${CI_COMMIT_TAG:-}" ]]; then
  if [[ ! -f "$sig_file" ]]; then
    echo "[release-readiness] tagged build requires signed checksums ($sig_file)"
    exit 1
  fi
  if [[ ! -f "$release_manifest_sig" ]]; then
    echo "[release-readiness] tagged build requires signed release manifest ($release_manifest_sig)"
    exit 1
  fi
  if [[ ! -f "$sbom_file" ]]; then
    echo "[release-readiness] tagged build requires sbom file ($sbom_file)"
    exit 1
  fi
  if [[ ! -f "$dependency_scan_file" ]]; then
    echo "[release-readiness] tagged build requires dependency scan report ($dependency_scan_file)"
    exit 1
  fi
fi

if [[ -f "$sig_file" ]]; then
  if [[ -z "${RELEASE_SIGNING_PUBLIC_KEY:-}" ]]; then
    echo "[release-readiness] signature exists but RELEASE_SIGNING_PUBLIC_KEY is not set"
    exit 1
  fi
  openssl dgst -sha256 -verify "$RELEASE_SIGNING_PUBLIC_KEY" -signature "$sig_file" "$checksums_file"
fi
if [[ -f "$release_manifest_sig" ]]; then
  if [[ -z "${RELEASE_SIGNING_PUBLIC_KEY:-}" ]]; then
    echo "[release-readiness] release manifest signature exists but RELEASE_SIGNING_PUBLIC_KEY is not set"
    exit 1
  fi
  openssl dgst -sha256 -verify "$RELEASE_SIGNING_PUBLIC_KEY" -signature "$release_manifest_sig" "$release_manifest"
fi

echo "[release-readiness] passed"
