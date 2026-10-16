"""Install the private pilot without editing the user's Codex configuration by hand."""
import json
import os
import pathlib
import platform
import shutil
import subprocess
import sys


def main():
    root = pathlib.Path(__file__).resolve().parent
    target = json.loads((root / "platform.json").read_text(encoding="utf-8"))
    machine = platform.machine().lower()
    arch = "arm64" if machine in ("arm64", "aarch64") else "amd64" if machine in ("x86_64", "amd64") else machine
    if target != {"os": platform.system().lower(), "arch": arch}:
        raise RuntimeError("This package is for a different operating system or processor. Download the matching Knit pilot package.")
    codex = shutil.which("codex")
    if not codex:
        raise RuntimeError("Codex CLI was not found. Install the official Codex CLI before running this private-pilot installer.")
    subprocess.run([codex, "plugin", "marketplace", "add", str(root), "--json"], check=True, stdout=subprocess.DEVNULL)
    installed = json.loads(subprocess.check_output([codex, "plugin", "add", "knit-desktop@knit-private-pilot", "--json"], text=True))
    cache = pathlib.Path(installed["installedPath"]).resolve()
    binary = cache / "bin" / ("knit.exe" if sys.platform == "win32" else "knit")
    if not binary.is_file():
        raise RuntimeError("Installed Knit binary is missing; installation is incomplete.")
    # Current Codex builds do not all expand PLUGIN_ROOT in legacy MCP manifests.
    # Resolve only installation paths; never copy tokens or runtime state.
    config = {"mcpServers": {"knit_desktop": {"command": str(binary), "args": ["mcp", "--desktop"], "cwd": str(cache)}}}
    overrides = {key: os.environ[key] for key in ("KNIT_ADDR", "KNIT_DATA_DIR", "KNIT_CONFIG_PATH", "KNIT_SQLITE_PATH") if os.environ.get(key)}
    if overrides:
        config["mcpServers"]["knit_desktop"]["env"] = overrides
    (cache / "mcp.json").write_text(json.dumps(config, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(installed))


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print("Knit pilot installation failed: " + str(error), file=sys.stderr)
        sys.exit(1)
