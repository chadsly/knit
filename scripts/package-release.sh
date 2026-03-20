#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${1:-$ROOT_DIR/dist}"
DIST_DIR="$(cd "$DIST_DIR" && pwd)"
PKG_DIR="$DIST_DIR/packages"
NPM_TEMPLATE_DIR="$ROOT_DIR/packaging/npm/knit-daemon"
PYTHON_TEMPLATE_DIR="$ROOT_DIR/packaging/python/knit"
VERSION="${VERSION:-dev}"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-0}"

normalize_python_package_version() {
  local version="$1"
  if [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    printf '%s' "$version"
    return
  fi
  if [[ "$version" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-ci\.([0-9]+)\.([0-9]+)$ ]]; then
    printf '%s.dev%s' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
    return
  fi
  if [[ "$version" =~ ^([0-9]+\.[0-9]+\.[0-9]+)-.*$ ]]; then
    printf '%s.dev0' "${BASH_REMATCH[1]}"
    return
  fi
  printf '%s' "$version"
}

PYTHON_PACKAGE_VERSION="$(normalize_python_package_version "$VERSION")"

mkdir -p "$PKG_DIR"
INSTALLER_DIR="$PKG_DIR/installers"
mkdir -p "$INSTALLER_DIR"

hash_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file"
  else
    sha256sum "$file"
  fi
}

write_checksums() {
  local checksums_path="$1"
  rm -f "$checksums_path"
  while IFS= read -r -d '' pkg_file; do
    hash_file "$pkg_file" >> "$checksums_path"
  done < <(find "$PKG_DIR" -type f \( -name "*.tar.gz" -o -name "*.zip" -o -name "*.tgz" -o -name "*.whl" -o -name "*.install.sh" -o -name "*.install.command" -o -name "*.install.ps1" \) -print0 | sort -z)
  echo " - created $checksums_path"
}

write_release_manifest() {
  local manifest_path="$1"
  local checksums_path="$2"
  {
    echo "{"
    echo "  \"version\": \"${VERSION}\","
    echo "  \"source_date_epoch\": \"${SOURCE_DATE_EPOCH}\","
    echo "  \"checksums_file\": \"checksums.txt\","
    echo "  \"signed_checksums\": $( [[ -f "$PKG_DIR/checksums.sig" ]] && echo "true" || echo "false" ),"
    echo "  \"has_build_manifest\": $( [[ -f "$PKG_DIR/build-manifest.json" ]] && echo "true" || echo "false" ),"
    echo "  \"has_sbom\": $( [[ -f "$PKG_DIR/sbom.spdx.json" ]] && echo "true" || echo "false" ),"
    echo "  \"has_dependency_scan\": $( [[ -f "$PKG_DIR/dependency-scan.json" ]] && echo "true" || echo "false" ),"
    echo "  \"artifacts\": ["
    first_artifact=1
    while read -r sum file; do
      [[ -z "${sum:-}" ]] && continue
      [[ -z "${file:-}" ]] && continue
      if [[ $first_artifact -eq 0 ]]; then
        echo ","
      fi
      first_artifact=0
      rel="${file#$PKG_DIR/}"
      printf '    {"sha256":"%s","path":"%s"}' "$sum" "$rel"
    done < "$checksums_path"
    echo
    echo "  ]"
    echo "}"
  } > "$manifest_path"
  echo " - created $manifest_path"
}

