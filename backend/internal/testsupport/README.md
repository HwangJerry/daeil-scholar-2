# Backend test helpers

For mobile end-to-end tests, use the [local Docker Compose backend](../../../e2e/local-backend/README.md).
One command starts the pinned MariaDB baseline and API with disposable synthetic accounts.

## MariaDB 10.1.38

Install package cleanup once in `main_test.go`:

```go
func TestMain(m *testing.M) {
    os.Exit(mariadb.Run(m))
}
```

Each test requests its own database. SQL inputs run in order and can mix files
with statements; provide only the minimal schema and synthetic data needed.
No full migration bootstrap is attempted (test strategy O1).

```go
database := mariadb.Start(t).NewDatabase(t,
    mariadb.Statement("CREATE TABLE fixture (id INT PRIMARY KEY)"),
    mariadb.File("testdata/seed.sql"),
)
db, dsn := database.DB, database.DSN
```

`DFLH_DOCKER_TESTS=1 go test ./internal/repository ./internal/testsupport/mariadb -count=1`
enables the migrated test and harness integration test. Tests skip if Docker is
missing or its daemon is unavailable. Image/startup/schema failures fail the test.
The image digest comes from `migrations/testdata/mariadb-10.1.38.image`.
The first enabled test starts one container; `TestMain` removes it and its volumes
after all tests finish. Each test closes its pool and drops its database in
`t.Cleanup`, including setup failures. Parallel tests can use separate databases.
Close any extra pools opened from `DSN` in that test's cleanup too.

`Start(t, "LEGACY_DOCKER_INTEGRATION")` also accepts an existing opt-in variable.
Only `TestAccountDeletionUpdatesOnlyMemberStatusOnMariaDB101` is migrated here;
other existing integration tests retain their own gates and container lifecycle.

## Golden JSON

```go
golden.Assert(t, "login", recorder.Body.Bytes(), "requestId", "elapsedMs")
```

Fixtures live in the caller's `testdata/golden/login.json`. Run
`GOLDEN_UPDATE=1 go test ./path/to/package -count=1` to create/update them, then
review the diff. Normal runs fail on a missing or changed fixture.

`golden.Normalize(body, volatileKeys...)` can also be used separately. It sorts
object keys and pretty-prints JSON, keeping array order and numeric precision.
Whole-string RFC3339 (including fractional seconds/offsets) and SQL datetime
values become `<timestamp>`. JWTs and nonempty strings under keys ending in
`token`/`tokens` (case-insensitive, ignoring `_`/`-`) or named `jwt` become
`<token>`; Bearer values become `Bearer <token>`. Other opaque values must be
identified with volatile keys, which become `<volatile>` at any depth.
Nulls and empty token strings remain visible to catch contract regressions.
This is a fixture stabilizer for synthetic responses, not a data anonymizer.

### Tier 1 router goldens (TS07)

`cmd/server/tier1_golden_test.go` exercises `wireDeps` and `registerRoutes` through
`httptest.NewServer`, using a fresh production-baseline MariaDB database and the
real middleware, handlers, services and repositories. Fixtures are committed in
`cmd/server/testdata/golden/`:

| Files (`.json`) | Contract |
| --- | --- |
| `settings_public` | Public settings, including both update policies as JSON **strings** |
| `check_id_available`, `check_phone_taken` | Availability booleans (both HTTP 200) |
| `phone_verification_request`, `phone_verification_confirm` | Review-number OTP request and grant |
| `register_success`, `register_id_taken` | Pending native signup (201) and duplicate ID (409) |
| `mobile_login_success`, `mobile_login_invalid_password` | Mobile session and invalid credentials (401) |
| `refresh_success`, `auth_me` | Rotated session and approved member |
| `app_update_required_426` | Forced update error below the minimum build |
| `account_deletion_request`, `account_deletion_receipt` | Cancellable deletion acceptance (202) and unauthenticated receipt lookup |

From `backend/`, regenerate and then compare without update mode:

