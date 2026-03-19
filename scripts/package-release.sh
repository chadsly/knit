#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${1:-$ROOT_DIR/dist}"
DIST_DIR="$(cd "$DIST_DIR" && pwd)"
PKG_DIR="$DIST_DIR/packages"
NPM_TEMPLATE_DIR="$ROOT_DIR/packaging/npm/knit-daemon"
VERSION="${VERSION:-dev}"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-0}"

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

echo "Packaging release artifacts from: $DIST_DIR/bin"
while IFS= read -r -d '' target_dir; do
  target_name="$(basename "$target_dir")"
  os="${target_name%%_*}"
  archive_base="knit_${VERSION}_${target_name}"
  installer_base="$INSTALLER_DIR/knit_${VERSION}_${target_name}"
  if [[ "$os" == "windows" ]]; then
    (cd "$target_dir" && zip -qr "$PKG_DIR/${archive_base}.zip" .)
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
    tar -czf "$PKG_DIR/${archive_base}.tar.gz" -C "$target_dir" .
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
done < <(find "$DIST_DIR/bin" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)

pkg_checksums="$PKG_DIR/checksums.txt"
rm -f "$pkg_checksums"
while IFS= read -r -d '' pkg_file; do
  hash_file "$pkg_file" >> "$pkg_checksums"
done < <(find "$PKG_DIR" -type f \( -name "*.tar.gz" -o -name "*.zip" -o -name "*.install.sh" -o -name "*.install.command" -o -name "*.install.ps1" \) -print0 | sort -z)
echo " - created $pkg_checksums"

for report in build-manifest.json sbom.spdx.json dependency-scan.json; do
  if [[ -f "$DIST_DIR/$report" ]]; then
    cp "$DIST_DIR/$report" "$PKG_DIR/$report"
    echo " - copied $PKG_DIR/$report"
  fi
done

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
  done < "$pkg_checksums"
  echo
  echo "  ]"
  echo "}"
} > "$release_manifest"
echo " - created $release_manifest"

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
fi

echo "Packaging complete. Output: $PKG_DIR"