refresh_packaging_metadata() {
  local checksums_path="$1"
  local manifest_path="$2"
  rm -f "$PKG_DIR/checksums.sig" "$PKG_DIR/release-manifest.sig"
  write_checksums "$checksums_path"
  if [[ -n "${RELEASE_SIGNING_PRIVATE_KEY:-}" ]]; then
    echo "Re-signing package checksums after package generation."
    local sig_file="$PKG_DIR/checksums.sig"
    openssl dgst -sha256 -sign "$RELEASE_SIGNING_PRIVATE_KEY" -out "$sig_file" "$checksums_path"
    echo " - created $sig_file"
    if [[ -n "${RELEASE_SIGNING_PUBLIC_KEY:-}" ]]; then
      openssl dgst -sha256 -verify "$RELEASE_SIGNING_PUBLIC_KEY" -signature "$sig_file" "$checksums_path"
      echo " - verified signature with public key"
    fi
  fi
  write_release_manifest "$manifest_path" "$checksums_path"
  if [[ -n "${RELEASE_SIGNING_PRIVATE_KEY:-}" ]]; then
    local manifest_sig="$PKG_DIR/release-manifest.sig"
    openssl dgst -sha256 -sign "$RELEASE_SIGNING_PRIVATE_KEY" -out "$manifest_sig" "$manifest_path"
    echo " - created $manifest_sig"
    if [[ -n "${RELEASE_SIGNING_PUBLIC_KEY:-}" ]]; then
      openssl dgst -sha256 -verify "$RELEASE_SIGNING_PUBLIC_KEY" -signature "$manifest_sig" "$manifest_path"
      echo " - verified release manifest signature with public key"
    fi
  fi
}

echo "Packaging release artifacts from: $DIST_DIR/bin"
if [[ ! -d "$DIST_DIR/bin" ]]; then
  echo "missing build output directory: $DIST_DIR/bin" >&2
  exit 1
fi

target_dir_count=0
while IFS= read -r -d '' target_dir; do
  target_dir_count=$((target_dir_count + 1))
  target_name="$(basename "$target_dir")"
  os="${target_name%%_*}"
  archive_base="knit_${VERSION}_${target_name}"
  installer_base="$INSTALLER_DIR/knit_${VERSION}_${target_name}"
  staging_dir="$(mktemp -d)"
  cp -R "$target_dir"/. "$staging_dir"/
  cp -R "$ROOT_DIR/docs" "$staging_dir/docs"
  if [[ "$os" == "windows" ]]; then
    (cd "$staging_dir" && zip -qr "$PKG_DIR/${archive_base}.zip" .)
    echo " - created $PKG_DIR/${archive_base}.zip"
    cat > "${installer_base}.install.ps1" <<EOF
\$ErrorActionPreference = "Stop"
\$Root = Split-Path -Parent \$MyInvocation.MyCommand.Path
\$Archive = Join-Path (Split-Path -Parent \$Root) "${archive_base}.zip"
\$InstallDir = Join-Path \$env:LOCALAPPDATA "Knit"
New-Item -ItemType Directory -Force -Path \$InstallDir | Out-Null
Expand-Archive -Force -Path \$Archive -DestinationPath \$InstallDir
Write-Host "Installed Knit to \$InstallDir"
EOF
    echo " - created ${installer_base}.install.ps1"
  else
    tar -czf "$PKG_DIR/${archive_base}.tar.gz" -C "$staging_dir" .
    echo " - created $PKG_DIR/${archive_base}.tar.gz"
    if [[ "$os" == "darwin" ]]; then
      cat > "${installer_base}.install.command" <<EOF
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")/.." && pwd)"
ARCHIVE="\$ROOT_DIR/${archive_base}.tar.gz"
INSTALL_DIR="\${HOME}/Applications/Knit"
mkdir -p "\$INSTALL_DIR"
tar -xzf "\$ARCHIVE" -C "\$INSTALL_DIR"
echo "Installed Knit to \$INSTALL_DIR"
EOF
      chmod +x "${installer_base}.install.command"
      echo " - created ${installer_base}.install.command"
    else
      cat > "${installer_base}.install.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")/.." && pwd)"
ARCHIVE="\$ROOT_DIR/${archive_base}.tar.gz"
INSTALL_DIR="\${XDG_DATA_HOME:-\$HOME/.local/share}/knit"
mkdir -p "\$INSTALL_DIR"
tar -xzf "\$ARCHIVE" -C "\$INSTALL_DIR"
echo "Installed Knit to \$INSTALL_DIR"
EOF
      chmod +x "${installer_base}.install.sh"
      echo " - created ${installer_base}.install.sh"
    fi
  fi
  rm -rf "$staging_dir"
