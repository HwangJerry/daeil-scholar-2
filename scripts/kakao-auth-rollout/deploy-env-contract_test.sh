#!/usr/bin/env bash
# deploy-env-contract_test.sh — Verify deploy preflight never returns runtime secrets locally.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
DEPLOY="$ROOT/deploy.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

if grep -Eq 'UNIT_CONTENT|DB_PASS_VAL|Environment=\\?"?DB_PASSWORD' "$DEPLOY"; then
  fail "deploy.sh still parses inline unit credentials or stores DB password locally"
fi
if grep -Eq "MYSQL_PWD='\\?\$\{" "$DEPLOY"; then
  fail "deploy.sh still interpolates DB password into a remote command"
fi
if ! grep -Fq 'ENV_FILE_PATH="/etc/sysconfig/alumni-backend"' "$DEPLOY"; then
  fail "deploy.sh does not declare the external EnvironmentFile contract"
fi
if grep -Fq 'systemctl cat alumni-backend' "$DEPLOY"; then
  fail "deploy preflight trusts unit text instead of the effective EnvironmentFile property"
fi
# shellcheck disable=SC2016 # Match the literal PHP variable in deploy.sh.
ACTIVE_UNIT_BLOCK=$(sed -n '/ACTIVE_UNIT_PROTOCOL=1/,/^\$failed = false;/p' "$DEPLOY")
[[ -n $ACTIVE_UNIT_BLOCK ]] || fail "deploy preflight active-unit protocol block is missing"
if grep -Fq -- '--value' <<< "$ACTIVE_UNIT_BLOCK" ||
   grep -Fq -- '--property' <<< "$ACTIVE_UNIT_BLOCK"; then
  fail "deploy preflight requires unsupported filtered systemctl show metadata"
fi
grep -Fq '/bin/systemctl show alumni-backend' <<< "$ACTIVE_UNIT_BLOCK" ||
  fail "deploy preflight does not inspect active unit metadata"
grep -Fq "/bin/grep -E '^EnvironmentFiles?='" <<< "$ACTIVE_UNIT_BLOCK" ||
  fail "deploy preflight does not reduce active unit metadata to the EnvironmentFile allowlist"
if ! grep -Fq '0640' "$DEPLOY" || ! grep -Fq 'alumni-backend' "$DEPLOY"; then
  fail "deploy preflight does not preserve the approved 0640 root:alumni-backend contract"
fi
for maintenance_key in MAINTENANCE_SENTINEL_PATH MAINTENANCE_SMOKE_PROOF_SHA256 MAINTENANCE_SMOKE_ALLOWED_PATHS; do
  grep -Fq "$maintenance_key" "$DEPLOY" || fail "deploy preflight does not validate $maintenance_key"
done
if ! grep -Fq -- '--preflight-only' "$DEPLOY"; then
  fail "deploy.sh does not expose a non-mutating preflight mode"
fi
if grep -Eq 'APPLY_MIGRATIONS|run_remote_apply_migration|Apply these .*migration' "$DEPLOY"; then
  fail "deploy.sh must never apply migrations; use the separately approved dedicated runner"
fi
grep -Fq 'RECORD_MAINTENANCE_DEPLOY_EVIDENCE' "$DEPLOY" ||
  fail "deploy.sh does not expose generation-bound deployment evidence recording"
grep -Fq 'maintenance-record-deployment.sh' "$DEPLOY" ||
  fail "deploy.sh does not invoke the deployment evidence producer"
for deployment_binding in APPROVED_SOURCE_REVISION APPROVED_BACKEND_ARTIFACT_SHA256 ACTIVE_UNIT_PROTOCOL BACKEND_ROLLBACK_PATH; do
  grep -Fq "$deployment_binding" "$DEPLOY" || fail "deploy.sh is missing ${deployment_binding} binding"
