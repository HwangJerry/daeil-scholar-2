"""Offline regression checks for the local-stack safety boundary."""

import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parent
SCRIPTS = ("up.sh", "down.sh", "reset.sh", "smoke.sh")
PRODUCTION_URL = "https://ADMS.DAEILFOUNDATION.OR.KR"


class LocalGuardTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.stack = self.root / "e2e/local-backend"
        self.stack.mkdir(parents=True)
        for name in (*SCRIPTS, "common.sh", "guard.py", "local.env", "compose.yaml", "seed.sql"):
            shutil.copyfile(SOURCE / name, self.stack / name)
        pinned = self.root / "backend/migrations/testdata/mariadb-10.1.38.image"
        pinned.parent.mkdir(parents=True)
        shutil.copyfile(SOURCE / "../../backend/migrations/testdata/mariadb-10.1.38.image", pinned)
        self.env = {"PATH": os.environ["PATH"], "HOME": str(self.root)}
        self.config = {
            "networks": {"default": {"driver": "bridge"}},
            "services": {
                "db": {"image": pinned.read_text().strip()},
                "api": {
                    "command": ["/bin/sh", "/local/start-api.sh"],
                    "dns": ["127.0.0.1"],
                    "environment": {"DB_HOST": "db", "DB_NAME": "alumni_e2e",
                                    "PUSH_ENABLED": "false", "SMS_PROVIDER": "",
                                    "SITE_BASE_URL": "https://localhost:18080"},
                },
            },
        }
        for service in self.config["services"].values():
            service.update(networks={"default": None}, ports=[{"host_ip": "127.0.0.1"}])

    def guard(self, config=None):
        args = [sys.executable, str(self.stack / "guard.py")]
        if config is not None:
            args.append("--compose")
        return subprocess.run(args, env=self.env, input=json.dumps(config), text=True, capture_output=True)

    def test_local_configuration_passes(self):
        result = self.guard(self.config)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_production_environment_rejected_without_value(self):
        self.env["SITE_BASE_URL"] = PRODUCTION_URL + "/private-marker"
        result = self.guard()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SITE_BASE_URL", result.stderr)
        self.assertNotIn("private-marker", result.stderr)

    def test_production_reference_in_each_input_rejected(self):
        for name in ("local.env", "compose.yaml", "seed.sql"):
            with self.subTest(name=name):
                target = self.stack / name
                original = target.read_text()
                target.write_text(original + "\n# " + PRODUCTION_URL)
                self.assertNotEqual(self.guard().returncode, 0)
                target.write_text(original)

    def test_non_local_resolved_target_rejected(self):
        for target in (PRODUCTION_URL, "https://external.example.test"):
            with self.subTest(target=target):
                config = copy.deepcopy(self.config)
                config["services"]["api"]["environment"]["SITE_BASE_URL"] = target
                self.assertNotEqual(self.guard(config).returncode, 0)

    def test_public_port_and_host_network_rejected(self):
        for override in ({"ports": [{"host_ip": "0.0.0.0"}]}, {"network_mode": "host"}):
            with self.subTest(override=override):
                config = copy.deepcopy(self.config)
                config["services"]["api"].update(override)
                self.assertNotEqual(self.guard(config).returncode, 0)

    def test_unpinned_image_rejected(self):
        self.config["services"]["db"]["image"] = "mariadb:latest"
        self.assertNotEqual(self.guard(self.config).returncode, 0)

    def test_external_delivery_rejected(self):
        for key, value in (("PUSH_ENABLED", "true"), ("SMS_PROVIDER", "ncp")):
            with self.subTest(key=key):
                config = copy.deepcopy(self.config)
                config["services"]["api"]["environment"][key] = value
                self.assertNotEqual(self.guard(config).returncode, 0)

    def test_symlink_rejected_before_reading(self):
        target = self.stack / "local.env"
        target.unlink()
        target.symlink_to(self.root / "nonexistent-env")
        result = self.guard()
        self.assertIn("symlinked configuration", result.stderr)

    def test_every_entrypoint_rejects_production_before_docker(self):
        self.env["UNRELATED_URL"] = PRODUCTION_URL
        for script in SCRIPTS:
            with self.subTest(script=script):
                result = subprocess.run(["bash", str(self.stack / script)], env=self.env, text=True, capture_output=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("production reference", result.stderr)

    def test_every_entrypoint_skips_unavailable_docker(self):
        binary = self.root / "bin/docker"
        binary.parent.mkdir()
        binary.write_text('#!/bin/sh\nif [ "$1" = context ]; then echo unix:///synthetic.sock; exit 0; fi\nexit 1\n')
        binary.chmod(0o755)
        self.env["PATH"] = str(binary.parent) + os.pathsep + self.env["PATH"]
        for script in SCRIPTS:
            with self.subTest(script=script):
                result = subprocess.run(["bash", str(self.stack / script)], env=self.env, text=True, capture_output=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn("SKIP: Docker", result.stdout)


if __name__ == "__main__":
    unittest.main()