done < <(find "$DIST_DIR/bin" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)

if [[ "$target_dir_count" -eq 0 ]]; then
  echo "no build target directories found under: $DIST_DIR/bin" >&2
  exit 1
fi

for report in build-manifest.json sbom.spdx.json dependency-scan.json; do
  if [[ -f "$DIST_DIR/$report" ]]; then
    cp "$DIST_DIR/$report" "$PKG_DIR/$report"
    echo " - copied $PKG_DIR/$report"
  fi
done

pkg_checksums="$PKG_DIR/checksums.txt"
write_checksums "$pkg_checksums"

if [[ -n "${MACOS_SIGN_IDENTITY:-}" ]]; then
  echo "Signing macOS binaries with identity: $MACOS_SIGN_IDENTITY"
  while IFS= read -r -d '' mac_bin; do
    codesign --force --timestamp --sign "$MACOS_SIGN_IDENTITY" "$mac_bin"
  done < <(find "$DIST_DIR/bin" -type f \( -path "*/darwin_*/*" \) -print0)
fi

if [[ -n "${WINDOWS_SIGN_COMMAND:-}" ]]; then
  echo "Signing Windows binaries with command: $WINDOWS_SIGN_COMMAND"
  while IFS= read -r -d '' win_bin; do
    eval "$WINDOWS_SIGN_COMMAND \"$win_bin\""
  done < <(find "$DIST_DIR/bin" -type f -name "*.exe" -print0)
fi

if [[ -n "${RELEASE_SIGNING_PRIVATE_KEY:-}" ]]; then
  echo "Signing package checksums with OpenSSL private key."
  sig_file="$PKG_DIR/checksums.sig"
  openssl dgst -sha256 -sign "$RELEASE_SIGNING_PRIVATE_KEY" -out "$sig_file" "$pkg_checksums"
  echo " - created $sig_file"
  if [[ -n "${RELEASE_SIGNING_PUBLIC_KEY:-}" ]]; then
    openssl dgst -sha256 -verify "$RELEASE_SIGNING_PUBLIC_KEY" -signature "$sig_file" "$pkg_checksums"
    echo " - verified signature with public key"
  fi
fi

release_manifest="$PKG_DIR/release-manifest.json"
write_release_manifest "$release_manifest" "$pkg_checksums"

if [[ -n "${RELEASE_SIGNING_PRIVATE_KEY:-}" ]]; then
  manifest_sig="$PKG_DIR/release-manifest.sig"
  openssl dgst -sha256 -sign "$RELEASE_SIGNING_PRIVATE_KEY" -out "$manifest_sig" "$release_manifest"
  echo " - created $manifest_sig"
  if [[ -n "${RELEASE_SIGNING_PUBLIC_KEY:-}" ]]; then
    openssl dgst -sha256 -verify "$RELEASE_SIGNING_PUBLIC_KEY" -signature "$manifest_sig" "$release_manifest"
    echo " - verified release manifest signature with public key"
  fi
fi

