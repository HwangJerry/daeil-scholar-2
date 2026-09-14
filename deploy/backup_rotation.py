#!/usr/bin/python3.6
"""backup_rotation.py — Weekly full backups kept 28 days; daily prune of older backups and logs.

Runs as root from cron (see dflh-backup-rotation.cron). The newest complete
backup in BACKUP_ROOT is never deleted, so a failing weekly job cannot leave
the server without a backup; the status file then reports the overdue backup
and the erasure verifier keeps the backup check open for a person to review.
The status file holds only dates and counts, never personal data.
"""
import argparse
import gzip
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
from datetime import datetime, timedelta, timezone
from pathlib import Path

SERVICE = 'alumni-backend'
BACKUP_ROOT = Path('/app/backups')
RELEASE_ROOT = Path('/app/releases')
LEGACY_DUMPS = Path('/root/db-backups')
HTTPD_LOGS = Path('/var/log/httpd')
JOURNALD_DROPIN = Path('/etc/systemd/journald.conf.d/dflh-retention.conf')
HTTPD_LOGROTATE = Path('/etc/logrotate.d/httpd')
STATUS_PATH = Path('/app/backend/backup-rotation-status.json')
RETENTION_DAYS = 28
STAMP = re.compile(r'(\d{8}T\d{6}Z)')
ROTATED_LOG = re.compile(r'-\d{8}(\.gz)?$')
RELEASE_DATA = re.compile(r'^(database\.sql\.gz|data-checksums\.json|uploads-\d+\.tar\.gz)$')


def process_environment():
    pid = subprocess.check_output(['systemctl', 'show', SERVICE, '-p', 'MainPID']).decode().strip().split('=', 1)[1]
    if not pid.isdigit() or int(pid) <= 0:
        raise RuntimeError('backend must be running to read database settings')
    env = {}
    for item in Path('/proc/' + pid + '/environ').read_bytes().split(b'\0'):
        if b'=' in item:
            key, value = item.split(b'=', 1)
            env[key.decode()] = value.decode()
    return env


def mysqldump_args(env):
    for key in ('DB_USER', 'DB_PASSWORD', 'DB_NAME'):
        if not env.get(key):
            raise ValueError('missing database setting: ' + key)
    if not re.match(r'^[A-Za-z0-9_]+$', env['DB_NAME']):
        raise ValueError('unsupported database name')
    # Consistent InnoDB snapshot without blocking writes on the live site.
    return ['mysqldump', '--single-transaction', '--quick', '--routines', '--events', '--triggers', '--hex-blob',
            '--host=' + env.get('DB_HOST', '127.0.0.1'), '--port=' + env.get('DB_PORT', '3306'),
            '--user=' + env['DB_USER'], env['DB_NAME']]


def sha256(path):
    h = hashlib.sha256()
    with open(str(path), 'rb') as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b''):
            h.update(chunk)
    return h.hexdigest()


def created_at(path):
    match = STAMP.search(path.name)
    if match:
        return datetime.strptime(match.group(1), '%Y%m%dT%H%M%SZ').replace(tzinfo=timezone.utc)
    return datetime.fromtimestamp(path.stat().st_mtime, timezone.utc)


def complete_backups():
    if not BACKUP_ROOT.is_dir():
        return []
    dirs = [p for p in BACKUP_ROOT.iterdir() if p.is_dir() and not p.name.endswith('.partial') and (p / 'database.sql.gz').is_file()]
    return sorted(dirs, key=created_at)


def weekly(now):
    env = process_environment()
    dest = BACKUP_ROOT / ('weekly-' + now.strftime('%Y%m%dT%H%M%SZ'))
    work = dest.with_name(dest.name + '.partial')
    work.mkdir(mode=0o700, parents=True)
    dump = work / 'database.sql.gz'
    with gzip.open(str(dump), 'wb') as stream:
        proc = subprocess.Popen(mysqldump_args(env), env=dict(os.environ, MYSQL_PWD=env['DB_PASSWORD']), stdout=subprocess.PIPE)
        shutil.copyfileobj(proc.stdout, stream)
        proc.stdout.close()
        if proc.wait() != 0:
            raise RuntimeError('database dump failed')
    with gzip.open(str(dump), 'rb') as stream:
        if not stream.read(1024):
            raise RuntimeError('empty database dump')
    roots = sorted({env.get('UPLOAD_BASE_PATH', '/var/www/uploads'), env.get('ACCOUNT_ERASURE_LEGACY_ROOT', env.get('UPLOAD_LEGACY_PATH', '/var/www/legacy/files'))})
    for index, root in enumerate(roots):
        if os.path.isdir(root):
            with tarfile.open(str(work / ('uploads-%d.tar.gz' % index)), 'w:gz') as tar:
                tar.add(root, arcname='uploads', recursive=True)
    (work / 'data-checksums.json').write_text(json.dumps({p.name: sha256(p) for p in work.iterdir() if p.name.endswith('.gz')}, indent=2))
    os.rename(str(work), str(dest))
    return dest


