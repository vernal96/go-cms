#!/usr/bin/env python3
"""Verify the released kernel from an independent Go module."""

import json
import os
from pathlib import Path
import subprocess


ROOT = Path(__file__).resolve().parents[1]
CONSUMER = ROOT / "examples" / "external-app"
KERNEL_MODULE = "github.com/vernal96/go-cms-kernel"
KERNEL_VERSION = "v0.1.0"


def run(*args, capture=False):
    environment = os.environ | {"GOWORK": "off", "GOFLAGS": ""}
    return subprocess.run(
        args,
        cwd=CONSUMER,
        env=environment,
        check=True,
        text=True,
        capture_output=capture,
    )


def verify_manifest():
    manifest = json.loads(run("go", "mod", "edit", "-json", capture=True).stdout)
    if manifest.get("Replace"):
        raise RuntimeError("external consumer must not use replace directives")
    required = {
        item["Path"]: item["Version"] for item in manifest.get("Require", [])
    }
    if required.get(KERNEL_MODULE) != KERNEL_VERSION:
        raise RuntimeError(
            f"external consumer must require {KERNEL_MODULE} {KERNEL_VERSION}"
        )


def verify_imports():
    obsolete = (
        "github.com/vernal96/go-cms/kernel",
        "github.com/vernal96/go-cms/connectors",
        "github.com/vernal96/go-cms/internal/",
    )
    for source in CONSUMER.rglob("*.go"):
        contents = source.read_text()
        for prefix in obsolete:
            if prefix in contents:
                raise RuntimeError(f"{source.relative_to(ROOT)} imports {prefix}")


def test():
    environment = os.environ | {"GOWORK": "off", "GOFLAGS": ""}
    process = subprocess.Popen(
        ["go", "test", "-mod=readonly", "-count=1", "-json", "./..."],
        cwd=CONSUMER,
        env=environment,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    output = []
    skips = []
    assert process.stdout is not None
    for line in process.stdout:
        output.append(line)
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("Action") == "skip" and event.get("Test"):
            skips.append(f"{event['Package']}/{event['Test']}")
    if process.wait():
        print("".join(output), end="")
        raise subprocess.CalledProcessError(process.returncode, process.args)
    for skipped in skips:
        print("SKIP", skipped)
    print(f"PASS external consumer tests ({len(skips)} skipped)")


def main():
    verify_manifest()
    verify_imports()
    run("go", "list", "-m", f"{KERNEL_MODULE}@{KERNEL_VERSION}")
    run("go", "mod", "tidy", "-diff")
    test()
    run("go", "vet", "-mod=readonly", "./...")
    run("go", "build", "-mod=readonly", "./...")
    print(f"PASS external kernel {KERNEL_MODULE}@{KERNEL_VERSION}; GOWORK=off, no replace")


if __name__ == "__main__":
    main()
