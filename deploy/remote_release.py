#!/usr/bin/env python3
"""Production release transaction. Invoked only by an explicit deployment command.

Python 3.6 compatible. Never prints service environment values. Database restore
is deliberately manual; application rollback must not overwrite newer records.
"""
import argparse
from contextlib import contextmanager
import fcntl
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import time
import tempfile
import urllib.request

SERVICE = 'alumni-backend'
HTTPD = 'httpd'
GATES = ('ACCOUNT_ERASURE_REQUESTS_ENABLED', 'ACCOUNT_ERASURE_WORKER_ENABLED', 'PRIVACY_RETENTION_ENABLED')
ROLLOUT_ENV = Path('/app/backend/release-rollout.env')
ROLLOUT_UNIT = Path('/etc/systemd/system/alumni-backend.service.d/90-release-rollout.conf')
HTTPD_CONFIG = Path('/etc/httpd/conf.d/alumni.conf')
SHIMS = ('_set_docroot.php', '_legacy_docroot.php', '_legacy_url_rewriter.php')


@contextmanager
def release_lock(path=Path('/var/lock/daeil-release.lock')):
    # One host-wide lock for deployment, activation and pause. Never unlink the
    # lock file: replacing its inode would let a second process bypass the lock.
    fd = os.open(str(path), os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise RuntimeError('another release operation is running; retry after it finishes')
        yield
    finally:
        os.close(fd)


def atomic_text(path, text):
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary = tempfile.mkstemp(prefix='.' + path.name + '-', dir=str(path.parent))
    try:
        with os.fdopen(fd, 'w') as stream:
            stream.write(text)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, str(path))
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)


def output(args):
    return subprocess.check_output(args).decode().strip()


def process_environment():
    pid = output(['systemctl', 'show', SERVICE, '-p', 'MainPID']).split('=', 1)[1]
    if not pid.isdigit() or int(pid) <= 0:
        raise RuntimeError('backend must be running for preflight')
    env = {}
    for item in Path('/proc/' + pid + '/environ').read_bytes().split(b'\0'):
        if b'=' in item:
            key, value = item.split(b'=', 1)
            env[key.decode()] = value.decode()
    return env


def sha256(path):
    h = hashlib.sha256()
    with open(str(path), 'rb') as stream:
        for data in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(data)
    return h.hexdigest()


def unpack(archive, destination):
    destination.mkdir(mode=0o700)
    with tarfile.open(str(archive), 'r:gz') as tar:
        for member in tar.getmembers():
            path = Path(member.name)
            if member.name != 'manifest.json' and not member.name.startswith('artifacts/'):
                raise ValueError('unexpected archive entry')
            if path.is_absolute() or '..' in path.parts or not member.isfile():
                raise ValueError('unsafe archive entry')
            target = destination / path
            if target.exists():
                raise ValueError('duplicate archive entry')
            target.parent.mkdir(parents=True, exist_ok=True)
            with open(str(target), 'wb') as stream:
                shutil.copyfileobj(tar.extractfile(member), stream)
            target.chmod(0o600)
    manifest = json.loads((destination / 'manifest.json').read_text())
    actual = {p.relative_to(destination / 'artifacts').as_posix() for p in (destination / 'artifacts').rglob('*') if p.is_file()}
    if actual != set(manifest['files']):
        raise ValueError('artifact inventory mismatch')
    for name, expected in manifest['files'].items():
        if sha256(destination / 'artifacts' / name) != expected:
            raise ValueError('artifact hash mismatch')
    return manifest


def mysql_args(env, executable='mysql'):
    for key in ('DB_USER', 'DB_PASSWORD', 'DB_NAME'):
        if not env.get(key):
            raise ValueError('missing database setting: ' + key)
    if not re.match(r'^[A-Za-z0-9_]+$', env['DB_NAME']):
        raise ValueError('database name requires an explicit supported identifier')
    return [executable, '--host=' + env.get('DB_HOST', '127.0.0.1'),
            '--port=' + env.get('DB_PORT', '3306'), '--user=' + env['DB_USER'], env['DB_NAME']]


def db_env(env):
    return dict(os.environ, MYSQL_PWD=env['DB_PASSWORD'])


