#!/bin/sh
set -eu
if env | grep -iq 'daeilfoundation\.or\.kr'; then
    echo 'Refusing production configuration.' >&2
    exit 1
fi
# Keep only the connected Docker subnet and loopback; external providers cannot
# be reached even if a future code path accidentally uses a hardcoded URL.
ip route del default
exec /app/server