done
for required_example_key in EASYPAY_IMMEDIATELY_MALL_ID EASYPAY_PROFILE_MALL_ID EASYPAY_BIN_BASE ENV; do
  grep -Eq "^${required_example_key}=" "$ROOT/deploy/alumni-backend.env.example" ||
    fail "EnvironmentFile example is missing ${required_example_key}"
done
grep -Fq 'mktemp /app/backend/.server.new.XXXXXX' "$DEPLOY" ||
  fail "backend upload does not use a server-generated random staging path"
BACKFILL_STAGE_LINE=$(grep -nF 'mktemp /app/backend/.backfill.new.XXXXXX' "$DEPLOY" | head -n 1 | cut -d: -f1)
SERVER_STAGE_LINE=$(grep -nF 'mktemp /app/backend/.server.new.XXXXXX' "$DEPLOY" | head -n 1 | cut -d: -f1)
[[ -n $BACKFILL_STAGE_LINE && -n $SERVER_STAGE_LINE && $BACKFILL_STAGE_LINE -lt $SERVER_STAGE_LINE ]] ||
  fail "active backend binary is replaced before dormant backfill upload completes"
grep -Fq 'BUILD_BACKFILL_ARTIFACT=0' "$DEPLOY" ||
  fail "evidence-enabled deployment does not exclude the unapproved backfill artifact"
# The deploy source pattern must contain the literal variable reference.
# shellcheck disable=SC2016
[[ $(grep -Fc 'if [[ $BUILD_BACKFILL_ARTIFACT == 1 ]]; then' "$DEPLOY") -ge 2 ]] ||
  fail "backfill build and upload are not both scoped out of evidence deployment"
if ! grep -Fq 'mv -fT' "$DEPLOY" || ! grep -Fq '/app/backend/server' "$DEPLOY"; then
  fail "backend destination does not reject directory replacement races"
fi
grep -Fq 'restore_backend_without_restart' "$DEPLOY" ||
  fail "backend failure path does not use the stopped rollback helper"
grep -Fq 'restore_backend_for_failed_restart' "$DEPLOY" ||
  fail "ordinary deployment restart failure no longer restores service availability"
grep -Fq "if [[ \$RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 ]]" "$DEPLOY" ||
  fail "stopped rollback is not scoped to evidence-enabled maintenance deployment"

grep -Fq 'verify_remote_maintenance_active' "$DEPLOY" ||
  fail "evidence deploy does not verify active maintenance before mutation"
if grep -Fq 'curl --noproxy' "$DEPLOY"; then
  fail "deploy control-plane curl does not disable user curl configuration first"
fi
MAINTENANCE_GUARD_CALL_LINE=$(grep -n "verify_remote_maintenance_active \"\$TARGET\"" "$DEPLOY" | tail -1 | cut -d: -f1)
FIRST_UPLOAD_LINE=$(grep -n '=== Uploading' "$DEPLOY" | head -1 | cut -d: -f1)
[[ -n $MAINTENANCE_GUARD_CALL_LINE && -n $FIRST_UPLOAD_LINE && $MAINTENANCE_GUARD_CALL_LINE -lt $FIRST_UPLOAD_LINE ]] ||
  fail "evidence deploy does not verify maintenance before its first upload"
grep -Fq 'maintenance_guard_failed_before_backend_replace' "$DEPLOY" ||
  fail "backend replacement is not coupled to a current maintenance guard"
RESTART_BLOCK=$(sed -n '/if ! ssh .*systemctl restart alumni-backend/,/fi/p' "$DEPLOY")
[[ $(grep -Fc 'systemctl restart alumni-backend' <<< "$RESTART_BLOCK") == 1 ]] ||
  fail "backend restart failure can restart a sentinel-unaware rollback binary"
grep -Fq 'Backend evidence failed; previous binary was restored and left stopped' "$DEPLOY" ||
  fail "deployment evidence failure does not restore the previous binary safely"