def pending_migrations(payload, manifest, env):
    data = subprocess.check_output(mysql_args(env) + ['-BN', '-e', 'SELECT filename,sha256 FROM _migration_history ORDER BY filename'], env=db_env(env)).decode()
    history = dict(line.split('\t', 1) for line in data.splitlines())
    migrations = sorted((payload / 'artifacts/migrations').glob('*.sql'))
    pending = []
    for path in migrations:
        if not re.match(r'^\d{3}_[a-zA-Z0-9_]+\.sql$', path.name):
            raise ValueError('invalid migration filename')
        if path.name in history:
            if history[path.name] != sha256(path):
                raise ValueError('applied migration hash mismatch: ' + path.name)
        else:
            pending.append(path)
    if set(history) - {p.name for p in migrations}:
        raise ValueError('production has migrations absent from candidate')
    if 'backend' not in manifest['components'] and manifest['minimum_web_migration'] not in history:
        raise ValueError('deploy the matching backend/schema before these web assets')
    return pending if 'backend' in manifest['components'] else []


def validate_migration_storage(pending, env):
    if not any(path.name == '058_convert_erasure_tables_to_innodb.sql' for path in pending):
        return
    query = "SHOW GLOBAL VARIABLES WHERE Variable_name IN ('innodb_file_format','innodb_large_prefix','innodb_file_per_table','innodb_page_size')"
    data = subprocess.check_output(mysql_args(env) + ['-BN', '-e', query], env=db_env(env)).decode()
    settings = dict(line.split('\t', 1) for line in data.splitlines())
    required = {'innodb_file_format': 'Barracuda', 'innodb_large_prefix': 'ON',
                'innodb_file_per_table': 'ON', 'innodb_page_size': '16384'}
    if any(settings.get(key, '').lower() != value.lower() for key, value in required.items()):
        raise ValueError('migration 058 requires verified Barracuda/large-prefix/file-per-table with 16KB pages; no services stopped')


def validate_activation(binary, env, test_user=0):
    required = ('ALLOWED_ORIGIN', 'SITE_BASE_URL', 'DB_USER', 'DB_PASSWORD', 'DB_NAME',
                'KAKAO_CLIENT_ID', 'KAKAO_CLIENT_SECRET', 'KAKAO_REDIRECT_URI', 'JWT_SECRET',
                'UPLOAD_LEGACY_PATH', 'EASYPAY_IMMEDIATELY_MALL_ID', 'EASYPAY_PROFILE_MALL_ID',
                'EASYPAY_GW_URL', 'EASYPAY_BIN_BASE', 'EASYPAY_RETURN_BASE_URL',
                'SMTP_HOST', 'SMTP_USER', 'SMTP_PASSWORD', 'VISIT_IP_SALT', 'ENV')
    for key in required:
        if not env.get(key, '').strip():
            raise ValueError('missing production setting: ' + key)
    for key, placeholder in {'JWT_SECRET': 'change-me-in-production', 'EASYPAY_GW_URL': 'testgw.easypay.co.kr', 'ENV': 'dev'}.items():
        if env.get(key) == placeholder:
            raise ValueError('placeholder production setting: ' + key)
    checked = dict(env)
    checked.update({key: 'true' for key in GATES})
    checked['ACCOUNT_ERASURE_TEST_USER_SEQ'] = str(test_user)
    checked['PRIVACY_RETENTION_ENABLED'] = 'false' if test_user else 'true'
    binary.chmod(0o755)
    run([str(binary), '--check-release-config'], env=checked, stdout=subprocess.DEVNULL)
    if env.get('SENTRY_AUTH_TOKEN') and (env.get('SENTRY_IOS_PROJECT') != 'daeil-ios-release' or env.get('SENTRY_ANDROID_PROJECT') != 'daeil-android-release'):
        raise ValueError('Sentry monitoring must use the release projects')


