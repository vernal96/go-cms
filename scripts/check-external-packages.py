#!/usr/bin/env python3
"""Compile the consumer and relocated connectors without changing the checkout."""
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
BACKEND = ROOT / 'backend'


def run(args, cwd):
    subprocess.run(args, cwd=cwd, check=True)


with tempfile.TemporaryDirectory(prefix='cms-package-boundaries-') as temporary:
    base = Path(temporary)
    consumer = base / 'consumer'
    shutil.copytree(ROOT / 'examples/external-app', consumer)
    run(['go', 'mod', 'edit', '-replace=github.com/vernal96/go-cms=' + str(BACKEND)], consumer)
    run(['go', 'test', '-count=1', '-v', './...'], consumer)
    run(['go', 'build', '-o', str(base / 'consumer-app'), '.'], consumer)
    for connector in sorted((BACKEND / 'connectors').iterdir()):
        if not connector.is_dir():
            continue
        target = base / connector.name
        shutil.copytree(connector, target)
        module = (BACKEND / 'go.mod').read_text().replace(
            'module github.com/vernal96/go-cms', 'module example.org/connector-' + connector.name, 1)
        (target / 'go.mod').write_text(module + '\nrequire github.com/vernal96/go-cms v0.0.0\n'
                                    + 'replace github.com/vernal96/go-cms => ' + str(BACKEND) + '\n')
        shutil.copyfile(BACKEND / 'go.sum', target / 'go.sum')
        run(['go', 'build', './...'], target)
        print('PASS relocated connector:', connector.name, flush=True)
