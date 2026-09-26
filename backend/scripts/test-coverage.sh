#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
profile=$(mktemp "${TMPDIR:-/tmp}/dflh-coverage.XXXXXX")
trap 'rm -f "$profile"' EXIT

# DFLH_DOCKER_TESTS=1 includes tests using the shared MariaDB harness.
# Unmigrated integrations still use their existing opt-in variables.
# Never inherit the legacy integration's external database DSN.
env -u SOCIAL_LINK_TEST_DSN go test ./... -coverprofile="$profile"

printf '\nPer-package statement coverage:\n'
awk 'NR > 1 {
  package = $1
  sub(/\/[^\/]+$/, "", package)
  statements[package] += $2
  if ($3 > 0) covered[package] += $2
}
END {
  for (package in statements) {
    if (statements[package] > 0)
      printf "%s\t%.1f%%\n", package, 100 * covered[package] / statements[package]
    else
      printf "%s\t[no statements]\n", package
  }
}' "$profile" | sort

go tool cover -func="$profile" | awk '/^total:/ { print }'
