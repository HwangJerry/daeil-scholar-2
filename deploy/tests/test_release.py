# test_release.py — Artifact integrity, deployment ordering and failure recovery.
import json
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import release_bundle as bundle
import remote_release as remote


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()

    def candidate(self):
        files = ['backend/server', 'backend/backfill', 'frontend/index.html', 'frontend/assets/new.js',
                 'deploy/remote_release.py', 'deploy/httpd-alumni.conf',
                 *['deploy/' + name for name in remote.SHIMS], 'migrations/062_create_profile_file_history.sql']
        for name in files:
            path = self.root / 'artifacts' / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('synthetic ' + name)
        manifest = {'format': 1, 'commit': 'a' * 40, 'components': ['backend', 'frontend'], 'patch_mode': 'false',
                    'minimum_web_migration': '062_create_profile_file_history.sql',
                    'files': {name: bundle.digest(self.root / 'artifacts' / name) for name in files}}
        (self.root / 'manifest.json').write_text(json.dumps(manifest))
        return manifest

    def test_verified_inventory_rejects_modified_extra_and_symlink_files(self):
        self.candidate()
        bundle.verify(self.root)
        path = self.root / 'artifacts/backend/server'
        original = path.read_text()
        path.write_text('changed')
        with self.assertRaises(ValueError):
            bundle.verify(self.root)
        path.write_text(original)
        extra = self.root / 'artifacts/backend/unreviewed'
        extra.write_text('extra')
        with self.assertRaises(ValueError):
            bundle.verify(self.root)
        extra.unlink()
        external = self.root / 'external'
        path.rename(external)
        path.symlink_to(external)
        with self.assertRaises(ValueError):
            bundle.verify(self.root)

    def test_remote_unpack_rejects_traversal_before_extraction(self):
        source = self.root / 'source'
        source.write_text('not allowed')
        archive = self.root / 'bad.tar.gz'
        with tarfile.open(archive, 'w:gz') as tar:
            tar.add(source, arcname='../escape')
        with self.assertRaises(ValueError):
            remote.unpack(archive, self.root / 'payload')
        self.assertFalse((self.root.parent / 'escape').exists())

    def test_web_install_keeps_old_chunks_and_readable_directories(self):
        source, target = self.root / 'source', self.root / 'target'
        (source / 'assets').mkdir(parents=True)
        target.mkdir()
        (source / 'index.html').write_text('new index')
        (source / 'assets/new.js').write_text('new chunk')
        (target / 'old.js').write_text('old chunk')
        remote.install_web(source, target)
        self.assertEqual((target / 'old.js').read_text(), 'old chunk')
        self.assertEqual((target / 'index.html').read_text(), 'new index')
        self.assertEqual((target / 'assets').stat().st_mode & 0o777, 0o755)

    def test_snapshot_restores_exact_app_files_and_absent_override(self):
        backup = self.root / 'backup'
        backup.mkdir()
        app = self.root / 'app'
        app.mkdir()
        (app / 'old').write_text('original')
        override = self.root / 'rollout.env'
        records = []
        remote.snapshot(app, backup, records)
        remote.snapshot(override, backup, records)
        (app / 'old').write_text('changed')
        (app / 'new').write_text('new')
        override.write_text('paused')
        remote.restore(records)
        self.assertEqual((app / 'old').read_text(), 'original')
        self.assertFalse((app / 'new').exists())
        self.assertFalse(override.exists())

    def test_migration_failure_restores_app_and_never_installs_new_binary(self):
        manifest = self.candidate()
        events = []
        env = {'SERVER_PORT': '8080', 'DB_USER': 'test', 'DB_PASSWORD': 'synthetic', 'DB_NAME': 'test'}
        def command(args, **kwargs):
            events.append(' '.join(args))
        def fail_migration(*args):
            events.append('migration failed')
            raise RuntimeError('synthetic migration failure')
        with patch.object(remote, 'unpack', return_value=manifest), \
             patch.object(remote, 'process_environment', return_value=env), \
             patch.object(remote, 'pending_migrations', return_value=[Path('058_test.sql')]), \
             patch.object(remote, 'validate_activation'), \
             patch.object(remote.subprocess, 'check_output', return_value=b'1'), \
             patch.object(remote, 'output', return_value='1'), \
             patch.object(remote.shutil, 'disk_usage', return_value=type('Disk', (), {'free': 10**12})()), \
             patch.object(remote, 'snapshot'), \
             patch.object(remote, 'run', side_effect=command), \
             patch.object(remote, 'database_backup', side_effect=lambda *a: events.append('backup verified')), \
             patch.object(remote, 'set_rollout'), \
             patch.object(remote, 'apply_migrations', side_effect=fail_migration), \
             patch.object(remote, 'restore', side_effect=lambda *a: events.append('restore apps')), \
             patch.object(remote, 'install_file') as install:
            with self.assertRaises(RuntimeError):
                remote.deploy(self.root, True)
            install.assert_not_called()
        self.assertLess(events.index('systemctl stop alumni-backend'), events.index('backup verified'))
        self.assertLess(events.index('backup verified'), events.index('migration failed'))
        self.assertLess(events.index('migration failed'), events.index('restore apps'))
        self.assertIn('systemctl start alumni-backend', events)
        self.assertIn('systemctl start httpd', events)

    def test_missing_schema_approval_or_settings_never_stops_services(self):
        manifest = self.candidate()
        with patch.object(remote, 'unpack', return_value=manifest), \
             patch.object(remote, 'process_environment', return_value={}), \
             patch.object(remote, 'pending_migrations', return_value=[Path('058_test.sql')]), \
             patch.object(remote, 'run') as command:
            with self.assertRaises(ValueError):
                remote.deploy(self.root, False)
            command.assert_not_called()

    def test_backend_health_failure_rolls_back_before_web_install(self):
        manifest = self.candidate()
        events = []
        with patch.object(remote, 'unpack', return_value=manifest), \
             patch.object(remote, 'process_environment', return_value={'DB_USER': 'test', 'DB_PASSWORD': 'synthetic', 'DB_NAME': 'test'}), \
             patch.object(remote, 'pending_migrations', return_value=[]), \
             patch.object(remote, 'validate_activation'), \
             patch.object(remote.subprocess, 'check_output', return_value=b'1'), \
             patch.object(remote, 'output', return_value='1'), \
             patch.object(remote.shutil, 'disk_usage', return_value=type('Disk', (), {'free': 10**12})()), \
             patch.object(remote, 'snapshot'), patch.object(remote, 'database_backup'), \
             patch.object(remote, 'run'), patch.object(remote, 'apply_migrations'), \
             patch.object(remote, 'set_rollout', side_effect=lambda enabled: events.append(('gates', enabled))), \
             patch.object(remote, 'install_file', side_effect=lambda *a: events.append(('install', a[1].name))), \
             patch.object(remote, 'healthy', side_effect=RuntimeError('synthetic unhealthy backend')), \
             patch.object(remote, 'restore', side_effect=lambda *a: events.append(('restore', True))), \
             patch.object(remote, 'install_web') as web:
            with self.assertRaises(RuntimeError):
                remote.deploy(self.root, True)
            web.assert_not_called()
        self.assertLess(events.index(('gates', False)), events.index(('install', 'server')))
        self.assertGreater(events.index(('restore', True)), events.index(('install', 'server')))

    def test_test_account_activation_keeps_unrelated_retention_paused(self):
        env_file = self.root / 'rollout.env'
        unit_file = self.root / 'service.d/90-release.conf'
        with patch.object(remote, 'ROLLOUT_ENV', env_file), patch.object(remote, 'ROLLOUT_UNIT', unit_file), patch.object(remote, 'run'):
            remote.set_rollout(True, 42)
        settings = dict(line.split('=', 1) for line in env_file.read_text().splitlines())
        self.assertEqual(settings['ACCOUNT_ERASURE_TEST_USER_SEQ'], '42')
        self.assertEqual(settings['ACCOUNT_ERASURE_REQUESTS_ENABLED'], 'true')
        self.assertEqual(settings['ACCOUNT_ERASURE_WORKER_ENABLED'], 'true')
        self.assertEqual(settings['PRIVACY_RETENTION_ENABLED'], 'false')
        self.assertEqual(env_file.stat().st_mode & 0o777, 0o600)


if __name__ == '__main__':
    unittest.main()
