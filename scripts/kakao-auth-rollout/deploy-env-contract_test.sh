#!/usr/bin/env bash
# deploy-env-contract_test.sh — Verify deploy preflight never returns runtime secrets locally.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
DEPLOY="$ROOT/deploy.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

ROLLBACK_PARSER=$(sed -n '/^parse_backend_rollback_result()/,/^}/p' "$DEPLOY")
[[ -n $ROLLBACK_PARSER ]] || fail "backend rollback response parser is missing"
eval "$ROLLBACK_PARSER"

parse_backend_rollback_result 'NONE NONE' || fail "canonical no-rollback response was rejected"
[[ $BACKEND_ROLLBACK_PATH == NONE && $BACKEND_ROLLBACK_SHA256 == NONE ]] ||
  fail "canonical no-rollback response parsed incorrectly"
VALID_ROLLBACK_PATH=/app/backend/.server.rollback.Abc123
VALID_ROLLBACK_SHA=$(printf 'a%.0s' {1..64})
parse_backend_rollback_result "$VALID_ROLLBACK_PATH $VALID_ROLLBACK_SHA" ||
  fail "canonical rollback response was rejected"
[[ $BACKEND_ROLLBACK_PATH == "$VALID_ROLLBACK_PATH" && $BACKEND_ROLLBACK_SHA256 == "$VALID_ROLLBACK_SHA" ]] ||
  fail "canonical rollback response parsed incorrectly"
for malformed in \
  ' NONE NONE' \
  'NONE NONE ' \
  'NONE  NONE' \
  $'NONE\tNONE' \
  $'NONE NONE\nNONE NONE' \
  'NONE NONE EXTRA' \
  "/app/backend/.server.rollback.Abc123  $VALID_ROLLBACK_SHA" \
  "/app/backend/.server.rollback.Abc123 $VALID_ROLLBACK_SHA "; do
  if parse_backend_rollback_result "$malformed"; then
    fail "noncanonical rollback response was accepted"
  fi
done

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
for deployment_binding in APPROVED_SOURCE_REVISION APPROVED_BACKEND_ARTIFACT_SHA256 APPROVED_MAINTENANCE_GENERATION APPROVED_PREDEPLOY_BINARY_SHA256 APPROVED_PREDEPLOY_BINARY_SIZE ACTIVE_UNIT_PROTOCOL BACKEND_ROLLBACK_PATH; do
  grep -Fq "$deployment_binding" "$DEPLOY" || fail "deploy.sh is missing ${deployment_binding} binding"
done
for external_binding in APPROVED_EXTERNAL_GO_MOD_SHA256 APPROVED_EXTERNAL_KAKAO_CLIENT_SHA256; do
  grep -Fq "$external_binding" "$DEPLOY" || fail "deploy.sh is missing ${external_binding} binding"
done
[[ $(grep -Fc 'verify_approved_external_inputs' "$DEPLOY") -ge 3 ]] ||
  fail "bundle-local external inputs are not checked before and after backend build"
# shellcheck disable=SC2016 # Match the literal command substitution in deploy.sh.
grep -Fq '[[ $(GOTOOLCHAIN=local go env GOVERSION) == go1.25.2 ]]' "$DEPLOY" ||
  fail "deploy does not pin the approved Go toolchain before building"
grep -Fq "GOWORK=off GOFLAGS='' GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -trimpath -buildvcs=false -o ../dist/server ./cmd/server" "$DEPLOY" ||
  fail "backend build is not path-independent or does not match the approved artifact environment"
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
grep -Fq 'BACKEND_ROLLBACK_SHA256' "$DEPLOY" ||
  fail "backend rollback artifact is not bound to its creation-time SHA-256"
if grep -Fq 'read -r BACKEND_ROLLBACK_PATH' "$DEPLOY"; then
  fail "backend rollback result uses permissive IFS parsing"
fi
grep -Fq 'rollback_checksum_mismatch' "$DEPLOY" ||
  fail "backend rollback recovery does not authenticate rollback bytes before restore"
if grep -Fq -- 'systemctl show --property MainPID --value' "$DEPLOY"; then
  fail "backend rollback recovery uses unsupported old-systemd --value filtering"
