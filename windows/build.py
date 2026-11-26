"""Stage the Windows executables and Edge extension for a release archive."""

import argparse
import hashlib
import json
import os
import shutil
import subprocess
from base64 import b64decode
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--version", default="dev")
    args = parser.parse_args()
    output = args.output.resolve()
    root = Path(__file__).resolve().parent.parent
    extension = root / "windows" / "browser-extension"

    manifest = json.loads((extension / "manifest.json").read_text(encoding="utf-8"))
    digest = hashlib.sha256(b64decode(manifest["key"], validate=True)).hexdigest()[:32]
    extension_id = digest.translate(str.maketrans("0123456789abcdef", "abcdefghijklmnop"))
    origin = f"chrome-extension://{extension_id}/"

    output.mkdir(parents=True, exist_ok=True)
    env = {**os.environ, "GOOS": "windows", "CGO_ENABLED": "0"}
    builds = (
        ("openmessage.exe", f"-s -w -X main.version={args.version}", "."),
        (
            "openmessage-browser-host.exe",
            f"-s -w -H windowsgui -X main.allowedOrigin={origin}",
            "./cmd/openmessage-browser-host",
        ),
    )
    for filename, linker_flags, package in builds:
        subprocess.run(
            ["go", "build", "-trimpath", "-ldflags", linker_flags, "-o", str(output / filename), package],
            cwd=root,
            env=env,
            check=True,
        )

    dest = output / "browser-extension"
    dest.mkdir(exist_ok=True)
    for filename in ("background.js", "manifest.json", "README.md"):
        shutil.copyfile(extension / filename, dest / filename)

    host_manifest = {
        "name": "com.openmessage.google_refresh",
        "description": "OpenMessage local Google login refresh",
        "path": "openmessage-browser-host.exe",
        "type": "stdio",
        "allowed_origins": [origin],
    }
    (output / "openmessage-browser-host.json").write_text(
        json.dumps(host_manifest, indent=2) + "\n", encoding="utf-8"
    )
    shutil.copyfile(root / "windows" / "register-browser-refresh.ps1", output / "register-browser-refresh.ps1")


if __name__ == "__main__":
    main()