if grep -Fq 'restore_backend_without_restart || true' "$DEPLOY"; then
  fail "deployment evidence failure ignores rollback recovery failure"
fi
grep -Fq 'Backend evidence and rollback recovery both failed' "$DEPLOY" ||
  fail "deployment does not report uncertain state when rollback recovery fails"
if grep -Fq '/tmp/alumni.conf.new' "$DEPLOY" || grep -Fq "/tmp/\${shim}.new" "$DEPLOY"; then
  fail "deploy.sh still uses predictable privileged staging paths"
fi
grep -Fq 'BEGIN_FRONTEND_DEPLOY_SIDE_EFFECTS' "$DEPLOY" ||
  fail "frontend/Apache side effects are not isolated from backend-only deploy"
grep -Fq 'Evidence-enabled deployment requires --frontend=false' "$DEPLOY" ||
  fail "canonical evidence-enabled deploy does not enforce backend-only scope"
set +e
SKIP_OUTPUT=$(
  RECORD_MAINTENANCE_DEPLOY_EVIDENCE=1 \
  APPROVED_SOURCE_REVISION=$(git rev-parse HEAD) \
  APPROVED_BACKEND_ARTIFACT_SHA256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  SKIP_ENV_CHECK=1 SKIP_DEBUG_AGENT_CHECK=1 SKIP_MIGRATION_CHECK=1 \
  bash "$DEPLOY" fake-target --backend=true --frontend=false --preflight-only 2>&1
)
SKIP_STATUS=$?
set -e
[[ $SKIP_STATUS -ne 0 && $SKIP_OUTPUT == *'does not allow SKIP_* bypasses'* ]] ||
  fail "evidence-enabled deployment accepted or misclassified preflight bypasses"
RESTART_LINE=$(grep -nF 'sudo systemctl daemon-reload && sudo systemctl restart alumni-backend' "$DEPLOY" | head -n 1 | cut -d: -f1)
EVIDENCE_LINE=$(grep -nF 'maintenance-record-deployment.sh' "$DEPLOY" | tail -n 1 | cut -d: -f1)
[[ -n $RESTART_LINE && -n $EVIDENCE_LINE && $EVIDENCE_LINE -gt $RESTART_LINE ]] ||
  fail "deployment evidence is not recorded after backend restart"
HTTPD_RELOAD_LINE=$(grep -nF 'sudo systemctl reload httpd' "$DEPLOY" | tail -n 1 | cut -d: -f1)
[[ -n $HTTPD_RELOAD_LINE && $EVIDENCE_LINE -gt $HTTPD_RELOAD_LINE ]] ||
  fail "deployment evidence is recorded before all selected deployment side effects"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/deploy-env-contract.XXXXXX")
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/bin"

cat > "$TMP/bin/ssh" <<'FAKE_SSH'
#!/usr/bin/env bash
set -euo pipefail
payload=$(cat)
case "$payload" in
  *ENV_VALIDATOR_PROTOCOL=1*)
    if [[ "${FAKE_ENV_VALIDATION:-ok}" == "ok" ]]; then
      printf 'STATUS OK\n'
      exit 0
    fi
    printf 'MISSING DB_PASSWORD\n'
    exit 1
    ;;
  *MIGRATION_HISTORY_PROTOCOL=1*)
    cat "${FAKE_MIGRATION_LIST_FILE}"
    exit 0
    ;;
  *)
    printf 'unexpected fake ssh protocol\n' >&2
    exit 2
    ;;
esac
FAKE_SSH
chmod +x "$TMP/bin/ssh"

MIGRATION_LIST="$TMP/migrations.txt"
(
  cd "$ROOT/backend/migrations"
  printf '%s\n' [0-9][0-9][0-9]_*.sql | sort
) > "$MIGRATION_LIST"