fi
# shellcheck disable=SC2016 # Match the literal remote-shell command substitution in deploy.sh.
grep -Fq 'pid_output=\$(sudo systemctl show --property MainPID alumni-backend)' "$DEPLOY" ||
  fail "backend rollback recovery does not read unfiltered MainPID metadata"
# shellcheck disable=SC2016 # Match the literal escaped remote-shell variables in deploy.sh.
grep -Fq '[[ \$pid_output == MainPID=* && \$pid_output != *\$' "$DEPLOY" ||
  fail "backend rollback recovery does not require exactly one MainPID property line"
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
grep -Fq 'backend_guard_failed_before_backend_replace' "$DEPLOY" ||
  fail "evidence deployment does not require a stopped backend immediately before replacement"
BACKEND_GUARD_LINE=$(grep -nF 'backend_guard_failed_before_backend_replace' "$DEPLOY" | head -n 1 | cut -d: -f1)
# shellcheck disable=SC2016 # Match the literal escaped remote-shell staging command in deploy.sh.
SERVER_STAGE_REMOTE_LINE=$(grep -nF 'staged=\$(mktemp /app/backend/.server.new.XXXXXX)' "$DEPLOY" | head -n 1 | cut -d: -f1)
[[ -n $BACKEND_GUARD_LINE && -n $SERVER_STAGE_REMOTE_LINE && $BACKEND_GUARD_LINE -lt $SERVER_STAGE_REMOTE_LINE ]] ||
  fail "backend stopped-state guard does not precede server upload staging"
grep -Fq 'backend_guard_failed_immediately_before_backend_replace' "$DEPLOY" ||
  fail "backend stopped-state is not rechecked immediately before replacement"
SECOND_BACKEND_GUARD_LINE=$(grep -nF 'backend_guard_failed_immediately_before_backend_replace' "$DEPLOY" | head -n 1 | cut -d: -f1)
ROLLBACK_COPY_LINE=$(grep -nF 'cp -p -- /app/backend/server' "$DEPLOY" | head -n 1 | cut -d: -f1)
PREDEPLOY_IDENTITY_LINE=$(grep -nF 'predeploy_binary_identity_changed_before_backend_replace' "$DEPLOY" | head -n 1 | cut -d: -f1)
# shellcheck disable=SC2016 # Match the literal remote $staged variable.
STAGED_CHMOD_LINE=$(grep -nF 'chmod 0755 \"\$staged\"' "$DEPLOY" | tail -n 1 | cut -d: -f1)
# shellcheck disable=SC2016 # Contract intentionally matches the literal remote $staged variable.
SERVER_REPLACE_LINE=$(grep -nF 'mv -fT \"\$staged\" /app/backend/server' "$DEPLOY" | head -n 1 | cut -d: -f1)
[[ -n $SECOND_BACKEND_GUARD_LINE && -n $ROLLBACK_COPY_LINE && -n $PREDEPLOY_IDENTITY_LINE && -n $STAGED_CHMOD_LINE && -n $SERVER_REPLACE_LINE &&
   $ROLLBACK_COPY_LINE -lt $PREDEPLOY_IDENTITY_LINE && $PREDEPLOY_IDENTITY_LINE -lt $STAGED_CHMOD_LINE && $STAGED_CHMOD_LINE -lt $SECOND_BACKEND_GUARD_LINE &&
   $SECOND_BACKEND_GUARD_LINE -lt $SERVER_REPLACE_LINE ]] ||
  fail "approved predeploy identity and second backend guard are not enforced before replacement"
[[ $(grep -Fc 'maintenance_generation_changed_before_backend_replace' "$DEPLOY") -ge 2 ]] ||
  fail "approved maintenance generation is not checked initially and immediately before replacement"
for quoted_binding in APPROVED_MAINTENANCE_GENERATION_Q APPROVED_PREDEPLOY_BINARY_SHA256_Q APPROVED_PREDEPLOY_BINARY_SIZE_Q; do
  grep -Fq "printf -v $quoted_binding '%q'" "$DEPLOY" ||
    fail "remote approval binding ${quoted_binding} is not shell-quoted"
done
grep -Fq 'current_binary_missing_before_backend_replace' "$DEPLOY" ||
  fail "evidence deployment does not reject an absent or dangling preflight binary"
grep -Fq 'current_binary_identity_changed_immediately_before_backend_replace' "$DEPLOY" ||
  fail "active production binary is not rehashed at the final mutation boundary"
