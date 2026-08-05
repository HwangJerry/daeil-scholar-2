#!/bin/bash
set -euo pipefail

# Deploy script — cross-compile, upload, and restart services
# Usage: ./deploy.sh [user@host] [ssh_port] --patch-mode=true|false
#       --backend=true|false --frontend=true|false [--preflight-only]
#   ssh_port is optional; omit to use ~/.ssh/config or the default 22.
#   --patch-mode is required (no default). Omit to be prompted interactively.
#     true  → user SPA bundles VITE_WIP_ADMIN_CODE from frontend/.env (cover page ON)
#     false → user SPA built with VITE_WIP_ADMIN_CODE="" forced (cover page OFF)
#   --backend and --frontend control build targets.
#     default: both true
#     backend=false only builds/uploads frontend (user+admin SPA)
#     frontend=false only builds/uploads backend binaries + restart backend

POSITIONAL=()
PATCH_MODE=""
BUILD_BACKEND=true
BUILD_FRONTEND=true
PREFLIGHT_ONLY=false
RECORD_MAINTENANCE_DEPLOY_EVIDENCE=${RECORD_MAINTENANCE_DEPLOY_EVIDENCE:-0}
APPROVED_SOURCE_REVISION=${APPROVED_SOURCE_REVISION:-}
APPROVED_BACKEND_ARTIFACT_SHA256=${APPROVED_BACKEND_ARTIFACT_SHA256:-}
BUILD_BACKFILL_ARTIFACT=1
if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 ]]; then
  BUILD_BACKFILL_ARTIFACT=0
fi
for arg in "$@"; do
  case "$arg" in
    --patch-mode=true)  PATCH_MODE="true" ;;
    --patch-mode=false) PATCH_MODE="false" ;;
    --backend=true)   BUILD_BACKEND=true ;;
    --backend=false)  BUILD_BACKEND=false ;;
    --frontend=true)  BUILD_FRONTEND=true ;;
    --frontend=false) BUILD_FRONTEND=false ;;
    --preflight-only) PREFLIGHT_ONLY=true ;;
    --patch-mode=*)
      echo "✗ Invalid --patch-mode value: '${arg#--patch-mode=}' (must be true or false)" >&2
      exit 1
      ;;
    --backend=*)
      echo "✗ Invalid --backend value: '${arg#--backend=}' (must be true or false)" >&2
      exit 1
      ;;
    --frontend=*)
      echo "✗ Invalid --frontend value: '${arg#--frontend=}' (must be true or false)" >&2
      exit 1
      ;;
    --backend-only)
      BUILD_BACKEND=true
      BUILD_FRONTEND=false
      ;;
    --frontend-only)
      BUILD_BACKEND=false
      BUILD_FRONTEND=true
      ;;
    --patch-mode)
      echo "✗ --patch-mode requires =true or =false (e.g. --patch-mode=true)" >&2
      exit 1
      ;;
    *) POSITIONAL+=("$arg") ;;
  esac
done

TARGET=${POSITIONAL[0]:-"daeil-prod"}
SSH_PORT=${POSITIONAL[1]:-}

