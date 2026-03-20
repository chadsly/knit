import hashlib
import json
import os
import platform
import shutil
import tarfile
import tempfile
import zipfile
from pathlib import Path

from . import __version__


def package_root() -> Path:
    return Path(__file__).resolve().parent


def artifacts_dir() -> Path:
    return package_root() / "artifacts"


def host_target() -> str:
    system = platform.system().lower()
    if system == "darwin":
        os_name = "darwin"
    elif system == "linux":
        os_name = "linux"
    elif system == "windows":
        os_name = "windows"
    else:
        raise RuntimeError(f"Unsupported platform: {platform.system()}")

    machine = platform.machine().lower()
    if machine in ("x86_64", "amd64"):
        arch = "amd64"
    elif machine in ("arm64", "aarch64"):
        arch = "arm64"
    else:
        raise RuntimeError(f"Unsupported architecture: {platform.machine()}")
    return f"{os_name}_{arch}"


def package_manifest() -> dict:
    return json.loads((artifacts_dir() / "release-manifest.json").read_text(encoding="utf-8"))


def runtime_dir_for_package() -> Path:
    target = host_target()
    if os.name == "nt":
        root = Path(os.environ.get("LOCALAPPDATA", str(Path.home() / "AppData" / "Local")))
        return root / "Knit" / "python-runtime" / __version__ / target
    if platform.system().lower() == "darwin":
        return Path.home() / "Library" / "Application Support" / "Knit" / "python-runtime" / __version__ / target
    root = Path(os.environ.get("XDG_DATA_HOME", str(Path.home() / ".local" / "share")))
    return root / "knit" / "python-runtime" / __version__ / target


def archive_path_for_target() -> tuple[Path, str]:
    target = host_target()
    suffix = ".zip" if target.startswith("windows_") else ".tar.gz"
    for artifact in package_manifest().get("artifacts", []):
        rel_path = str(artifact.get("path") or "")
        if f"_{target}{suffix}" in rel_path:
            return artifacts_dir() / Path(rel_path).name, str(artifact.get("sha256") or "")
    raise RuntimeError(f"No packaged archive found for target {target}")


def verify_archive(archive_path: Path, expected_sha256: str) -> None:
    actual = hashlib.sha256(archive_path.read_bytes()).hexdigest()
    if actual != expected_sha256:
        raise RuntimeError(f"Archive checksum mismatch for {archive_path.name}")


def extract_archive(archive_path: Path, out_dir: Path) -> None:
    with tempfile.TemporaryDirectory(prefix="knit-python-runtime-") as temp_dir:
        temp_path = Path(temp_dir)
        if archive_path.suffix == ".zip":
            with zipfile.ZipFile(archive_path) as archive:
                archive.extractall(temp_path)
        else:
            with tarfile.open(archive_path, mode="r:gz") as archive:
                archive.extractall(temp_path)
        out_dir.parent.mkdir(parents=True, exist_ok=True)
        if out_dir.exists():
            shutil.rmtree(out_dir)
        shutil.move(str(temp_path), str(out_dir))


def daemon_binary_path(runtime_dir: Path) -> Path:
    return runtime_dir / ("daemon.exe" if os.name == "nt" else "daemon")


def ensure_installed(force: bool = False) -> Path:
    runtime_dir = runtime_dir_for_package()
    daemon_path = daemon_binary_path(runtime_dir)
    if not force and daemon_path.exists():
        return runtime_dir
    archive_path, expected_sha256 = archive_path_for_target()
    verify_archive(archive_path, expected_sha256)
    extract_archive(archive_path, runtime_dir)
    if not daemon_path.exists():
        raise RuntimeError(f"Daemon binary missing after extraction: {daemon_path}")
    if os.name != "nt":
        daemon_path.chmod(0o755)
    return runtime_dir
