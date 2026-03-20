import os
import subprocess
import sys

from . import __version__
from .install import daemon_binary_path, ensure_installed


def usage() -> None:
    sys.stdout.write("Usage: knit <start|path|version> [args...]\n")


def main() -> int:
    command = sys.argv[1] if len(sys.argv) > 1 else "start"
    if command == "version":
        sys.stdout.write(f"{__version__}\n")
        return 0
    if command == "path":
        runtime_dir = ensure_installed()
        sys.stdout.write(f"{daemon_binary_path(runtime_dir)}\n")
        return 0
    if command != "start":
        usage()
        return 1
    runtime_dir = ensure_installed()
    daemon_path = str(daemon_binary_path(runtime_dir))
    result = subprocess.run([daemon_path, *sys.argv[2:]], env=os.environ.copy(), check=False)
    return int(result.returncode)


if __name__ == "__main__":
    raise SystemExit(main())
