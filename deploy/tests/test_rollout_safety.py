# test_rollout_safety.py — Fail-closed pause, activation recovery and process exclusion.
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import remote_release as remote
import release_bundle as bundle


class RolloutSafetyTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.env = self.root / 'rollout.env'
        self.unit = self.root / 'service.conf'
        self.env.write_text(''.join(key + '=true\n' for key in remote.GATES))
        for name, value in [('ROLLOUT_ENV', self.env), ('ROLLOUT_UNIT', self.unit)]:
            mock = patch.object(remote, name, value)
            mock.start()
            self.addCleanup(mock.stop)

    def test_pause_with_backend_down_and_db_unhealthy(self):
        def current():
            return dict(line.split('=', 1) for line in self.env.read_text().splitlines())
        with patch.object(remote, 'run') as run, patch.object(remote, 'process_environment', side_effect=current), patch.object(remote, 'healthy', side_effect=RuntimeError('DB unavailable')) as health:
            remote.change_rollout(self.root, False)
            health.assert_not_called()
        self.assertEqual(run.call_args_list[0].args[0], ['systemctl', 'stop', remote.SERVICE])
        self.assertTrue(all(current()[key] == 'false' for key in remote.GATES))

    def test_failed_pause_never_restores_enabled_flags(self):
        with patch.object(remote, 'run') as run, patch.object(remote, 'process_environment', side_effect=RuntimeError('backend down')):
            with self.assertRaises(RuntimeError):
                remote.change_rollout(self.root, False)
        self.assertNotIn('=true', self.env.read_text())
        self.assertEqual(run.call_args_list[-1].args[0], ['systemctl', 'stop', remote.SERVICE])

    def test_stop_failure_still_persists_disabled_settings(self):
        def command(args, **kwargs):
            if args == ['systemctl', 'stop', remote.SERVICE]:
                raise RuntimeError('stop failed')
        with patch.object(remote, 'run', side_effect=command):
            with self.assertRaises(RuntimeError):
                remote.change_rollout(self.root, False)
        self.assertNotIn('=true', self.env.read_text())

    def test_failed_activation_stops_service_and_persists_pause(self):
        (self.root / 'result.json').write_text(json.dumps({'status': 'DEPLOYED_ERASURE_PAUSED'}))
        (self.root / 'payload').mkdir()
        (self.root / 'payload/manifest.json').write_text(json.dumps({'files': {'backend/server': 'hash'}}))
        with patch.object(remote, 'sha256', return_value='hash'), patch.object(remote, 'process_environment', return_value={}), patch.object(remote, 'validate_activation'), patch.object(remote, 'healthy', side_effect=RuntimeError('DB unavailable')), patch.object(remote, 'run') as run:
            with self.assertRaises(RuntimeError):
                remote.change_rollout(self.root, True, 42)
        self.assertNotIn('=true', self.env.read_text())
        commands = [call.args[0] for call in run.call_args_list]
        self.assertEqual(commands.count(['systemctl', 'restart', remote.SERVICE]), 1)
        self.assertIn(['systemctl', 'stop', remote.SERVICE], commands)

    def test_activation_preflight_uses_target_scope_not_previous_process(self):
        binary = self.root / 'server'
        binary.write_text('synthetic')
        keys = ('ALLOWED_ORIGIN SITE_BASE_URL DB_USER DB_PASSWORD DB_NAME KAKAO_CLIENT_ID '
                'KAKAO_CLIENT_SECRET KAKAO_REDIRECT_URI JWT_SECRET UPLOAD_LEGACY_PATH '
                'EASYPAY_IMMEDIATELY_MALL_ID EASYPAY_PROFILE_MALL_ID EASYPAY_GW_URL '
                'EASYPAY_BIN_BASE EASYPAY_RETURN_BASE_URL SMTP_HOST SMTP_USER SMTP_PASSWORD VISIT_IP_SALT ENV')
        env = {key: 'synthetic' for key in keys.split()}
        env['ACCOUNT_ERASURE_TEST_USER_SEQ'] = '42'
        for target in [0, 43]:
            with patch.object(remote, 'run') as command:
                remote.validate_activation(binary, env, target)
            checked = command.call_args.kwargs['env']
            self.assertEqual(checked['ACCOUNT_ERASURE_TEST_USER_SEQ'], str(target))
            self.assertEqual(checked['PRIVACY_RETENTION_ENABLED'], 'false')
        self.assertEqual(env['ACCOUNT_ERASURE_TEST_USER_SEQ'], '42')

    def test_erasure_enable_never_enables_retention(self):
        with patch.object(remote, 'run'):
            remote.set_rollout(True)
        values = dict(line.split('=', 1) for line in self.env.read_text().splitlines())
        self.assertEqual(values['ACCOUNT_ERASURE_WORKER_ENABLED'], 'true')
        self.assertEqual(values['PRIVACY_RETENTION_ENABLED'], 'false')

    def test_retention_can_run_without_erasure(self):
        with patch.object(remote, 'run'):
            remote.set_rollout(False, retention=True)
        values = dict(line.split('=', 1) for line in self.env.read_text().splitlines())
        self.assertEqual(values['ACCOUNT_ERASURE_WORKER_ENABLED'], 'false')
        self.assertEqual(values['PRIVACY_RETENTION_ENABLED'], 'true')

    def test_test_account_rejects_retention_before_mutation(self):
        previous = self.env.read_text()
        with patch.object(remote, 'run') as command:
            with self.assertRaises(ValueError):
                remote.change_rollout(self.root, True, 42, retention=True)
        command.assert_not_called()
        self.assertEqual(self.env.read_text(), previous)

    def test_lock_excludes_another_process_and_releases_after_failure(self):
        path = self.root / 'release.lock'
        code = "import sys; from pathlib import Path; import remote_release as r\nwith r.release_lock(Path(sys.argv[1])): pass"
        env = dict(os.environ, PYTHONPATH=str(Path(remote.__file__).parent))
        with self.assertRaisesRegex(RuntimeError, 'synthetic'):
            with remote.release_lock(path):
                result = subprocess.run([sys.executable, '-c', code, str(path)], env=env, capture_output=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(b'another release operation', result.stderr)
                raise RuntimeError('synthetic')
        result = subprocess.run([sys.executable, '-c', code, str(path)], env=env, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_hold_blocks_candidate_before_transfer(self):
        (self.root / 'REVIEW_STATUS.json').write_text(json.dumps({'status': 'HOLD_ADDITIONAL_REVIEW_FINDINGS'}))
        with self.assertRaisesRegex(ValueError, 'review hold'):
            bundle.verify(self.root)
