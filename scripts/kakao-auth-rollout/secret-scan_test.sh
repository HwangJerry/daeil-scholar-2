#!/usr/bin/env bash
# secret-scan_test.sh — Verify credential-source externalization and scanner behavior.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
UNIT="$ROOT/deploy/alumni-backend.service"
ENV_EXAMPLE="$ROOT/deploy/alumni-backend.env.example"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

if [[ ! -f "$UNIT" ]]; then
  fail "service unit is missing"
fi

if grep -Eq '^[[:space:]]*Environment[[:space:]]*=' "$UNIT"; then
  fail "service unit still contains inline Environment assignments"
fi

if ! grep -Fxq 'EnvironmentFile=/etc/sysconfig/alumni-backend' "$UNIT"; then
  fail "service unit does not require /etc/sysconfig/alumni-backend"
fi

if [[ ! -f "$ENV_EXAMPLE" ]]; then
  fail "deployment environment example is missing"
fi

for key in DB_PASSWORD JWT_SECRET KAKAO_CLIENT_SECRET SMTP_PASSWORD DEBUG_AGENT_SECRET VISIT_IP_SALT; do
  if ! grep -Eq "^${key}=$" "$ENV_EXAMPLE"; then
    fail "${key} must be present and blank in the environment example"
  fi
done

printf 'PASS: service environment externalization\n'

SCANNER="$ROOT/scripts/kakao-auth-rollout/secret-scan.sh"
if [[ ! -f "$SCANNER" ]]; then
  fail "secret scanner is missing"
fi

TMP_REPO=$(mktemp -d "${TMPDIR:-/tmp}/kakao-secret-scan.XXXXXX")
trap 'rm -rf "$TMP_REPO"' EXIT
git -C "$TMP_REPO" init -q
git -C "$TMP_REPO" config user.email scanner-test@example.invalid
git -C "$TMP_REPO" config user.name scanner-test

mkdir -p "$TMP_REPO/config" "$TMP_REPO/dist"
{
  printf '%s=\n' DB_PASSWORD KAKAO_CLIENT_SECRET SMTP_PASSWORD
  printf '%s=%s\n' JWT_SECRET change-me-in-production
  printf '%s=%s\n' DEBUG_AGENT_SECRET "\${DEBUG_AGENT_SECRET:-\$(openssl rand -hex 32)}"
  printf '  %s:%s|%s:%s)\n' DEBUG_AGENT_SECRET change-me DEBUG_AGENT_SECRET test-secret
  printf "  '%s' => '%s',\n" JWT_SECRET change-me-in-production
} > "$TMP_REPO/config/runtime.env.example"
git -C "$TMP_REPO" add config/runtime.env.example
git -C "$TMP_REPO" commit -qm 'safe baseline'

if ! bash "$SCANNER" --repo "$TMP_REPO" --current >/dev/null; then
  fail "blank and placeholder values must not be findings"
fi

CURRENT_SECRET='fixture-current-A7!q2Z9m'
printf '%s=%s\n' DB_PASSWORD "$CURRENT_SECRET" > "$TMP_REPO/config/runtime.env"
set +e
CURRENT_OUTPUT=$(bash "$SCANNER" --repo "$TMP_REPO" --current 2>&1)
CURRENT_STATUS=$?
set -e
[[ $CURRENT_STATUS -eq 1 ]] || fail "current literal credential must exit 1"
[[ "$CURRENT_OUTPUT" == *'scope=current'* ]] || fail "current finding must identify its scope"
[[ "$CURRENT_OUTPUT" == *'key=DB_PASSWORD'* ]] || fail "current finding must identify its key"
[[ "$CURRENT_OUTPUT" == *'path=config/runtime.env'* ]] || fail "current finding must identify its path"
[[ "$CURRENT_OUTPUT" != *"$CURRENT_SECRET"* ]] || fail "current finding leaked the secret value"
printf 'PASS: current-tree literal detection is non-leaking\n'

git -C "$TMP_REPO" add config/runtime.env
git -C "$TMP_REPO" commit -qm 'add historical fixture'
rm "$TMP_REPO/config/runtime.env"
git -C "$TMP_REPO" add -u
git -C "$TMP_REPO" commit -qm 'remove historical fixture'

set +e
HISTORY_OUTPUT=$(bash "$SCANNER" --repo "$TMP_REPO" --history 2>&1)
HISTORY_STATUS=$?
set -e
[[ $HISTORY_STATUS -eq 1 ]] || fail "history-only credential must exit 1"
[[ "$HISTORY_OUTPUT" == *'scope=history'* ]] || fail "history finding must identify its scope"
[[ "$HISTORY_OUTPUT" == *'key=DB_PASSWORD'* ]] || fail "history finding must identify its key"
[[ "$HISTORY_OUTPUT" != *"$CURRENT_SECRET"* ]] || fail "history finding leaked the secret value"
printf 'PASS: Git-history literal detection is non-leaking\n'

ARTIFACT_SECRET='fixture-artifact-R8@t4M6x'
printf '%s=%s\n' SMTP_PASSWORD "$ARTIFACT_SECRET" > "$TMP_REPO/dist/release.env"
set +e
ARTIFACT_OUTPUT=$(bash "$SCANNER" --repo "$TMP_REPO" --artifacts 2>&1)
ARTIFACT_STATUS=$?
set -e
[[ $ARTIFACT_STATUS -eq 1 ]] || fail "artifact credential must exit 1"
[[ "$ARTIFACT_OUTPUT" == *'scope=artifact'* ]] || fail "artifact finding must identify its scope"
[[ "$ARTIFACT_OUTPUT" == *'key=SMTP_PASSWORD'* ]] || fail "artifact finding must identify its key"
[[ "$ARTIFACT_OUTPUT" != *"$ARTIFACT_SECRET"* ]] || fail "artifact finding leaked the secret value"
printf 'PASS: artifact literal detection is non-leaking\n'