def set_rollout(enabled, test_user=0):
    ROLLOUT_ENV.parent.mkdir(parents=True, exist_ok=True)
    values = {key: enabled for key in GATES}
    if test_user:
        values['PRIVACY_RETENTION_ENABLED'] = False
    atomic_text(ROLLOUT_ENV, ''.join(key + '=' + ('true' if value else 'false') + '\n' for key, value in values.items()) + 'ACCOUNT_ERASURE_TEST_USER_SEQ=' + str(test_user) + '\n')
    ROLLOUT_ENV.chmod(0o600)
    ROLLOUT_UNIT.parent.mkdir(parents=True, exist_ok=True)
    atomic_text(ROLLOUT_UNIT, '[Service]\nEnvironmentFile=' + str(ROLLOUT_ENV) + '\n')
    run(['systemctl', 'daemon-reload'])


def healthy(env):
    url = 'http://127.0.0.1:' + env.get('SERVER_PORT', '8080') + '/api/health'
    for _ in range(20):
        try:
            with urllib.request.urlopen(url, timeout=2) as response:
                if response.status == 200 and json.load(response).get('status') == 'ok':
                    return
        except Exception:
            pass
        time.sleep(1)
    raise RuntimeError('backend health check failed')


def snapshot(path, backup, records):
    target = backup / ('item-' + str(len(records)))
    record = {'path': str(path), 'backup': str(target), 'exists': path.exists()}
    if path.is_symlink():
        raise ValueError('deployment target must not be a symlink: ' + str(path))
    if path.exists():
        if path.is_dir():
            shutil.copytree(str(path), str(target))
        else:
            shutil.copy2(str(path), str(target))
            if sha256(path) != sha256(target):
                raise RuntimeError('backup hash mismatch')
    records.append(record)


def restore(records):
    for record in reversed(records):
        path = Path(record['path'])
        if path.is_dir():
            shutil.rmtree(str(path))
        elif path.exists():
            path.unlink()
        if record['exists']:
            saved = Path(record['backup'])
            path.parent.mkdir(parents=True, exist_ok=True)
            if saved.is_dir():
                shutil.copytree(str(saved), str(path))
            else:
                shutil.copy2(str(saved), str(path))


def database_backup(env, backup):
    # One database is dumped while Apache and the backend are stopped. Lock
    # its tables without requiring the server-wide RELOAD privilege.
    destination = backup / 'database.sql.gz'
    with gzip.open(str(destination), 'wb') as stream:
        proc = subprocess.Popen(mysql_args(env, 'mysqldump') + ['--routines', '--events', '--triggers', '--hex-blob', '--lock-tables'], env=db_env(env), stdout=subprocess.PIPE)
        shutil.copyfileobj(proc.stdout, stream)
        proc.stdout.close()
        if proc.wait() != 0:
            raise RuntimeError('database backup failed')
    with gzip.open(str(destination), 'rb') as stream:
        total = 0
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            total += len(chunk)
    if total == 0:
        raise RuntimeError('empty database backup')
    for index, root in enumerate(sorted({env.get('UPLOAD_BASE_PATH', '/var/www/uploads'), env.get('ACCOUNT_ERASURE_LEGACY_ROOT', env.get('UPLOAD_LEGACY_PATH', '/var/www/legacy/files'))})):
        with tarfile.open(str(backup / ('uploads-' + str(index) + '.tar.gz')), 'w:gz') as tar:
            tar.add(root, arcname='uploads', recursive=True)
    (backup / 'data-checksums.json').write_text(json.dumps({p.name: sha256(p) for p in backup.iterdir() if p.name.endswith('.gz')}, indent=2))


def apply_migrations(paths, env):
    for path in paths:
        with open(str(path), 'rb') as stream:
            run(mysql_args(env), env=db_env(env), stdin=stream, stdout=subprocess.DEVNULL)
        statement = "INSERT INTO _migration_history (filename,sha256) VALUES ('{}','{}')".format(path.name, sha256(path))
        run(mysql_args(env) + ['-e', statement], env=db_env(env), stdout=subprocess.DEVNULL)
        print('Applied ' + path.name, flush=True)


def install_file(source, destination, mode):
    destination.parent.mkdir(parents=True, exist_ok=True)
    staged = destination.with_name(destination.name + '.release-new')
    shutil.copyfile(str(source), str(staged))
    staged.chmod(mode)
    os.replace(str(staged), str(destination))


