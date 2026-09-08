#!/usr/bin/env python3
"""Prepare locally, or explicitly deploy an already verified release directory."""
import argparse
from datetime import datetime, timezone
import os
from pathlib import Path
import re
import shlex
import subprocess
import tarfile
import tempfile
import json
import urllib.request
from release_bundle import prepare, verify, COMPONENTS


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('target', nargs='?', default='daeil-prod')
    parser.add_argument('port', nargs='?', type=int)
    parser.add_argument('--only', default=None)
    parser.add_argument('--patch-mode', choices=['true', 'false'])
    parser.add_argument('--prepare-only', action='store_true')
    parser.add_argument('--bundle', type=Path)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--apply-migrations', action='store_true')
    action = parser.add_mutually_exclusive_group()
    action.add_argument('--activate-test-user', type=int)
    action.add_argument('--activate-all-users', action='store_true')
    action.add_argument('--pause-erasure', action='store_true')
    action.add_argument('--activate-retention', action='store_true')
    action.add_argument('--pause-retention', action='store_true')
    parser.add_argument('--release-id')
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[1]
    components = args.only.split(',') if args.only else ['backend', 'frontend', 'admin']
    if not components or not set(components) <= COMPONENTS or len(set(components)) != len(components):
        parser.error('--only must contain distinct backend,frontend,admin components')
    if args.bundle and (args.prepare_only or args.output):
        parser.error('--bundle cannot be combined with --prepare-only or --output')
    rollout_action = args.activate_test_user is not None or args.activate_all_users or args.pause_erasure or args.activate_retention or args.pause_retention
    if rollout_action:
        if args.bundle or args.prepare_only or args.output or args.apply_migrations or args.only or args.patch_mode:
            parser.error('rollout actions cannot be combined with build/deploy options')
        if not args.release_id or not re.fullmatch(r'[0-9]{8}T[0-9]{6}Z-[0-9a-f]{12}', args.release_id):
            parser.error('--release-id must identify a verified deployment')
        if not re.fullmatch(r'[A-Za-z0-9_][A-Za-z0-9_.@-]*', args.target):
            parser.error('invalid SSH target')
        if args.port is not None and not 1 <= args.port <= 65535:
            parser.error('invalid SSH port')
        remote = '/app/releases/' + args.release_id
        command = ['sudo', '/usr/bin/python3.6', remote + '/remote_release.py', remote]
        if args.activate_test_user is not None:
            if args.activate_test_user <= 0:
                parser.error('test user must be a positive disposable account ID')
            command += ['--activate-test-user', str(args.activate_test_user)]
        elif args.activate_retention or args.pause_retention:
            command += ['--activate-retention' if args.activate_retention else '--pause-retention']
        else:
            command += ['--activate-all-users' if args.activate_all_users else '--pause-erasure']
        ssh = ['ssh', '-o', 'BatchMode=yes'] + (['-p', str(args.port)] if args.port else []) + [args.target]
        subprocess.run(ssh + [' '.join(shlex.quote(value) for value in command)], check=True)
        return
    if args.release_id:
        parser.error('--release-id requires a rollout action')
    if args.prepare_only:
        if 'frontend' in components and args.patch_mode is None:
            parser.error('--patch-mode is required when preparing frontend')
        output = args.output or Path.home() / '.local/share/dflh-release' / ('candidate-' + datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
        bundle = prepare(repo, output, components, args.patch_mode)
        print('Prepared locally; no production connection: ' + str(bundle))
        return
    if not args.bundle:
        parser.error('Prepare first with --prepare-only, then deploy the reviewed directory with --bundle=PATH. No production action performed.')
    manifest = verify(args.bundle)
    if args.only and components != manifest['components']:
        parser.error('--only must match the reviewed bundle components')
    if args.patch_mode is not None and args.patch_mode != manifest['patch_mode']:
        parser.error('--patch-mode differs from the reviewed bundle')
    if not re.fullmatch(r'[A-Za-z0-9_][A-Za-z0-9_.@-]*', args.target):
        parser.error('invalid SSH target')
    if args.port is not None and not 1 <= args.port <= 65535:
        parser.error('invalid SSH port')
    release_id = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + manifest['commit'][:12]
    remote = '/app/releases/' + release_id
    ssh = ['ssh', '-o', 'BatchMode=yes'] + (['-p', str(args.port)] if args.port else []) + [args.target]
    scp = ['scp'] + (['-P', str(args.port)] if args.port else [])
    # No build, npm install or source mutation takes place on this path.
    with tempfile.TemporaryDirectory(prefix='dflh-release-transfer-') as temporary:
        archive = Path(temporary) / 'bundle.tar.gz'
        with tarfile.open(archive, 'w:gz') as tar:
            tar.add(args.bundle / 'manifest.json', arcname='manifest.json', recursive=False)
            for name in sorted(manifest['files']):
                tar.add(args.bundle / 'artifacts' / name, arcname='artifacts/' + name, recursive=False)
        subprocess.run(ssh + ['umask 077 && mkdir -p ' + shlex.quote(remote)], check=True)
        helper = args.bundle / 'artifacts/deploy/remote_release.py'
        subprocess.run(scp + [str(archive), args.target + ':' + remote + '/bundle.tar.gz'], check=True)
        subprocess.run(scp + [str(helper), args.target + ':' + remote + '/remote_release.py'], check=True)
        command = ['sudo', '/usr/bin/python3.6', remote + '/remote_release.py', remote]
        if args.apply_migrations:
            command.append('--apply-migrations')
        subprocess.run(ssh + [' '.join(shlex.quote(value) for value in command)], check=True)
    # Run from outside the production host; its public-IP hairpin route is not
    # available. A failure here leaves the locally verified deployment installed.
    with urllib.request.urlopen('https://daeilfoundation.or.kr/api/health', timeout=15) as response:
        if response.status != 200 or json.load(response).get('status') != 'ok':
            raise RuntimeError('External API health check failed; inspect the installed release')
    print('External HTTPS API health verified from release client')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, subprocess.CalledProcessError) as error:
        print('Release stopped: ' + (str(error) if isinstance(error, ValueError) else 'command failed'))
        raise SystemExit(1)