```bash
DFLH_DOCKER_TESTS=1 GOLDEN_UPDATE=1 go test ./cmd/server -run Golden -count=1
DFLH_DOCKER_TESTS=1 go test ./cmd/server -run Golden -count=1 -v
```

Docker-off runs skip the four top-level golden tests. Docker-unavailable runs
also skip cleanly through the shared harness. Review generated diffs before
committing; snapshots are always captured from HTTP responses.

The harness loads configuration from explicitly set test environment variables,
never env files. SMS delivery and push are disabled; `01000000001` / `123456`
exercise the configured review-number OTP path. Erasure requests are enabled
with distinct synthetic keys, manual external processing and temporary storage;
`validateErasureRuntime` must pass. Erasure/retention workers and all other
background jobs are not started. The schema-only baseline has no identity
backfill records, so synthetic completed run/journal rows activate canonical
password wiring. Approved members and their companion rows are synthetic SQL
seeds; signup itself creates its rows through HTTP. No schema alterations or
production test seams are used.

Request shapes were checked against Android `core/auth/AuthApi.kt`,
`core/network/AppClientHeaders.kt`, and `AccountDeletionRepository.kt`, and iOS
`Feature/Login/AuthRepository.swift`, `Network/AppClientHeaders.swift`, and
`Infrastructure/Token/TokenRefresher.swift`. Both send `{usrId, password}` to
`POST /api/auth/mobile/login`, JSON content type, and bearer auth on protected
requests. Session tests run with both platforms' release headers against
separate databases/caches. Both apps use **POST**, not GET, for the public
`/api/account-deletion/receipt` lookup, with `{receiptToken}` in the body.

Explicit volatile keys are `verificationId` for the OTP request and `sid`,
`jti`, `accessIssuedAt`, `accessExpiresAt`, `refreshExpiresAt` for login/refresh.
The helper automatically normalizes tokens and timestamp strings. Fixed member
and receipt IDs remain numeric. Raw responses are decoded and their business
facts checked before normalization, including numeric token lifetimes, password
credentials, consent, phone claims, OTP consumption, refresh rotation and
deletion/session revocation. Downstream DTO tests (TS08) must materialize the
normalizer placeholders: epoch fields marked `<volatile>` need JSON numbers,
timestamp markers need valid date strings, and opaque ID/token markers need
synthetic strings. The signup response intentionally preserves the server's
current empty email/verification fields.

The coverage script instruments each package separately. To include these
`cmd/server` tests in handler/service coverage, additionally run:

```bash
DFLH_DOCKER_TESTS=1 go test ./... \
  -coverpkg=./internal/handler,./internal/service -coverprofile=/tmp/ts07-coverage.out
```

## Coverage

Run `bash scripts/test-coverage.sh` from `backend/` for per-package statement
coverage and the weighted total. Its temporary profile is removed on exit.
Docker integrations are opt-in; `DFLH_DOCKER_TESTS=1` includes shared-harness
tests, while unmigrated tests still require their original opt-in variables.
The script clears `SOCIAL_LINK_TEST_DSN` so it cannot use an inherited external DB.

## Production baseline schema

`mariadb.ProdBaseline(t)` returns the production schema (no rows) as of migration 077 plus its
`_migration_history` rows, taken read-only from daeil-prod on 2026-09-27
(`migrations/testdata/prod_baseline_schema_20260927.sql`, `prod_baseline_applied_migrations_20260927.sql`).

```go
db := mariadb.Start(t).NewDatabase(t, append(mariadb.ProdBaseline(t),
    mariadb.Statement("INSERT INTO WEO_MEMBER (...) VALUES (...)"), // synthetic seed only
)...).DB
```

The container starts with the production server options that affect DDL/SQL (`innodb_file_format=Barracuda`,
`innodb_large_prefix=ON`, permissive `sql_mode`). Apply migrations newer than 077 with `mariadb.File`.
To refresh the baseline, re-run the read-only `mysqldump --no-data --routines --triggers --events`,
strip `AUTO_INCREMENT`/`DEFINER` and `DELIMITER` lines, and replace both files together.