def install_web(source, destination):
    # Preserve prior content-hashed chunks so already-open pages still load them.
    destination.mkdir(parents=True, exist_ok=True)
    destination.chmod(0o755)
    for directory in source.rglob('*'):
        if directory.is_dir():
            target = destination / directory.relative_to(source)
            target.mkdir(parents=True, exist_ok=True)
            target.chmod(0o755)
    for path in source.rglob('*'):
        if path.is_file() and path.name != 'index.html':
            install_file(path, destination / path.relative_to(source), 0o644)
    install_file(source / 'index.html', destination / 'index.html', 0o644)


def deploy(root, apply_schema):
    payload = root / 'payload'
    manifest = unpack(root / 'bundle.tar.gz', payload)
    env = process_environment()
    pending = pending_migrations(payload, manifest, env)
    if pending and not apply_schema:
        raise ValueError('pending schema changes require --apply-migrations after reviewing the plan')
    validate_migration_storage(pending, env)
    artifacts = payload / 'artifacts'
    backend = 'backend' in manifest['components']
    if backend:
        validate_activation(artifacts / 'backend/server', env)
    # Conservative admission check; this is not a promise of zero downtime.
    db_bytes = int(subprocess.check_output(mysql_args(env) + ['-BN', '-e', 'SELECT COALESCE(SUM(DATA_LENGTH+INDEX_LENGTH),0) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()'], env=db_env(env)).decode().strip())
    roots = ['/app/backend', '/var/www/app', '/var/www/admin']
    if backend:
        roots += list({env.get('UPLOAD_BASE_PATH', '/var/www/uploads'), env.get('ACCOUNT_ERASURE_LEGACY_ROOT', env.get('UPLOAD_LEGACY_PATH', '/var/www/legacy/files'))})
    footprint = sum(int(output(['du', '-sb', path]).split()[0]) for path in roots if Path(path).exists())
    required = footprint * 2 + db_bytes * 4 + 64 * 1024 * 1024
    if min(shutil.disk_usage(str(root)).free, shutil.disk_usage('/var/www').free, shutil.disk_usage('/var/lib/mysql').free) < required:
        raise ValueError('insufficient room for release and recovery data')
    backup = root / 'backup'
    backup.mkdir(mode=0o700)
    records = []
    targets = [HTTPD_CONFIG] + [Path('/var/www/html') / p for p in SHIMS]
    if backend:
        targets += [Path('/app/backend/server'), Path('/app/backend/backfill'), ROLLOUT_ENV, ROLLOUT_UNIT]
    targets += [Path('/var/www') / ('app' if c == 'frontend' else 'admin') for c in manifest['components'] if c != 'backend']
    for path in targets:
        snapshot(path, backup, records)
    (backup / 'application-restore.json').write_text(json.dumps(records, indent=2))
    stopped = False
    try:
        run(['systemctl', 'stop', HTTPD])
        stopped = True
        if backend:
            run(['systemctl', 'stop', SERVICE])
            database_backup(env, backup)
            apply_migrations(pending, env)
            set_rollout(False)
            for binary in ('server', 'backfill'):
                install_file(artifacts / 'backend' / binary, Path('/app/backend') / binary, 0o755)
            run(['systemctl', 'start', SERVICE])
            healthy(env)
            live_env = process_environment()
            if any(live_env.get(key) != 'false' for key in GATES):
                raise RuntimeError('erasure jobs were not paused by the rollout override')
            pid = output(['systemctl', 'show', SERVICE, '-p', 'MainPID']).split('=', 1)[1]
            if sha256(Path('/proc') / pid / 'exe') != manifest['files']['backend/server']:
                raise RuntimeError('running binary differs from verified artifact')
        for component in ('frontend', 'admin'):
            if component in manifest['components']:
                install_web(artifacts / component, Path('/var/www') / ('app' if component == 'frontend' else 'admin'))
        install_file(artifacts / 'deploy/httpd-alumni.conf', HTTPD_CONFIG, 0o644)
        for shim in SHIMS:
            install_file(artifacts / 'deploy' / shim, Path('/var/www/html') / shim, 0o644)
        run(['httpd', '-t'])
        run(['systemctl', 'start', HTTPD])
        healthy(env)
        origin = env.get('SITE_BASE_URL', '').rstrip('/')
        with urllib.request.urlopen(origin + '/api/health', timeout=10) as response:
            if response.status != 200 or json.load(response).get('status') != 'ok':
                raise RuntimeError('public API health check failed')
        (root / 'result.json').write_text(json.dumps({'status': 'DEPLOYED_ERASURE_PAUSED' if backend else 'DEPLOYED', 'commit': manifest['commit'], 'backup': str(backup)}, indent=2))
        print('Deployment verified. Erasure activation is a separate operation; recovery data: ' + str(backup))
    except BaseException:
        if stopped:
            run(['systemctl', 'stop', HTTPD])
            if backend:
                run(['systemctl', 'stop', SERVICE])
            restore(records)
            if backend:
                set_rollout(False)
            run(['systemctl', 'daemon-reload'])
            if backend:
                run(['systemctl', 'start', SERVICE])
            run(['systemctl', 'start', HTTPD])
            print('Application files restored. Database schema was NOT reverted; inspect migration history before retrying.', flush=True)
        raise


