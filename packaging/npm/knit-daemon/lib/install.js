const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const { spawnSync } = require("node:child_process");

function packageRoot() {
  return path.resolve(__dirname, "..");
}

function runtimeDirForPackage() {
  return path.join(packageRoot(), "artifacts", "runtime");
}

function hostTarget() {
  const platform = process.platform;
  const archMap = { x64: "amd64", arm64: "arm64" };
  const arch = archMap[process.arch];
  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}`);
  }
  if (!["darwin", "linux", "win32"].includes(platform)) {
    throw new Error(`Unsupported platform: ${platform}`);
  }
  const os = platform === "win32" ? "windows" : platform;
  return `${os}_${arch}`;
}

function packageManifest() {
  const raw = fs.readFileSync(path.join(packageRoot(), "artifacts", "release-manifest.json"), "utf8");
  return JSON.parse(raw);
}

function archivePathForTarget() {
  const target = hostTarget();
  const manifest = packageManifest();
  const suffix = target.startsWith("windows_") ? ".zip" : ".tar.gz";
  const archive = (manifest.artifacts || []).find((item) => String(item.path || "").includes(`_${target}${suffix}`));
  if (!archive) {
    throw new Error(`No packaged archive found for target ${target}`);
  }
  return {
    target,
    relPath: String(archive.path),
    sha256: String(archive.sha256 || "")
  };
}

function verifyArchive(absPath, expectedSHA256) {
  const actual = crypto.createHash("sha256").update(fs.readFileSync(absPath)).digest("hex");
  if (actual !== expectedSHA256) {
    throw new Error(`Archive checksum mismatch for ${path.basename(absPath)}`);
  }
}

function extractArchive(absArchive, outDir) {
  fs.rmSync(outDir, { recursive: true, force: true });
  fs.mkdirSync(outDir, { recursive: true });
  if (absArchive.endsWith(".zip")) {
    const ps = spawnSync("powershell", [
      "-NoProfile",
      "-Command",
      `Expand-Archive -Force -Path '${absArchive.replace(/'/g, "''")}' -DestinationPath '${outDir.replace(/'/g, "''")}'`
    ], { stdio: "pipe" });
    if (ps.status !== 0) {
      throw new Error(ps.stderr.toString() || "Expand-Archive failed");
    }
    return;
  }
  const tar = spawnSync("tar", ["-xzf", absArchive, "-C", outDir], { stdio: "pipe" });
  if (tar.status !== 0) {
    throw new Error(tar.stderr.toString() || "tar extraction failed");
  }
}

function daemonBinaryPath(runtimeDir) {
  return path.join(runtimeDir, process.platform === "win32" ? "daemon.exe" : "daemon");
}

function ensureInstalled(force = false) {
  const runtimeDir = runtimeDirForPackage();
  const daemonPath = daemonBinaryPath(runtimeDir);
  if (!force && fs.existsSync(daemonPath)) {
    return runtimeDir;
  }
  const archive = archivePathForTarget();
  const absArchive = path.join(packageRoot(), "artifacts", archive.relPath);
  verifyArchive(absArchive, archive.sha256);
  extractArchive(absArchive, runtimeDir);
  if (!fs.existsSync(daemonPath)) {
    throw new Error(`Daemon binary missing after extraction: ${daemonPath}`);
  }
  return runtimeDir;
}

if (require.main === module) {
  ensureInstalled();
}

module.exports = {
  ensureInstalled,
  runtimeDirForPackage,
  daemonBinaryPath,
  hostTarget,
  packageRoot
};
