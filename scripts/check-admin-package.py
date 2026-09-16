#!/usr/bin/env python3
"""Pack a clean admin SDK and compile its independent consumer (Node 24+)."""
import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def run(directory, *command):
    subprocess.run(command, cwd=directory, check=True)


def targets(value):
    if isinstance(value, str):
        yield value.removeprefix('./')
    else:
        for child in value.values():
            yield from targets(child)


def main():
    with tempfile.TemporaryDirectory(prefix='cms-admin-package-') as temporary:
        base = Path(temporary)
        sdk = base / 'admin'
        ignore = shutil.ignore_patterns('node_modules', 'dist', 'dist-sdk', '*.tgz')
        shutil.copytree(ROOT / 'frontend-admin', sdk, ignore=ignore)
        run(sdk, 'npm', 'ci', '--no-audit', '--no-fund')
        # No build step here: npm pack must produce every export on its own.
        run(sdk, 'npm', 'pack', '--pack-destination', str(base))
        archive, = base.glob('*.tgz')
        with tarfile.open(archive) as packed:
            manifest = json.load(packed.extractfile('package/package.json'))
            for target in targets(manifest['exports']):
                member = packed.getmember('package/' + target)
                if not member.isfile() or member.size == 0:
                    raise RuntimeError('Invalid packaged export: ' + target)
            if manifest.get('private') or manifest.get('dependencies'):
                raise RuntimeError('SDK must be publishable and share its runtime dependencies as peers')
            for member in packed.getmembers():
                if member.name.startswith(('package/src/', 'package/node_modules/')):
                    raise RuntimeError('Unexpected source or bundled dependency: ' + member.name)
        consumer = base / 'plugin'
        shutil.copytree(ROOT / 'examples/admin-plugin', consumer, ignore=ignore)
        run(consumer, 'npm', 'install', '--no-save', '--package-lock=false',
            '--no-audit', '--no-fund', str(archive))
        run(consumer, 'npm', 'run', 'build')
        # Build a real host too: library builds externalize the SDK and cannot
        # detect missing transitive JS/CSS exports by themselves.
        run(consumer, 'npx', '--no-install', 'vite', 'build', '--config', 'demo/vite.config.ts')
        run(consumer, 'npm', 'ls', '@go-cms/admin', 'vue', 'vue-router', 'element-plus')
        print('PASS clean npm pack: exports, plugin declarations, host build and shared UI dependencies')


if __name__ == '__main__':
    main()