def remove(path, removed, dry_run):
    removed.append(str(path))
    if dry_run:
        return
    if path.is_dir():
        shutil.rmtree(str(path))
    else:
        path.unlink()


def prune(now, dry_run=False):
    cutoff = now - timedelta(days=RETENTION_DAYS)
    removed = []
    backups = complete_backups()
    newest = backups[-1] if backups else None
    for backup in backups:
        if backup != newest and created_at(backup) < cutoff:
            remove(backup, removed, dry_run)
    if BACKUP_ROOT.is_dir():
        for partial in BACKUP_ROOT.glob('*.partial'):
            if created_at(partial) < now - timedelta(days=1):
                remove(partial, removed, dry_run)
    # Release backups keep their application snapshots for rollback; only
    # database and upload copies follow the personal-data retention period.
    for data in RELEASE_ROOT.glob('*/backup/*'):
        if RELEASE_DATA.match(data.name) and datetime.fromtimestamp(data.stat().st_mtime, timezone.utc) < cutoff:
            remove(data, removed, dry_run)
    for dump in LEGACY_DUMPS.glob('*'):
        if dump.is_file() and datetime.fromtimestamp(dump.stat().st_mtime, timezone.utc) < cutoff:
            remove(dump, removed, dry_run)
    for log in HTTPD_LOGS.glob('*'):
        if log.is_file() and ROTATED_LOG.search(log.name) and datetime.fromtimestamp(log.stat().st_mtime, timezone.utc) < cutoff:
            remove(log, removed, dry_run)
    return removed


def oldest_backup_data(now):
    ages = [created_at(b) for b in complete_backups()]
    ages += [datetime.fromtimestamp(p.stat().st_mtime, timezone.utc) for p in RELEASE_ROOT.glob('*/backup/*') if RELEASE_DATA.match(p.name)]
    ages += [datetime.fromtimestamp(p.stat().st_mtime, timezone.utc) for p in LEGACY_DUMPS.glob('*') if p.is_file()]
    return min(ages) if ages else None


def configured_days(path, pattern):
    try:
        match = re.search(pattern, path.read_text())
    except OSError:
        return None
    return int(match.group(1)) if match else None


def write_status(now, removed):
    backups = complete_backups()
    oldest = oldest_backup_data(now)
    status = {
        'checkedAt': now.strftime('%Y-%m-%dT%H:%M:%SZ'),
        'retentionDays': RETENTION_DAYS,
        'newestBackupAt': created_at(backups[-1]).strftime('%Y-%m-%dT%H:%M:%SZ') if backups else None,
        'oldestBackupAgeDays': round((now - oldest).total_seconds() / 86400, 2) if oldest else None,
        'journalRetentionDays': configured_days(JOURNALD_DROPIN, r'MaxRetentionSec=(\d+)day'),
        'httpLogMaxAgeDays': configured_days(HTTPD_LOGROTATE, r'maxage\s+(\d+)'),
        'removedLastRun': len(removed),
    }
    STATUS_PATH.parent.mkdir(parents=True, exist_ok=True)
    temporary = STATUS_PATH.with_name(STATUS_PATH.name + '.tmp')
    temporary.write_text(json.dumps(status, indent=2))
    os.chmod(str(temporary), 0o644)
    os.replace(str(temporary), str(STATUS_PATH))
    return status


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=['weekly', 'prune'])
    parser.add_argument('--dry-run', action='store_true')
    args = parser.parse_args(argv)
    now = datetime.now(timezone.utc)
    if args.action == 'weekly':
        print('weekly backup: ' + str(weekly(now)))
    removed = prune(now, args.dry_run)
    for path in removed:
        print(('would remove: ' if args.dry_run else 'removed: ') + path)
    if not args.dry_run:
        print(json.dumps(write_status(now, removed)))


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, ValueError, OSError, subprocess.CalledProcessError) as error:
        print('backup rotation failed: %s' % error, file=sys.stderr)
        raise SystemExit(1)