set +e
INVALID_TARGET_OUTPUT=$(
  PATH="$TMP/bin:$PATH" SKIP_MIGRATION_CHECK=1 \
  bash "$DEPLOY" '-oProxyCommand=fixture' --backend=true --frontend=false --preflight-only 2>&1
)
INVALID_TARGET_STATUS=$?
set -e
[[ $INVALID_TARGET_STATUS -eq 1 && "$INVALID_TARGET_OUTPUT" == *'Invalid SSH target'* ]] ||
  fail "option-shaped SSH target was not rejected before execution"

set +e
INVALID_PORT_OUTPUT=$(
  PATH="$TMP/bin:$PATH" SKIP_MIGRATION_CHECK=1 \
  bash "$DEPLOY" fake-target '-F' --backend=true --frontend=false --preflight-only 2>&1
)
INVALID_PORT_STATUS=$?
set -e
[[ $INVALID_PORT_STATUS -eq 1 && "$INVALID_PORT_OUTPUT" == *'Invalid SSH port'* ]] ||
  fail "option-shaped SSH port was not rejected before execution"

SUCCESS_OUTPUT=$(
  PATH="$TMP/bin:$PATH" \
  FAKE_ENV_VALIDATION=ok \
  SKIP_MIGRATION_CHECK=1 \
  bash "$DEPLOY" fake-target --backend=true --frontend=false --preflight-only
)
[[ "$SUCCESS_OUTPUT" == *'production EnvironmentFile validation passed'* ]] ||
  fail "successful remote EnvironmentFile validation was not reported"
[[ "$SUCCESS_OUTPUT" == *'Preflight complete; no build, upload, restart, or reload performed'* ]] ||
  fail "preflight-only mode did not stop before side effects"
printf 'PASS: remote EnvironmentFile validation preflight\n'

set +e
FAILURE_OUTPUT=$(
  PATH="$TMP/bin:$PATH" \
  FAKE_ENV_VALIDATION=fail \
  SKIP_MIGRATION_CHECK=1 \
  bash "$DEPLOY" fake-target --backend=true --frontend=false --preflight-only 2>&1
)
FAILURE_STATUS=$?
set -e
[[ $FAILURE_STATUS -eq 1 ]] || fail "missing required key must fail preflight"
[[ "$FAILURE_OUTPUT" == *'MISSING DB_PASSWORD'* ]] ||
  fail "missing key metadata was not surfaced"
printf 'PASS: remote validator failure is fail-closed\n'

MIGRATION_OUTPUT=$(
  PATH="$TMP/bin:$PATH" \
  FAKE_MIGRATION_LIST_FILE="$MIGRATION_LIST" \
  SKIP_ENV_CHECK=1 \
  SKIP_DEBUG_AGENT_CHECK=1 \
  bash "$DEPLOY" fake-target --backend=true --frontend=false --preflight-only
)
[[ "$MIGRATION_OUTPUT" == *'migrations applied on remote'* ]] ||
  fail "remote-only migration history query did not complete"
printf 'PASS: remote-only migration history preflight\n'

PENDING_LIST="$TMP/pending-migrations.txt"
sed '$d' "$MIGRATION_LIST" > "$PENDING_LIST"
set +e
PENDING_OUTPUT=$(
  PATH="$TMP/bin:$PATH" \
  FAKE_MIGRATION_LIST_FILE="$PENDING_LIST" \
  SKIP_ENV_CHECK=1 \
  SKIP_DEBUG_AGENT_CHECK=1 \
  bash "$DEPLOY" fake-target --backend=true --frontend=false --preflight-only 2>&1
)
PENDING_STATUS=$?
set -e
[[ $PENDING_STATUS -eq 1 ]] || fail "pending migration must fail preflight"
[[ "$PENDING_OUTPUT" == *'Migration drift detected'* ]] ||
  fail "pending migration drift was not reported"
[[ "$PENDING_OUTPUT" == *'deploy.sh never applies migrations'* ]] ||
  fail "pending preflight did not stop before the dedicated migration gate"
printf 'PASS: pending migration preflight is non-mutating\n'