if [[ ${#POSITIONAL[@]} -gt 2 ]]; then
  echo "✗ Too many positional arguments." >&2
  exit 1
fi
if [[ ! $TARGET =~ ^([A-Za-z0-9][A-Za-z0-9._-]*@)?([A-Za-z0-9][A-Za-z0-9._-]*|\[[A-Fa-f0-9:]+\])$ ]]; then
  echo "✗ Invalid SSH target." >&2
  exit 1
fi
if [[ -n $SSH_PORT ]]; then
  if [[ ! $SSH_PORT =~ ^[0-9]{1,5}$ ]] ||
     ((10#$SSH_PORT < 1 || 10#$SSH_PORT > 65535)); then
    echo "✗ Invalid SSH port." >&2
    exit 1
  fi
fi

if [[ -n "${SSH_PORT}" ]]; then
  SSH_OPTS=(-p "${SSH_PORT}")
else
  SSH_OPTS=()
fi

ssh_remote() {
  if [[ -n "${SSH_PORT}" ]]; then
    ssh -p "${SSH_PORT}" "$@"
  else
    ssh "$@"
  fi
}

verify_remote_maintenance_active() {
  local target=$1
  ssh_remote "$target" 'bash -s' <<'REMOTE_MAINTENANCE_GUARD'
set -euo pipefail
sentinel=/run/alumni/maintenance
mode=$(stat -c '%a' "$sentinel" 2>/dev/null) || exit 1
owner=$(stat -c '%u' "$sentinel" 2>/dev/null) || exit 1
[[ -f $sentinel && ! -L $sentinel && $mode == 644 && $owner == 0 ]] || exit 1
[[ $(grep -Fxc 'state=active' "$sentinel" || true) == 1 ]] || exit 1
[[ $(grep -Ec '^generation=[a-f0-9]{32}$' "$sentinel" || true) == 1 ]] || exit 1
REMOTE_MAINTENANCE_GUARD
}

# Resolve --patch-mode interactively if it wasn't provided on the command line.
# No default — the user must explicitly answer true or false (mirrors the
# /dev/tty prompt pattern used by the migration drift check below).
if [[ "${BUILD_FRONTEND}" == "true" && -z "${PATCH_MODE}" ]]; then
  if [[ ! -e /dev/tty ]]; then
    echo "✗ Non-interactive shell — pass --patch-mode=true or --patch-mode=false." >&2
    exit 1
  fi
  while true; do
    read -r -p "Enable WIP gate (admin-code cover page) on the deployed build? [true/false] " ANSWER < /dev/tty
    case "${ANSWER}" in
      true|TRUE|True)    PATCH_MODE="true";  break ;;
      false|FALSE|False) PATCH_MODE="false"; break ;;
      *) echo "  Please type exactly 'true' or 'false'." ;;
    esac
  done
fi
echo "=== patch-mode = ${PATCH_MODE} ==="

if [[ "${BUILD_BACKEND}" == "false" && "${BUILD_FRONTEND}" == "false" ]]; then
  echo "✗ both --backend=false and --frontend=false. Nothing to deploy."
  exit 1
fi
if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE != 0 && $RECORD_MAINTENANCE_DEPLOY_EVIDENCE != 1 ]]; then
  echo "✗ RECORD_MAINTENANCE_DEPLOY_EVIDENCE must be 0 or 1." >&2
  exit 1
fi
if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 && $BUILD_BACKEND != true ]]; then
  echo "✗ Deployment evidence requires --backend=true." >&2
  exit 1
fi
if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 && $BUILD_FRONTEND != false ]]; then
  echo "✗ Evidence-enabled deployment requires --frontend=false; Apache/PHP pre-stage is a separate approved gate." >&2
  exit 1
fi
if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 &&
      ( ${SKIP_ENV_CHECK:-0} == 1 || ${SKIP_DEBUG_AGENT_CHECK:-0} == 1 || ${SKIP_MIGRATION_CHECK:-0} == 1 ) ]]; then
  echo "✗ Evidence-enabled deployment does not allow SKIP_* bypasses." >&2
  exit 1
fi
if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 ]]; then
  [[ $APPROVED_SOURCE_REVISION =~ ^[a-f0-9]{40}$ ]] || {
    echo "✗ APPROVED_SOURCE_REVISION must be an exact 40-character commit ID." >&2
    exit 1
  }
  [[ $APPROVED_BACKEND_ARTIFACT_SHA256 =~ ^[a-f0-9]{64}$ ]] || {
    echo "✗ APPROVED_BACKEND_ARTIFACT_SHA256 must be an exact SHA-256." >&2
    exit 1
  }
  [[ $(git rev-parse HEAD) == "$APPROVED_SOURCE_REVISION" ]] || {
    echo "✗ Current source revision does not match the approved revision." >&2
    exit 1
  }
  [[ -z $(git status --porcelain) ]] || {
    echo "✗ Evidence-enabled deployment requires a clean worktree." >&2
    exit 1
  }
fi
if [[ "${BUILD_FRONTEND}" == "false" && -z "${PATCH_MODE}" ]]; then
  PATCH_MODE="false"
  echo "=== patch-mode is forced false because frontend build is disabled ==="
fi
echo "=== build targets: backend=${BUILD_BACKEND}, frontend=${BUILD_FRONTEND} ==="

