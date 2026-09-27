#!/usr/bin/env bash
set -euo pipefail

# Runs only inside the disposable MariaDB entrypoint, over its local socket.
if env | grep -iq 'daeilfoundation\.or\.kr' || grep -iq 'daeilfoundation\.or\.kr' /seed.sql; then
    echo 'Refusing production configuration.' >&2
    exit 1
fi
[[ "$MYSQL_DATABASE" == alumni_e2e && "$MYSQL_USER" == e2e ]]
export MYSQL_PWD="$MYSQL_ROOT_PASSWORD"
sql() { mysql --protocol=socket -uroot --default-character-set=utf8mb4 "$MYSQL_DATABASE" "$@"; }

# The Go harness baseline omits client DELIMITER commands. Restore them around
# mysqldump trigger blocks without changing their SQL or the checked-in baseline.
awk '
    /^\/\*!50003 CREATE.*TRIGGER/ { print "DELIMITER ;;"; trigger = 1 }
    { if (trigger && /\*\/;$/) { print $0 ";"; print "DELIMITER ;"; trigger = 0 } else print }
    END { if (trigger) exit 1 }
' /migrations/testdata/prod_baseline_schema_20260927.sql > /tmp/baseline.sql
sql < /tmp/baseline.sql
sql < /migrations/testdata/prod_baseline_applied_migrations_20260927.sql

for migration in /migrations/[0-9][0-9][0-9]_*.sql; do
    [[ -f "$migration" ]] || continue
    filename="${migration##*/}"
    [[ "$filename" =~ ^[0-9]{3}_[a-zA-Z0-9_]+\.sql$ ]] || exit 1
    checksum="$(sha256sum "$migration" | cut -d ' ' -f 1)"
    applied="$(sql -Nse "SELECT sha256 FROM _migration_history WHERE filename = '$filename'")"
    if [[ -n "$applied" ]]; then
        if [[ "$applied" != "$checksum" ]]; then
            echo "Applied migration checksum mismatch: $filename" >&2
            exit 1
        fi
        continue
    fi
    if grep -iq 'daeilfoundation\.or\.kr' "$migration"; then
        echo "Refusing production reference in new migration: $filename" >&2
        exit 1
    fi
    echo "Applying local migration: $filename"
    sql < "$migration"
    sql -e "INSERT INTO _migration_history (filename, sha256) VALUES ('$filename', '$checksum')"
done

sql < /seed.sql
echo 'Local E2E baseline, migrations and synthetic seed ready.'