grep -Fq 'deployment_evidence_appeared_before_backend_replace' "$DEPLOY" ||
  fail "racing deployment evidence is not rejected at the final mutation boundary"
grep -Fq 'verify_remote_deployment_evidence' "$DEPLOY" ||
  fail "deployment does not resolve recorder acknowledgment ambiguity by authenticated readback"
grep -Fq 'cleanup_deployment_evidence_for_rollback' "$DEPLOY" ||
  fail "rollback does not invalidate exact deployment PASS evidence"
grep -Fq 'if [[ ${RECORD_MAINTENANCE_DEPLOY_EVIDENCE} == 1 ]]; then' "$DEPLOY" ||
  fail "predeploy identity check is not scoped to evidence deployment"
grep -Fq 'if [[ -n \"\${staged:-}\" ]]; then rm -f \"\$staged\"; fi; if [[ \"\${rollback:-NONE}\" != NONE ]]; then rm -f \"\$rollback\"; fi' "$DEPLOY" ||
  fail "remote transaction does not clean staged and rollback temporary artifacts on failure"
[[ $(grep -Fc 'sudo systemctl daemon-reload && sudo systemctl restart alumni-backend' "$DEPLOY") == 1 ]] ||
  fail "backend restart failure can restart a sentinel-unaware rollback binary"
grep -Fq 'Backend evidence failed; previous binary was restored and left stopped' "$DEPLOY" ||
  fail "deployment evidence failure does not restore the previous binary safely"
if grep -Fq 'ssh "${SSH_OPTS[@]}"' "$DEPLOY"; then
  fail "raw SSH invocation still expands an empty option array under Bash 3.2 nounset"
fi
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
EVIDENCE_LINE=$(grep -nF 'record_and_verify_deployment_evidence' "$DEPLOY" | tail -n 1 | cut -d: -f1)
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
if [[ -n ${FAKE_CAPTURE_ARGS_FILE:-} ]]; then
  printf '%s\n' "$@" > "$FAKE_CAPTURE_ARGS_FILE"
  exit 0
fi
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

SSH_REMOTE_FUNCTION=$(sed -n '/^ssh_remote()/,/^}/p' "$DEPLOY")
[[ -n $SSH_REMOTE_FUNCTION ]] || fail "ssh_remote helper is missing"
eval "$SSH_REMOTE_FUNCTION"
SSH_ARGS="$TMP/ssh-args.txt"
SSH_PORT=
PATH="$TMP/bin:$PATH" FAKE_CAPTURE_ARGS_FILE="$SSH_ARGS" ssh_remote fake-target 'printf no-port'
[[ $(sed -n '1p' "$SSH_ARGS") == fake-target && $(sed -n '2p' "$SSH_ARGS") == 'printf no-port' &&
   $(wc -l < "$SSH_ARGS" | tr -d ' ') == 2 ]] ||
  fail "Bash 3.2 no-port ssh_remote argv is invalid"
SSH_PORT=2222
PATH="$TMP/bin:$PATH" FAKE_CAPTURE_ARGS_FILE="$SSH_ARGS" ssh_remote fake-target 'printf explicit-port'
[[ $(sed -n '1p' "$SSH_ARGS") == -p && $(sed -n '2p' "$SSH_ARGS") == 2222 &&
   $(sed -n '3p' "$SSH_ARGS") == fake-target && $(sed -n '4p' "$SSH_ARGS") == 'printf explicit-port' &&
   $(wc -l < "$SSH_ARGS" | tr -d ' ') == 4 ]] ||
  fail "Bash 3.2 explicit-port ssh_remote argv is invalid"
grep -Fq 'RSYNC_SSH_COMMAND=ssh' "$DEPLOY" || fail "frontend rsync no-port transport is not explicit"
if grep -Fq 'SSH_OPTS' "$DEPLOY"; then
  fail "deploy still depends on Bash empty-array SSH options"
fi