# =============================================================================
# DATABASE MIGRATIONS — MANUAL STEP REQUIRED
# =============================================================================
# Migrations are NOT auto-applied by this script.
# Production prerequisite history is canonical through 035. Checksummed auth
# migrations 036-039 are intentionally excluded from this deploy path and must
# be applied sequentially by scripts/kakao-auth-rollout/apply-migrations.sh only,
# after its separate backup, maintenance, DB-drain, and migration approval gates.
# =============================================================================

# =============================================================================
# REMOTE ENVIRONMENTFILE VALIDATION
# Credential values stay on the production host. The remote validator returns
# only key names and status metadata; values never enter local shell variables.
# =============================================================================
ENV_FILE_PATH="/etc/sysconfig/alumni-backend"

run_remote_env_validation() {
  local check_required=$1
  local check_debug=$2
  ssh_remote "${TARGET}" \
    "CHECK_REQUIRED=${check_required} CHECK_DEBUG=${check_debug} ENV_FILE_PATH=${ENV_FILE_PATH} php" <<'PHP'
<?php
// ENV_VALIDATOR_PROTOCOL=1
$path = getenv('ENV_FILE_PATH');
$checkRequired = getenv('CHECK_REQUIRED') === '1';
$checkDebug = getenv('CHECK_DEBUG') === '1';

if (!is_readable($path) || !is_file($path) || is_link($path)) {
    echo "ERROR ENV_FILE_UNREADABLE\n";
    exit(1);
}

$mode = fileperms($path) & 0777;
$owner = fileowner($path);
$group = filegroup($path);
$groupInfo = function_exists('posix_getgrgid') ? posix_getgrgid($group) : false;
$groupName = is_array($groupInfo) && isset($groupInfo['name']) ? $groupInfo['name'] : '';
$modeIsApproved = ($owner === 0 && $mode === 0600) ||
    ($owner === 0 && $mode === 0640 && $groupName === 'alumni-backend');
if (!$modeIsApproved) {
    echo "ERROR ENV_FILE_OWNERSHIP_OR_MODE\n";
    exit(1);
}

$env = parse_ini_file($path, false, INI_SCANNER_RAW);
if ($env === false) {
    echo "ERROR ENV_FILE_PARSE_FAILED\n";
    exit(1);
}

// ACTIVE_UNIT_PROTOCOL=1
$unitLines = array();
$unitStatus = 0;
exec("/bin/systemctl show alumni-backend 2>/dev/null | /bin/grep -E '^EnvironmentFiles?='", $unitLines, $unitStatus);
$effectiveEnvironmentFile = '';
$environmentFileLineCount = 0;
foreach ($unitLines as $line) {
    if (preg_match('/^EnvironmentFiles?=(.*)$/', $line, $matches) === 1) {
        $environmentFileLineCount++;
        $effectiveEnvironmentFile = trim($matches[1]);
    }
}
if ($unitStatus !== 0 ||
    $environmentFileLineCount !== 1 ||
    !preg_match('#^/etc/sysconfig/alumni-backend(?: \\(ignore_errors=(?:yes|no)\\))?$#', $effectiveEnvironmentFile)) {
    echo "INVALID ACTIVE_UNIT_ENVIRONMENTFILE\n";
    exit(1);
}

$failed = false;
if ($checkRequired) {
    $required = array(
        'ALLOWED_ORIGIN', 'SITE_BASE_URL', 'DB_USER', 'DB_PASSWORD', 'DB_NAME',
        'KAKAO_CLIENT_ID', 'KAKAO_CLIENT_SECRET', 'KAKAO_REDIRECT_URI',
        'JWT_SECRET', 'UPLOAD_LEGACY_PATH', 'EASYPAY_IMMEDIATELY_MALL_ID',
        'EASYPAY_PROFILE_MALL_ID', 'EASYPAY_GW_URL', 'EASYPAY_BIN_BASE',
        'EASYPAY_RETURN_BASE_URL', 'SMTP_HOST', 'SMTP_USER', 'SMTP_PASSWORD',
        'VISIT_IP_SALT', 'ENV', 'MAINTENANCE_SENTINEL_PATH',
        'MAINTENANCE_SMOKE_PROOF_SHA256', 'MAINTENANCE_SMOKE_ALLOWED_PATHS'
    );
    $placeholders = array(
        'JWT_SECRET' => 'change-me-in-production',
        'EASYPAY_GW_URL' => 'testgw.easypay.co.kr',
        'ENV' => 'dev'
    );
    foreach ($required as $key) {
        if (!isset($env[$key]) || trim((string)$env[$key]) === '') {
            echo "MISSING " . $key . "\n";
            $failed = true;
        } elseif (isset($placeholders[$key]) && hash_equals($placeholders[$key], (string)$env[$key])) {
            echo "PLACEHOLDER " . $key . "\n";
            $failed = true;
        }
    }
    if (isset($env['MAINTENANCE_SENTINEL_PATH']) &&
        !hash_equals('/run/alumni/maintenance', (string)$env['MAINTENANCE_SENTINEL_PATH'])) {
        echo "INVALID MAINTENANCE_SENTINEL_PATH\n";
        $failed = true;
    }
    if (isset($env['MAINTENANCE_SMOKE_PROOF_SHA256']) &&
        !preg_match('/^[a-f0-9]{64}$/', (string)$env['MAINTENANCE_SMOKE_PROOF_SHA256'])) {
        echo "INVALID MAINTENANCE_SMOKE_PROOF_SHA256\n";
        $failed = true;
    }
    if (isset($env['MAINTENANCE_SMOKE_ALLOWED_PATHS'])) {
        $paths = array_map('trim', explode(',', (string)$env['MAINTENANCE_SMOKE_ALLOWED_PATHS']));
        sort($paths);
        if ($paths !== array('/api/auth/login', '/api/auth/logout')) {
            echo "INVALID MAINTENANCE_SMOKE_ALLOWED_PATHS\n";
            $failed = true;
        }
    }
}

if ($checkDebug) {
    $debugKeys = array(
        'DEBUG_AGENT_ENDPOINT', 'DEBUG_AGENT_PROJECT',
        'DEBUG_AGENT_SECRET', 'DEBUG_AGENT_ENVIRONMENT'
    );
    $present = array();
    $missing = array();
    foreach ($debugKeys as $key) {
        if (isset($env[$key]) && trim((string)$env[$key]) !== '') {
            $present[] = $key;
        } else {
            $missing[] = $key;
        }
    }
    if (count($present) > 0 && count($missing) > 0) {
        foreach ($present as $key) {
            echo "DEBUG_PRESENT " . $key . "\n";
        }
        foreach ($missing as $key) {
            echo "DEBUG_MISSING " . $key . "\n";
        }
        $failed = true;
    }
    if (isset($env['DEBUG_AGENT_SECRET']) &&
        in_array((string)$env['DEBUG_AGENT_SECRET'], array('change-me', 'test-secret'), true)) {
        echo "DEBUG_PLACEHOLDER DEBUG_AGENT_SECRET\n";
        $failed = true;
    }
    if (isset($env['DEBUG_AGENT_ENVIRONMENT']) && $env['DEBUG_AGENT_ENVIRONMENT'] === 'dev') {
        echo "DEBUG_PLACEHOLDER DEBUG_AGENT_ENVIRONMENT\n";
        $failed = true;
    }
}

if ($failed) {
    exit(1);
}
echo "STATUS OK\n";
PHP
}