if [[ -d "$NPM_TEMPLATE_DIR" ]]; then
  npm_pkg_dir="$PKG_DIR/npm/knit-daemon"
  rm -rf "$npm_pkg_dir"
  mkdir -p "$npm_pkg_dir/bin" "$npm_pkg_dir/lib" "$npm_pkg_dir/artifacts"
  sed "s/__VERSION__/${VERSION//\//\\/}/g" "$NPM_TEMPLATE_DIR/package.json.template" > "$npm_pkg_dir/package.json"
  cp "$NPM_TEMPLATE_DIR/README.md" "$npm_pkg_dir/README.md"
  cp "$NPM_TEMPLATE_DIR/bin/knit-daemon.js" "$npm_pkg_dir/bin/knit-daemon.js"
  cp "$NPM_TEMPLATE_DIR/lib/install.js" "$npm_pkg_dir/lib/install.js"
  chmod +x "$npm_pkg_dir/bin/knit-daemon.js"
  cp "$release_manifest" "$npm_pkg_dir/artifacts/release-manifest.json"
  cp "$pkg_checksums" "$npm_pkg_dir/artifacts/checksums.txt"
  if [[ -f "$PKG_DIR/checksums.sig" ]]; then
    cp "$PKG_DIR/checksums.sig" "$npm_pkg_dir/artifacts/checksums.sig"
  fi
  while IFS= read -r -d '' artifact_file; do
    rel="${artifact_file#$PKG_DIR/}"
    mkdir -p "$npm_pkg_dir/artifacts/$(dirname "$rel")"
    cp "$artifact_file" "$npm_pkg_dir/artifacts/$rel"
  done < <(find "$PKG_DIR" -maxdepth 2 -type f \( -name "*.tar.gz" -o -name "*.zip" \) -print0 | sort -z)
  echo " - created npm package scaffold at $npm_pkg_dir"
  if ! command -v npm >/dev/null 2>&1; then
    echo "npm is required to pack @chadsly/knit release artifacts" >&2
    exit 1
  fi
  (cd "$npm_pkg_dir" && npm pack --pack-destination "$PKG_DIR")
  echo " - created npm package tarball in $PKG_DIR"
  refresh_packaging_metadata "$pkg_checksums" "$release_manifest"
fi

if [[ -d "$PYTHON_TEMPLATE_DIR" ]]; then
  python_pkg_dir="$PKG_DIR/python/knit"
  rm -rf "$python_pkg_dir"
  mkdir -p "$python_pkg_dir/src/chadsly_knit" "$python_pkg_dir/src/chadsly_knit/artifacts"
  sed "s/__VERSION__/${PYTHON_PACKAGE_VERSION//\//\\/}/g" "$PYTHON_TEMPLATE_DIR/pyproject.toml.template" > "$python_pkg_dir/pyproject.toml"
  sed "s/__VERSION__/${PYTHON_PACKAGE_VERSION//\//\\/}/g" "$PYTHON_TEMPLATE_DIR/src/chadsly_knit/__init__.py.template" > "$python_pkg_dir/src/chadsly_knit/__init__.py"
  cp "$PYTHON_TEMPLATE_DIR/README.md" "$python_pkg_dir/README.md"
  cp "$PYTHON_TEMPLATE_DIR/src/chadsly_knit/cli.py" "$python_pkg_dir/src/chadsly_knit/cli.py"
  cp "$PYTHON_TEMPLATE_DIR/src/chadsly_knit/install.py" "$python_pkg_dir/src/chadsly_knit/install.py"
  cp "$release_manifest" "$python_pkg_dir/src/chadsly_knit/artifacts/release-manifest.json"
  cp "$pkg_checksums" "$python_pkg_dir/src/chadsly_knit/artifacts/checksums.txt"
  if [[ -f "$PKG_DIR/checksums.sig" ]]; then
    cp "$PKG_DIR/checksums.sig" "$python_pkg_dir/src/chadsly_knit/artifacts/checksums.sig"
  fi
  while IFS= read -r -d '' artifact_file; do
    cp "$artifact_file" "$python_pkg_dir/src/chadsly_knit/artifacts/$(basename "$artifact_file")"
  done < <(find "$PKG_DIR" -maxdepth 1 -type f \( -name "*.tar.gz" -o -name "*.zip" \) -print0 | sort -z)
  echo " - created python package scaffold at $python_pkg_dir"
  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 is required to pack chadsly-knit release artifacts" >&2
    exit 1
  fi
  if ! python3 -m build --version >/dev/null 2>&1; then
    echo "python build module is required to pack chadsly-knit release artifacts" >&2
    exit 1
  fi
  (cd "$python_pkg_dir" && python3 -m build --sdist --wheel --outdir "$PKG_DIR")
  echo " - created python package artifacts in $PKG_DIR"
  refresh_packaging_metadata "$pkg_checksums" "$release_manifest"
fi

echo "Packaging complete. Output: $PKG_DIR"
