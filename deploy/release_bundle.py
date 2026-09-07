#!/usr/bin/env python3
"""Build and verify immutable release directories without contacting production."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

COMPONENTS = {'backend', 'frontend', 'admin'}
PUBLIC_VITE_KEYS = {'VITE_SITE_BASE_URL', 'VITE_APP_STORE_URL', 'VITE_GOOGLE_PLAY_URL', 'VITE_WIP_ADMIN_CODE'}


def run(args, cwd=None, env=None):
    subprocess.run(args, cwd=cwd, env=env, check=True)


def digest(path):
    h = hashlib.sha256()
    with open(path, 'rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def verify(root):
    root = Path(root)
    manifest = json.loads((root / 'manifest.json').read_text())
    if manifest.get('format') != 1 or not re.fullmatch(r'[0-9a-f]{40}', manifest.get('commit', '')):
        raise ValueError('invalid release manifest')
    if not set(manifest['components']) <= COMPONENTS or not manifest['components']:
        raise ValueError('invalid release components')
    expected = manifest['files']
    required = {'deploy/remote_release.py', 'deploy/httpd-alumni.conf', 'deploy/_set_docroot.php', 'deploy/_legacy_docroot.php', 'deploy/_legacy_url_rewriter.php'}
    for component in manifest['components']:
        required.update({'backend/server', 'backend/backfill'} if component == 'backend' else {component + '/index.html'})
    if not required <= set(expected) or 'migrations/' + manifest['minimum_web_migration'] not in expected:
        raise ValueError('release is missing required deployment files')
    actual = {p.relative_to(root / 'artifacts').as_posix() for p in (root / 'artifacts').rglob('*') if p.is_file()}
    if actual != set(expected):
        raise ValueError('artifact inventory mismatch')
    for name, sha in expected.items():
        path = root / 'artifacts' / name
        if not name or Path(name).is_absolute() or '..' in Path(name).parts:
            raise ValueError('invalid artifact path')
        if any(p.is_symlink() for p in [path, *list(path.parents)[:len(Path(name).parts)]]) or not path.is_file() or digest(path) != sha:
            raise ValueError('artifact hash or file type mismatch: ' + name)
    return manifest


def prepare(repo, output, components, patch_mode):
    repo, output = Path(repo).resolve(), Path(output).resolve()
    if output.exists():
        raise ValueError('output directory already exists; preserve prior candidates')
    dirty = subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=normal'], cwd=repo).strip()
    if dirty:
        raise ValueError('commit the reviewed source before preparing a release')
    commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo).decode().strip()
    output.mkdir(parents=True, mode=0o700)
    artifacts = output / 'artifacts'
    artifacts.mkdir()
    # A separate repository avoids the enclosing workspace VCS provenance bug.
    with tempfile.TemporaryDirectory(prefix='dflh-release-source-') as temporary:
        source = Path(temporary) / 'source'
        run(['git', 'clone', '--quiet', '--shared', '--no-checkout', str(repo), str(source)])
        run(['git', 'checkout', '--quiet', '--detach', commit], cwd=source)
        if 'backend' in components:
            (artifacts / 'backend').mkdir()
            env = dict(os.environ, CGO_ENABLED='0', GOOS='linux', GOARCH='amd64')
            for binary in ['server', 'backfill']:
                path = artifacts / 'backend' / binary
                run(['go', 'build', '-o', str(path), './cmd/' + binary], cwd=source / 'backend', env=env)
                metadata = subprocess.check_output(['go', 'version', '-m', str(path)]).decode()
                if 'vcs.revision=' + commit not in metadata or 'vcs.modified=false' not in metadata:
                    raise ValueError('binary does not identify the clean candidate commit')
        for component in ['frontend', 'admin']:
            if component not in components:
                continue
            env = {k: v for k, v in os.environ.items() if not k.startswith('VITE_')}
            env_file = repo / component / '.env'
            if env_file.exists():
                for line in env_file.read_text().splitlines():
                    key, sep, value = line.partition('=')
                    if sep and key.strip() in PUBLIC_VITE_KEYS:
                        env[key.strip()] = value.strip().strip('\"\'')
            if component == 'frontend':
                if patch_mode == 'false':
                    env['VITE_WIP_ADMIN_CODE'] = ''
                elif not env.get('VITE_WIP_ADMIN_CODE'):
                    raise ValueError('patch-mode=true requires the configured WIP code')
            run(['npm', 'ci', '--no-audit', '--no-fund'], cwd=source / component, env=env)
            run(['npm', 'run', 'build'], cwd=source / component, env=env)
            shutil.copytree(source / component / 'dist', artifacts / component)
        shutil.copytree(source / 'deploy', artifacts / 'deploy', ignore=shutil.ignore_patterns('*.service', '__pycache__', 'tests'))
        # Only numbered migrations; no fixtures, credentials or complete checkout.
        (artifacts / 'migrations').mkdir()
        for path in (source / 'backend/migrations').glob('[0-9][0-9][0-9]_*.sql'):
            shutil.copyfile(path, artifacts / 'migrations' / path.name)
    manifest = {'format': 1, 'commit': commit, 'components': components, 'patch_mode': patch_mode,
                'status': 'PREPARED_NOT_DEPLOYED', 'minimum_web_migration': '062_create_profile_file_history.sql',
                'files': {p.relative_to(artifacts).as_posix(): digest(p) for p in sorted(artifacts.rglob('*')) if p.is_file()}}
    (output / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    verify(output)
    return output