if [[ "${BUILD_BACKEND}" == "false" ]]; then
  echo "=== Skipping production EnvironmentFile validation because backend build is disabled ==="
else
  CHECK_REQUIRED=1
  CHECK_DEBUG=1
  if [[ "${SKIP_ENV_CHECK:-0}" == "1" ]]; then
    CHECK_REQUIRED=0
    echo "=== Skipping required env validation (SKIP_ENV_CHECK=1) ==="
  fi
  if [[ "${SKIP_DEBUG_AGENT_CHECK:-0}" == "1" ]]; then
    CHECK_DEBUG=0
    echo "=== Skipping debug agent validation (SKIP_DEBUG_AGENT_CHECK=1) ==="
  fi
  if [[ ${CHECK_REQUIRED} -eq 1 || ${CHECK_DEBUG} -eq 1 ]]; then
    echo "=== Validating production EnvironmentFile on ${TARGET} ==="
    set +e
    ENV_CHECK_OUTPUT=$(run_remote_env_validation "${CHECK_REQUIRED}" "${CHECK_DEBUG}" 2>&1)
    ENV_CHECK_STATUS=$?
    set -e
    if [[ ${ENV_CHECK_STATUS} -ne 0 ]]; then
      echo "✗ Production EnvironmentFile validation failed:" >&2
      printf '%s\n' "${ENV_CHECK_OUTPUT}" | sed 's/^/    /' >&2
      exit 1
    fi
    echo "✓ production EnvironmentFile validation passed"
  fi
