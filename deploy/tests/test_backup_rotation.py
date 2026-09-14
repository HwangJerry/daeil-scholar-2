# test_backup_rotation.py — 28-day pruning keeps the newest backup and application snapshots.
import json
import os
from datetime import datetime, timedelta, timezone
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import backup_rotation as rotation

NOW = datetime(2026, 9, 14, 3, 0, tzinfo=timezone.utc)


def aged(path, days):
    stamp = (NOW - timedelta(days=days)).timestamp()
    os.utime(str(path), (stamp, stamp))


class PruneTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        root = Path(self.tmp.name)
        for name in ('BACKUP_ROOT', 'RELEASE_ROOT', 'LEGACY_DUMPS', 'HTTPD_LOGS'):
            path = root / name.lower()
            path.mkdir()
            setattr(rotation, name, path)
        rotation.STATUS_PATH = root / 'status.json'
        rotation.JOURNALD_DROPIN = root / 'journald.conf'
        rotation.JOURNALD_DROPIN.write_text('[Journal]\nMaxRetentionSec=28day\n')
        rotation.HTTPD_LOGROTATE = root / 'httpd'
        rotation.HTTPD_LOGROTATE.write_text('/var/log/httpd/*log {\n    maxage 28\n}\n')

    def tearDown(self):
        self.tmp.cleanup()

    def backup(self, days):
        stamp = (NOW - timedelta(days=days)).strftime('%Y%m%dT%H%M%SZ')
        path = rotation.BACKUP_ROOT / ('weekly-' + stamp)
        path.mkdir()
        (path / 'database.sql.gz').write_bytes(b'x')
        return path

    def test_old_backups_go_but_newest_always_stays(self):
        old, recent = self.backup(35), self.backup(3)
        rotation.prune(NOW)
        self.assertFalse(old.exists())
        self.assertTrue(recent.exists())
        only = self.backup(60)
        recent_removed = rotation.BACKUP_ROOT / 'none'
        for path in rotation.complete_backups():
            if path != only:
                import shutil
                shutil.rmtree(str(path))
        rotation.prune(NOW)
        self.assertTrue(only.exists(), 'the only remaining backup must never be pruned')
        self.assertFalse(recent_removed.exists())

    def test_release_data_expires_but_application_snapshot_stays(self):
        backup = rotation.RELEASE_ROOT / '20260801T000000Z-abc' / 'backup'
        backup.mkdir(parents=True)
        for name in ('database.sql.gz', 'uploads-0.tar.gz', 'data-checksums.json', 'item-0', 'application-restore.json'):
            (backup / name).write_bytes(b'x')
            aged(backup / name, 40)
        rotation.prune(NOW)
        self.assertEqual(sorted(p.name for p in backup.iterdir()), ['application-restore.json', 'item-0'])

    def test_rotated_logs_and_legacy_dumps_expire(self):
        for name, days in (('access_log', 400), ('access_log-20190101', 400), ('access_log-20260910', 4)):
            (rotation.HTTPD_LOGS / name).write_bytes(b'x')
            aged(rotation.HTTPD_LOGS / name, days)
        (rotation.LEGACY_DUMPS / 'old.sql').write_bytes(b'x')
        aged(rotation.LEGACY_DUMPS / 'old.sql', 30)
        rotation.prune(NOW)
        self.assertEqual(sorted(p.name for p in rotation.HTTPD_LOGS.iterdir()), ['access_log', 'access_log-20260910'])
        self.assertFalse((rotation.LEGACY_DUMPS / 'old.sql').exists())

    def test_status_reports_ages_and_log_settings_without_paths(self):
        self.backup(2)
        status = rotation.write_status(NOW, [])
        self.assertEqual(status['retentionDays'], 28)
        self.assertEqual(status['journalRetentionDays'], 28)
        self.assertEqual(status['httpLogMaxAgeDays'], 28)
        self.assertAlmostEqual(status['oldestBackupAgeDays'], 2.0, places=1)
        written = rotation.STATUS_PATH.read_text()
        self.assertNotIn(self.tmp.name, written)
        self.assertEqual(json.loads(written)['newestBackupAt'], (NOW - timedelta(days=2)).strftime('%Y-%m-%dT%H:%M:%SZ'))


if __name__ == '__main__':
    unittest.main()
