#!/bin/bash
# migrate.sh — Apply SQL migration files using DB credentials from .env
# Usage: ./migrate.sh [migration_file]
#   No args:         apply all unapplied migrations in order
#   With arg:        apply a single specified migration file
#   --seed N:        mark migrations 001..N as applied without executing SQL

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/backend/.env"
MIGRATIONS_DIR="${SCRIPT_DIR}/backend/migrations"

# Load .env
if [ ! -f "$ENV_FILE" ]; then
  echo "Error: ${ENV_FILE} not found"
  exit 1
fi

while IFS='=' read -r key value; do
  # Skip comments and blank lines
  [[ -z "$key" || "$key" =~ ^# ]] && continue
  # Trim whitespace
  key=$(echo "$key" | xargs)
  value=$(echo "$value" | xargs)
  export "$key=$value"
done < "$ENV_FILE"

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-alumni}"

MYSQL_CMD="mysql --skip-ssl -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USER} ${DB_NAME}"
if [ -n "$DB_PASSWORD" ]; then
  MYSQL_CMD="mysql --skip-ssl -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USER} -p${DB_PASSWORD} ${DB_NAME}"
fi

# Ensure migration tracking table exists
$MYSQL_CMD -e "
CREATE TABLE IF NOT EXISTS _migration_history (
  filename VARCHAR(255) NOT NULL PRIMARY KEY,
  applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
" 2>/dev/null

apply_migration() {
  local file="$1"
  local filename
  filename="$(basename "$file")"

  # Check if already applied
  local applied
  applied=$($MYSQL_CMD -N -e "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}';" 2>/dev/null)

  if [ "$applied" -gt 0 ]; then
    echo "  SKIP  ${filename} (already applied)"
    return 0
  fi

  echo "  APPLY ${filename} ..."
  if $MYSQL_CMD < "$file" 2>&1; then
    $MYSQL_CMD -e "INSERT INTO _migration_history (filename) VALUES ('${filename}');" 2>/dev/null
    echo "  OK    ${filename}"
  else
    echo "  FAIL  ${filename}"
    exit 1
  fi
}

# --seed mode: register migrations as applied without executing SQL
if [ "${1:-}" = "--seed" ]; then
  if [ -z "${2:-}" ]; then
    echo "Usage: ./migrate.sh --seed <up_to_number>"
    echo "  e.g. ./migrate.sh --seed 006  — marks 001~006 as applied"
    exit 1
  fi
  UP_TO="$2"
  echo "=== DB Migration (seed mode) ==="
  echo "  Target: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
  echo "  Seeding migrations up to: ${UP_TO}"
  echo ""
  for file in "$MIGRATIONS_DIR"/*.sql; do
    [ -f "$file" ] || continue
    filename="$(basename "$file")"
    file_num="${filename%%_*}"
    if [ "$file_num" -le "$UP_TO" ] 2>/dev/null; then
      applied=$($MYSQL_CMD -N -e "SELECT COUNT(*) FROM _migration_history WHERE filename='${filename}';" 2>/dev/null)
      if [ "$applied" -gt 0 ]; then
        echo "  SKIP  ${filename} (already in history)"
      else
        $MYSQL_CMD -e "INSERT INTO _migration_history (filename) VALUES ('${filename}');" 2>/dev/null
        echo "  SEED  ${filename}"
      fi
    fi
  done
  echo ""
  echo "=== Seed complete ==="
  exit 0
fi

echo "=== DB Migration ==="
echo "  Target: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
echo ""

if [ $# -ge 1 ]; then
  # Apply single file
  if [ ! -f "$1" ]; then
    echo "Error: file not found: $1"
    exit 1
  fi
  apply_migration "$1"
else
  # Apply all migrations in order
  found=0
  for file in "$MIGRATIONS_DIR"/*.sql; do
    [ -f "$file" ] || continue
    found=1
    apply_migration "$file"
  done
  if [ "$found" -eq 0 ]; then
    echo "  No migration files found in ${MIGRATIONS_DIR}"
  fi
fi

echo ""
echo "=== Done ==="