fi

# =============================================================================
# MIGRATION DRIFT CHECK — fail if local backend/migrations/ has unapplied files
# on the production DB. Migration history is tracked in the _migration_history
# table created by migrate.sh. DB credentials remain inside the remote process.
# Skip with: SKIP_MIGRATION_CHECK=1 ./deploy.sh ...
# =============================================================================
run_remote_migration_history() {
  ssh_remote "${TARGET}" "ENV_FILE_PATH=${ENV_FILE_PATH} php" <<'PHP'
<?php
// MIGRATION_HISTORY_PROTOCOL=1
$path = getenv('ENV_FILE_PATH');
$env = is_readable($path) ? parse_ini_file($path, false, INI_SCANNER_RAW) : false;
$required = array('DB_USER', 'DB_PASSWORD', 'DB_NAME');
if ($env === false) {
    echo "EnvironmentFile unavailable or invalid\n";
    exit(1);
}
foreach ($required as $key) {
    if (!isset($env[$key]) || trim((string)$env[$key]) === '') {
        echo "Missing database configuration key: " . $key . "\n";
        exit(1);
    }
}
$host = isset($env['DB_HOST']) && $env['DB_HOST'] !== '' ? $env['DB_HOST'] : '127.0.0.1';
$port = isset($env['DB_PORT']) && $env['DB_PORT'] !== '' ? $env['DB_PORT'] : '3306';
putenv('MYSQL_PWD=' . $env['DB_PASSWORD']);
$command = 'mysql --skip-ssl'
    . ' -h ' . escapeshellarg($host)
    . ' -P ' . escapeshellarg($port)
    . ' -u ' . escapeshellarg($env['DB_USER'])
    . ' -BN ' . escapeshellarg($env['DB_NAME'])
    . ' -e ' . escapeshellarg('SELECT filename FROM _migration_history ORDER BY filename');
passthru($command, $status);
exit($status);
PHP
}

if [[ "${BUILD_BACKEND}" == "false" ]]; then
  echo "=== Skipping migration drift check because backend build is disabled ==="
elif [[ "${SKIP_MIGRATION_CHECK:-0}" == "1" ]]; then
  echo "=== Skipping migration drift check (SKIP_MIGRATION_CHECK=1) ==="
