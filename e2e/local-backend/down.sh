#!/usr/bin/env bash
set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/common.sh"
preflight "$@"
compose down --volumes --remove-orphans
echo 'Local E2E containers, network and volumes removed.'