def change_rollout(root, enabled, test_user=0):
    # Emergency pause does not depend on the DB, a live backend or old release
    # metadata. Stop the worker first so failed configuration/restart stays safe.
    if not enabled:
        try:
            try:
                run(['systemctl', 'stop', SERVICE])
            finally:
                set_rollout(False)
            run(['systemctl', 'start', SERVICE])
            verify_rollout(False, 0)
        except BaseException:
            run(['systemctl', 'stop', SERVICE])
            raise
        print('Rollout verified: paused (service health must be checked separately)')
        return
    result = json.loads((root / 'result.json').read_text())
    if not result['status'].startswith('DEPLOYED'):
        raise ValueError('release has not passed deployment verification')
    manifest = json.loads((root / 'payload/manifest.json').read_text())
    if sha256(Path('/app/backend/server')) != manifest['files']['backend/server']:
        raise ValueError('release binary is no longer installed')
    env = process_environment()
    validate_activation(Path('/app/backend/server'), env, test_user)
    try:
        set_rollout(True, test_user)
        run(['systemctl', 'restart', SERVICE])
        healthy(env)
        verify_rollout(True, test_user)
    except BaseException:
        # Never restore a previously enabled worker after failed activation.
        try:
            set_rollout(False)
        finally:
            run(['systemctl', 'stop', SERVICE])
        raise
    print('Rollout verified: ' + ('test account only' if test_user else 'all accounts'))


def verify_rollout(enabled, test_user):
    current = process_environment()
    expected = {'ACCOUNT_ERASURE_REQUESTS_ENABLED': enabled, 'ACCOUNT_ERASURE_WORKER_ENABLED': enabled,
                'PRIVACY_RETENTION_ENABLED': enabled and not test_user}
    if any(current.get(key) != ('true' if value else 'false') for key, value in expected.items()) or current.get('ACCOUNT_ERASURE_TEST_USER_SEQ') != str(test_user):
        raise RuntimeError('rollout environment did not take effect')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('directory')
    parser.add_argument('--apply-migrations', action='store_true')
    action = parser.add_mutually_exclusive_group()
    action.add_argument('--activate-test-user', type=int)
    action.add_argument('--activate-all-users', action='store_true')
    action.add_argument('--pause-erasure', action='store_true')
    args = parser.parse_args()
    os.umask(0o077)
    try:
        with release_lock():
            if args.activate_test_user is not None:
                if args.activate_test_user <= 0:
                    raise ValueError('test user must be a positive disposable account ID')
                change_rollout(Path(args.directory), True, args.activate_test_user)
            elif args.activate_all_users or args.pause_erasure:
                change_rollout(Path(args.directory), args.activate_all_users)
            else:
                deploy(Path(args.directory), args.apply_migrations)
    except Exception as error:
        # Do not echo subprocess commands or service environments.
        print('Release stopped: ' + (str(error) if isinstance(error, (ValueError, RuntimeError)) else type(error).__name__))
        raise SystemExit(1)