else
  echo "=== Checking migration drift on ${TARGET} ==="

  if ! REMOTE_LIST=$(run_remote_migration_history 2>&1); then
    echo "✗ Failed to query _migration_history on ${TARGET}:" >&2
    printf '    %s\n' "${REMOTE_LIST//$'\n'/$'\n    '}" >&2
    echo "" >&2
    echo "If _migration_history doesn't exist (fresh DB or pre-migrate.sh era):" >&2
    echo "    1. On the server, run: ./migrate.sh --seed NNN  (NNN = highest already-applied migration number)" >&2
    echo "    2. Re-run ./deploy.sh" >&2
    echo "  Or set SKIP_MIGRATION_CHECK=1 to bypass (NOT recommended)." >&2
    exit 1
  fi

  LOCAL_LIST=$(
    cd backend/migrations
    for migration_file in [0-9][0-9][0-9]_*.sql; do
      [[ -f "${migration_file}" ]] || continue
      [[ "${migration_file}" =~ ^[0-9]{3}_[A-Za-z0-9._-]+\.sql$ ]] || continue
      printf '%s\n' "${migration_file}"
    done | LC_ALL=C sort
  )

  PENDING=()
  while IFS= read -r f; do
    [[ -z "$f" ]] && continue
    if ! grep -Fxq "$f" <<< "${REMOTE_LIST}"; then
      PENDING+=("$f")
    fi
  done <<< "${LOCAL_LIST}"

  if [[ ${#PENDING[@]} -gt 0 ]]; then
    echo "" >&2
    echo "⚠ Migration drift detected — ${#PENDING[@]} unapplied migration(s) on ${TARGET}:" >&2
    for m in "${PENDING[@]}"; do
      echo "    - ${m}" >&2
    done
    echo "" >&2

    echo "✗ deploy.sh never applies migrations." >&2
    echo "  Resolve drift through the separately approved checksummed migration runner." >&2
    echo "  Backend was not built, uploaded, or restarted." >&2
    exit 1
  fi

  TOTAL=$(printf '%s\n' "${LOCAL_LIST}" | grep -c .)
  echo "✓ All ${TOTAL} migrations applied on remote"
fi

if [[ "${PREFLIGHT_ONLY}" == "true" ]]; then
  echo "=== Preflight complete; no build, upload, restart, or reload performed ==="
  exit 0
fi

if [[ "${BUILD_BACKEND}" == "true" ]]; then
  echo "=== Building Go backend (linux/amd64) ==="
  [[ $(GOTOOLCHAIN=local go env GOVERSION) == go1.25.2 ]] || {
    echo "✗ Backend artifact build requires local Go 1.25.2." >&2
    exit 1
  }
  cd backend
  GOWORK=off GOFLAGS='' GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -trimpath -buildvcs=false -o ../dist/server ./cmd/server
  chmod 755 ../dist/server
  if [[ $BUILD_BACKFILL_ARTIFACT == 1 ]]; then
    GOWORK=off GOFLAGS='' GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build -trimpath -buildvcs=false -o ../dist/backfill ./cmd/backfill
    chmod 755 ../dist/backfill
  fi
  cd ..
fi

if [[ "${BUILD_FRONTEND}" == "true" ]]; then
  echo "=== Building User SPA (patch-mode=${PATCH_MODE}) ==="
  cd frontend
  npm ci
  if [[ "${PATCH_MODE}" == "true" ]]; then
    # patch-mode=true requires a non-empty VITE_WIP_ADMIN_CODE in frontend/.env so
    # the WipGate component (frontend/src/components/common/WipGate.tsx) is active.
    WIP_CODE_VAL=""
    if [[ -f .env.production ]]; then
      WIP_CODE_VAL=$(grep -E '^VITE_WIP_ADMIN_CODE[[:space:]]*=' .env.production | tail -n 1 | sed -E 's/^VITE_WIP_ADMIN_CODE[[:space:]]*=[[:space:]]*"?([^"]*)"?[[:space:]]*$/\1/')
    fi
    if [[ -z "${WIP_CODE_VAL}" && -f .env ]]; then
      WIP_CODE_VAL=$(grep -E '^VITE_WIP_ADMIN_CODE[[:space:]]*=' .env | tail -n 1 | sed -E 's/^VITE_WIP_ADMIN_CODE[[:space:]]*=[[:space:]]*"?([^"]*)"?[[:space:]]*$/\1/')
    fi
    if [[ -z "${WIP_CODE_VAL}" ]]; then
      echo "✗ patch-mode=true requires VITE_WIP_ADMIN_CODE to be non-empty in frontend/.env.production (preferred) or frontend/.env" >&2
      echo "  Either set the code in one of these files or re-run with --patch-mode=false." >&2
      exit 1
    fi
    npm run build
  else
    # Force-empty so .env's value is overridden at build time. The repo .env stays untouched.
    # Vite's loadEnv merges process.env over .env for VITE_*-prefixed vars, so this wins.
    VITE_WIP_ADMIN_CODE="" npm run build
  fi
  cd ..

  echo "=== Building Admin SPA ==="
  cd admin
  npm ci
  npm run build
  cd ..
fi

if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 ]]; then
  verify_remote_maintenance_active "$TARGET" || {
    echo "✗ Evidence deploy requires active canonical maintenance before upload." >&2
    exit 1
  }
fi

if [[ "${BUILD_BACKEND}" == "true" ]]; then
  echo "=== Uploading Go binary ==="
  BACKEND_ARTIFACT_SHA256=$(shasum -a 256 dist/server | cut -d ' ' -f 1)
  [[ $BACKEND_ARTIFACT_SHA256 =~ ^[a-f0-9]{64}$ ]] || {
    echo "✗ Failed to calculate backend artifact SHA-256." >&2
    exit 1
  }
  if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 &&
        $BACKEND_ARTIFACT_SHA256 != "$APPROVED_BACKEND_ARTIFACT_SHA256" ]]; then
    echo "✗ Built backend digest does not match the approved artifact digest." >&2
    exit 1
  fi
  if [[ $BUILD_BACKFILL_ARTIFACT == 1 ]]; then
    ssh "${SSH_OPTS[@]}" "${TARGET}" \
      "set -euo pipefail; umask 077; staged=\$(mktemp /app/backend/.backfill.new.XXXXXX); \
       trap 'rm -f \"\$staged\"' EXIT; cat > \"\$staged\"; chmod 0755 \"\$staged\"; \
       mv -fT \"\$staged\" /app/backend/backfill; trap - EXIT" < dist/backfill
  fi
  BACKEND_ROLLBACK_PATH=$(
    ssh "${SSH_OPTS[@]}" "${TARGET}" \
      "set -euo pipefail; umask 077; \
       if [[ ${RECORD_MAINTENANCE_DEPLOY_EVIDENCE} == 1 ]]; then \
         sentinel=/run/alumni/maintenance; mode=\$(stat -c '%a' \"\$sentinel\" 2>/dev/null) || { echo maintenance_guard_failed_before_backend_replace >&2; exit 1; }; \
         owner=\$(stat -c '%u' \"\$sentinel\" 2>/dev/null) || { echo maintenance_guard_failed_before_backend_replace >&2; exit 1; }; \
         [[ -f \"\$sentinel\" && ! -L \"\$sentinel\" && \"\$mode\" == 644 && \"\$owner\" == 0 && \$(grep -Fxc 'state=active' \"\$sentinel\" || true) == 1 && \$(grep -Ec '^generation=[a-f0-9]{32}$' \"\$sentinel\" || true) == 1 ]] || { echo maintenance_guard_failed_before_backend_replace >&2; exit 1; }; \
       fi; \
       staged=\$(mktemp /app/backend/.server.new.XXXXXX); \
       trap 'rm -f \"\$staged\"' EXIT; cat > \"\$staged\"; \
       actual=\$(sha256sum \"\$staged\" | cut -d ' ' -f 1); \
       [[ \"\$actual\" == '${BACKEND_ARTIFACT_SHA256}' ]] || exit 41; \
       rollback=NONE; \
       if [[ -e /app/backend/server ]]; then \
         [[ -f /app/backend/server && ! -L /app/backend/server ]] || exit 42; \
         rollback=\$(mktemp /app/backend/.server.rollback.XXXXXX); \
         cp -p -- /app/backend/server \"\$rollback\"; \
       fi; \
       chmod 0755 \"\$staged\"; mv -fT \"\$staged\" /app/backend/server; \
       trap - EXIT; printf '%s\\n' \"\$rollback\"" < dist/server
  )
  [[ $BACKEND_ROLLBACK_PATH == NONE ||
     $BACKEND_ROLLBACK_PATH =~ ^/app/backend/\.server\.rollback\.[A-Za-z0-9]+$ ]] || {
    echo "✗ Remote backend rollback artifact is invalid." >&2
    exit 1
  }