EVIDENCE_ORCHESTRATOR=$(sed -n '/^record_and_verify_deployment_evidence()/,/^}/p' "$DEPLOY")
[[ -n $EVIDENCE_ORCHESTRATOR ]] || fail "deployment evidence orchestrator is missing"
(
  eval "$EVIDENCE_ORCHESTRATOR"
  TARGET=fake-target
  BACKEND_ARTIFACT_SHA256=$(printf 'a%.0s' {1..64})
  BACKEND_ROLLBACK_PATH=/app/backend/.server.rollback.fixture
  BACKEND_ROLLBACK_SHA256=$(printf 'b%.0s' {1..64})
  EVENTS="$TMP/ack-lost-events"
  ssh_remote() { printf 'recorder_failed\n' >> "$EVENTS"; return 1; }
  verify_remote_deployment_evidence() { printf 'verified\n' >> "$EVENTS"; return 0; }
  restore_backend_without_restart() { printf 'restored\n' >> "$EVENTS"; return 0; }
  record_and_verify_deployment_evidence >/dev/null
  [[ $(grep -Fxc recorder_failed "$EVENTS") == 1 && $(grep -Fxc verified "$EVENTS") == 2 &&
     $(grep -Fxc restored "$EVENTS" || true) == 0 ]] ||
    fail "lost recorder acknowledgment rolls back authenticated PASS evidence"
)
(
  eval "$EVIDENCE_ORCHESTRATOR"
  TARGET=fake-target
  BACKEND_ARTIFACT_SHA256=$(printf 'a%.0s' {1..64})
  BACKEND_ROLLBACK_PATH=/app/backend/.server.rollback.fixture
  BACKEND_ROLLBACK_SHA256=$(printf 'b%.0s' {1..64})
  EVENTS="$TMP/no-evidence-events"
  ssh_remote() { printf 'recorder_failed\n' >> "$EVENTS"; return 1; }
  verify_remote_deployment_evidence() { printf 'verify_failed\n' >> "$EVENTS"; return 1; }
  restore_backend_without_restart() { printf 'restored\n' >> "$EVENTS"; return 0; }
  set +e
  record_and_verify_deployment_evidence >/dev/null 2>&1
  STATUS=$?
  set -e
  [[ $STATUS -ne 0 && $(grep -Fxc verify_failed "$EVENTS") == 1 &&
     $(grep -Fxc restored "$EVENTS") == 1 ]] ||
    fail "recorder failure without valid evidence does not restore the previous binary"
)
(
  eval "$EVIDENCE_ORCHESTRATOR"
  TARGET=fake-target
  BACKEND_ARTIFACT_SHA256=$(printf 'a%.0s' {1..64})
  BACKEND_ROLLBACK_PATH=/app/backend/.server.rollback.fixture
  BACKEND_ROLLBACK_SHA256=$(printf 'b%.0s' {1..64})
  EVENTS="$TMP/readback-failed-events"
  ssh_remote() { printf 'recorder_ok\n' >> "$EVENTS"; return 0; }
  verify_remote_deployment_evidence() { printf 'verify_failed\n' >> "$EVENTS"; return 1; }
  restore_backend_without_restart() { printf 'restored\n' >> "$EVENTS"; return 0; }
  set +e
  record_and_verify_deployment_evidence >/dev/null 2>&1
  STATUS=$?
  set -e
  [[ $STATUS -ne 0 && $(grep -Fxc verify_failed "$EVENTS") == 1 &&
     $(grep -Fxc restored "$EVENTS") == 1 ]] ||
    fail "acknowledged recorder output bypasses authenticated evidence readback"
)

MIGRATION_LIST="$TMP/migrations.txt"
(
  cd "$ROOT/backend/migrations"
  printf '%s\n' [0-9][0-9][0-9]_*.sql | sort
) > "$MIGRATION_LIST"

set +e
EVIDENCE_PREFLIGHT_OUTPUT=$(
  PATH="$TMP/bin:$PATH" \
  FAKE_MIGRATION_LIST_FILE="$MIGRATION_LIST" \
  RECORD_MAINTENANCE_DEPLOY_EVIDENCE=1 \
  APPROVED_SOURCE_REVISION=$(git rev-parse HEAD) \
  APPROVED_BACKEND_ARTIFACT_SHA256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  bash "$DEPLOY" fake-target --backend=true --frontend=false --preflight-only 2>&1
)
EVIDENCE_PREFLIGHT_STATUS=$?
set -e
[[ $EVIDENCE_PREFLIGHT_STATUS -eq 0 &&
   $EVIDENCE_PREFLIGHT_OUTPUT == *'Preflight complete; no build, upload, restart, or reload performed'* ]] ||
  fail "read-only evidence preflight is blocked by operational-root dirtiness"

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
