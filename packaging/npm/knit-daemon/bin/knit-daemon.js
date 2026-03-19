#!/usr/bin/env node
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");
const { ensureInstalled, runtimeDirForPackage, daemonBinaryPath, packageRoot } = require("../lib/install");

function usage() {
  process.stdout.write("Usage: knit-daemon <start|path|version|install> [args...]\n");
}

function main() {
  const command = process.argv[2] || "start";
  if (command === "version") {
    const pkg = JSON.parse(fs.readFileSync(path.join(packageRoot(), "package.json"), "utf8"));
    process.stdout.write(String(pkg.version || "") + "\n");
    return;
  }
  if (command === "path") {
    ensureInstalled();
    process.stdout.write(daemonBinaryPath(runtimeDirForPackage()) + "\n");
    return;
  }
  if (command === "install") {
    ensureInstalled(true);
    process.stdout.write(runtimeDirForPackage() + "\n");
    return;
  }
  if (command !== "start") {
    usage();
    process.exitCode = 1;
    return;
  }
  ensureInstalled();
  const bin = daemonBinaryPath(runtimeDirForPackage());
  const args = process.argv.slice(3);
  const child = spawnSync(bin, args, { stdio: "inherit", env: process.env });
  if (typeof child.status === "number") {
    process.exitCode = child.status;
  } else if (child.error) {
    throw child.error;
  }
}

main();