fi

restore_backend_without_restart() {
  [[ ${BACKEND_ROLLBACK_PATH:-NONE} != NONE ]] || return 1
  ssh "${SSH_OPTS[@]}" "${TARGET}" \
    "set -euo pipefail; \
     sudo systemctl stop alumni-backend; \
     [[ -f '${BACKEND_ROLLBACK_PATH}' && ! -L '${BACKEND_ROLLBACK_PATH}' ]]; \
     mv -fT '${BACKEND_ROLLBACK_PATH}' /app/backend/server; chmod 0755 /app/backend/server; \
     ! sudo systemctl is-active --quiet alumni-backend; \
     pid=\$(sudo systemctl show --property MainPID --value alumni-backend); [[ \$pid == 0 ]]"
}

restore_backend_for_failed_restart() {
  restore_backend_without_restart || return 1
  if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE != 1 ]]; then
    ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl restart alumni-backend'
  fi
}

# BEGIN_FRONTEND_DEPLOY_SIDE_EFFECTS
if [[ "${BUILD_FRONTEND}" == "true" ]]; then
  echo "=== Uploading User SPA ==="
  rsync -avz --delete --chmod=Du=rwx,Dgo=rx,Fu=rw,Fgo=r -e "ssh ${SSH_OPTS[*]}" frontend/dist/ "${TARGET}:/var/www/app/"

  echo "=== Uploading Admin SPA ==="
  rsync -avz --delete --chmod=Du=rwx,Dgo=rx,Fu=rw,Fgo=r -e "ssh ${SSH_OPTS[*]}" admin/dist/ "${TARGET}:/var/www/admin/"

  echo "=== Installing Apache httpd config ==="
  ssh "${SSH_OPTS[@]}" "${TARGET}" 'set -euo pipefail; umask 077; incoming=$(mktemp /tmp/alumni-httpd.XXXXXX); backup=$(mktemp /tmp/alumni-httpd-backup.XXXXXX); trap '\''rm -f "$incoming" "$backup"'\'' EXIT; cat > "$incoming"; had_previous=0; if sudo test -f /etc/httpd/conf.d/alumni.conf; then sudo cp -p -- /etc/httpd/conf.d/alumni.conf "$backup"; had_previous=1; fi; sudo install -o root -g root -m 0644 "$incoming" /etc/httpd/conf.d/alumni.conf; if ! sudo httpd -t; then if [[ $had_previous == 1 ]]; then sudo install -o root -g root -m 0644 "$backup" /etc/httpd/conf.d/alumni.conf; else sudo rm -f /etc/httpd/conf.d/alumni.conf; fi; exit 43; fi' < deploy/httpd-alumni.conf

  echo "=== Installing legacy PHP compat shims ==="
  for shim in _set_docroot.php _maintenance_gate.php _legacy_docroot.php _legacy_url_rewriter.php; do
    ssh "${SSH_OPTS[@]}" "${TARGET}" \
      "set -euo pipefail; umask 077; incoming=\$(mktemp /tmp/alumni-shim.XXXXXX); \
       trap 'rm -f \"\$incoming\"' EXIT; cat > \"\$incoming\"; \
       sudo install -o root -g root -m 0644 \"\$incoming\" '/var/www/html/${shim}'" < "deploy/${shim}"
  done

  echo "=== Reloading Apache httpd ==="
  ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl reload httpd'

  echo "=== Verifying /old/ legacy routing ==="
  SMOKE_HOST="daeilfoundation.or.kr"
  ssh "${SSH_OPTS[@]}" "${TARGET}" \
    "status=\$(curl --disable --noproxy '*' --silent --show-error --max-time 10 --output /dev/null --write-out '%{http_code}' --resolve ${SMOKE_HOST}:443:127.0.0.1 https://${SMOKE_HOST}/old/index.php); [[ \$status == 200 || \$status == 503 ]]"
