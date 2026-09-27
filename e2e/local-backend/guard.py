#!/usr/bin/env python3
"""Reject production references without logging environment values."""

import json
import os
from pathlib import Path
import sys
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parent
FORBIDDEN = "daeilfoundation.or.kr"


def require(condition, message):
    if not condition:
        sys.exit("Refusing local E2E: " + message)


for key, value in os.environ.items():
    require(FORBIDDEN not in value.lower(), "production reference in environment variable " + key)
for name in ("local.env", "compose.yaml", "seed.sql"):
    require(not (ROOT / name).is_symlink(), "symlinked configuration: " + name)
    require(FORBIDDEN not in (ROOT / name).read_text().lower(), "production reference in " + name)

if "--compose" in sys.argv:
    config = json.load(sys.stdin)
    require(FORBIDDEN not in json.dumps(config).lower(), "production reference in resolved Compose configuration")
    services = config["services"]
    require(set(services) == {"db", "api"}, "unexpected Compose services")
    require(config["networks"]["default"].get("driver") == "bridge", "network must be a local bridge")
    for service in services.values():
        require(not service.get("network_mode"), "network_mode override")
        require(service.get("networks") == {"default": None}, "unexpected service network")
        for port in service.get("ports", []):
            require(port.get("host_ip") == "127.0.0.1", "ports must bind to IPv4 loopback")
    api_env = services["api"]["environment"]
    require(services["api"].get("command") == ["/bin/sh", "/local/start-api.sh"], "API must remove its default route")
    require(services["api"].get("dns") == ["127.0.0.1"], "external DNS must remain disabled")
    require(api_env["DB_HOST"] == "db" and api_env["DB_NAME"] == "alumni_e2e", "unexpected database target")
    require(api_env["PUSH_ENABLED"] == "false" and api_env["SMS_PROVIDER"] == "", "delivery must remain disabled")
    for key, value in api_env.items():
        if key.endswith(("URL", "URI", "URIS", "ORIGIN", "HOST")) and value:
            for target in value.split(","):
                host = urlsplit(target).hostname if "://" in target else target
                require(host in {"localhost", "127.0.0.1", "db"}, "non-local target in " + key)
    pinned = (ROOT / "../../backend/migrations/testdata/mariadb-10.1.38.image").read_text().strip()
    require(services["db"]["image"] == pinned, "MariaDB must use the harness image digest")
