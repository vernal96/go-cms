#!/usr/bin/env python3
"""Export and verify versioned CMS/connector modules without go.work or replace."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import uuid
import zipfile

ROOT = Path(__file__).resolve().parents[1]
BACKEND = ROOT / 'backend'
CMS_MODULE = 'github.com/vernal96/go-cms'
PACKAGES = json.loads((Path(__file__).with_name('go-packages.json')).read_text())


def module_path(name):
    return CMS_MODULE if name == 'cms' else CMS_MODULE + '/connectors/' + name


def run(directory, environment, *args):
    subprocess.run(['go', *args], cwd=directory, env=environment, check=True)


def publish(proxy, modules, version):
    """Create immutable Go proxy artifacts. Bootstrap and final versions differ."""
    for name, directory in modules.items():
        module = module_path(name)
        target = proxy / module / '@v'
        target.mkdir(parents=True, exist_ok=True)
        (target / (version + '.mod')).write_bytes((directory / 'go.mod').read_bytes())
        (target / (version + '.info')).write_text(json.dumps({
            'Version': version, 'Time': '2026-09-16T00:00:00Z'}))
        (target / 'list').write_text(version + '\n')
        with zipfile.ZipFile(target / (version + '.zip'), 'w', zipfile.ZIP_DEFLATED) as archive:
            for file in sorted(directory.rglob('*')):
                if file.is_file():
                    archive.write(file, module + '@' + version + '/' + file.relative_to(directory).as_posix())


def test(directory, environment, log):
    # Keep the full evidence while making service-dependent skips visible.
    with log.open('w') as output:
        result = subprocess.run(['go', 'test', '-mod=readonly', '-count=1', '-json', './...'],
                                cwd=directory, env=environment, stdout=output,
                                stderr=subprocess.STDOUT)
    skips = []
    for line in log.read_text().splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get('Action') == 'skip' and event.get('Test'):
            skips.append(event['Package'] + '/' + event['Test'])
    for skipped in skips:
        print('SKIP', skipped, flush=True)
    if result.returncode:
        print(log.read_text(), flush=True)
        raise subprocess.CalledProcessError(result.returncode, result.args)
    print('PASS tests:', directory.name, f'({len(skips)} skipped; {log})', flush=True)


def check(base):
    declared = set(PACKAGES) - {'cms'}
    actual = {path.name for path in (BACKEND / 'connectors').iterdir() if path.is_dir()}
    if declared != actual:
        raise RuntimeError(f'Update go-packages.json for connectors: {declared ^ actual}')
    identifier = uuid.uuid4().hex
    bootstrap = 'v0.0.0-bootstrap.' + identifier
    version = 'v0.0.0-check.' + identifier
    proxy = base / 'proxy'
    # Keep third-party checksum verification enabled. Local synthetic versions
    # never go to the public checksum database, nor bypass our local proxy.
    environment = os.environ | {
        'GOWORK': 'off', 'GOFLAGS': '', 'GONOPROXY': 'none',
        'GONOSUMDB': ','.join(filter(None, [os.environ.get('GONOSUMDB'), CMS_MODULE])),
        'GOPROXY': proxy.as_uri() + ',' + os.environ.get('GOPROXY', 'https://proxy.golang.org,direct'),
    }
    modules = {}
    for name, definition in PACKAGES.items():
        directory = base / 'modules' / name
        source = BACKEND / definition.get('source', 'connectors/' + name)
        shutil.copytree(source, directory / definition.get('target', ''))
        manifest = (BACKEND / 'go.mod').read_text().replace(
            'module ' + CMS_MODULE, 'module ' + module_path(name), 1)
        manifest += '\nrequire (\n' + ''.join(
            '\t' + module_path(dependency) + ' ' + bootstrap + '\n'
            for dependency in definition['dependencies']) + ')\n'
        (directory / 'go.mod').write_text(manifest)
        shutil.copyfile(BACKEND / 'go.sum', directory / 'go.sum')
        modules[name] = directory
    # The package graph is acyclic; the module graph includes CMS adapters ->
    # PostgreSQL -> kernel. A bootstrap proxy lets tidy resolve that graph.
    publish(proxy, modules, bootstrap)
    for directory in modules.values():
        run(directory, environment, 'mod', 'tidy')
    for directory in modules.values():
        manifest = directory / 'go.mod'
        manifest.write_text(manifest.read_text().replace(bootstrap, version))
        sums = directory / 'go.sum'
        sums.write_text(''.join(line for line in sums.read_text().splitlines(keepends=True)
                                if bootstrap not in line))
    publish(proxy, modules, version)
    logs = base / 'logs'
    logs.mkdir()
    for name, directory in modules.items():
        before = (directory / 'go.mod').read_bytes()
        run(directory, environment, 'mod', 'tidy')
        if (directory / 'go.mod').read_bytes() != before:
            raise RuntimeError(f'Final dependency graph changed after packaging: {name}')
        run(directory, environment, 'mod', 'tidy', '-diff')
        test(directory, environment, logs / (name + '.jsonl'))
        run(directory, environment, 'vet', '-mod=readonly', './...')
        run(directory, environment, 'build', '-mod=readonly', './...')
        print('PASS standalone module:', module_path(name), flush=True)
    consumer = base / 'consumer'
    shutil.copytree(ROOT / 'examples/external-app', consumer)
    run(consumer, environment, 'mod', 'edit', '-dropreplace=' + CMS_MODULE,
        '-require=' + CMS_MODULE + '@' + version)
    run(consumer, environment, 'mod', 'tidy')
    run(consumer, environment, 'build', '-mod=readonly', '-o', str(base / 'consumer-app'), '.')
    test(consumer, environment, logs / 'consumer.jsonl')
    run(consumer, environment, 'vet', '-mod=readonly', './...')
    # Verify the exported and consumer manifests did not acquire escape hatches.
    for directory in [*modules.values(), consumer]:
        manifest = json.loads(subprocess.check_output(
            ['go', 'mod', 'edit', '-json'], cwd=directory, env=environment))
        if manifest.get('Replace'):
            raise RuntimeError(f'Unexpected replace directives in {directory}')
    (base / 'result.json').write_text(json.dumps({'version': version,
        'modules': {name: module_path(name) for name in modules},
        'workspace': False, 'replace': False}, indent=2) + '\n')
    print('PASS versioned external consumer; GOWORK=off, no replace', flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path,
                        help='retain exported modules, local proxy and logs in a new directory')
    args = parser.parse_args()
    if args.output:
        directory = args.output.resolve()
        directory.mkdir(parents=True, exist_ok=False)
        check(directory)
    else:
        with tempfile.TemporaryDirectory(prefix='cms-package-boundaries-') as temporary:
            check(Path(temporary))


if __name__ == '__main__':
    main()