fi
# END_FRONTEND_DEPLOY_SIDE_EFFECTS

if [[ "${BUILD_BACKEND}" == "true" ]]; then
  echo "=== Reloading systemd and restarting backend ==="
  if ! ssh "${SSH_OPTS[@]}" "${TARGET}" 'sudo systemctl daemon-reload && sudo systemctl restart alumni-backend'; then
    if ! restore_backend_for_failed_restart; then
      echo "✗ Backend restart and rollback recovery both failed; backend remains fail-closed." >&2
      exit 1
    fi
    if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 ]]; then
      echo "✗ Backend restart failed; previous binary was restored and intentionally left stopped." >&2
    else
      echo "✗ Backend restart failed; previous binary was restored and restarted." >&2
    fi
    exit 1
  fi
fi

if [[ $RECORD_MAINTENANCE_DEPLOY_EVIDENCE == 1 ]]; then
  echo "=== Recording generation-bound backend deployment evidence ==="
  if ! ssh "${SSH_OPTS[@]}" "${TARGET}" \
    "sudo env MAINTENANCE_DEPLOY_EVIDENCE_APPROVED=1 BACKEND_EXPECTED_SHA256=${BACKEND_ARTIFACT_SHA256} BACKEND_ROLLBACK_PATH=${BACKEND_ROLLBACK_PATH} /bin/bash -s" \
    < scripts/kakao-auth-rollout/maintenance-record-deployment.sh; then
    if ! restore_backend_without_restart; then
      echo "✗ Backend evidence and rollback recovery both failed; backend state is uncertain and maintenance must remain active." >&2
      exit 1
    fi
    echo "✗ Backend evidence failed; previous binary was restored and left stopped." >&2
    exit 1
  fi
fi

echo "=== Deploy complete ==="
