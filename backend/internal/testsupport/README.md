# Backend test helpers

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

## Coverage

Run `bash scripts/test-coverage.sh` from `backend/` for per-package statement
coverage and the weighted total. Its temporary profile is removed on exit.
Docker integrations are opt-in; `DFLH_DOCKER_TESTS=1` includes shared-harness
tests, while unmigrated tests still require their original opt-in variables.
The script clears `SOCIAL_LINK_TEST_DSN` so it cannot use an inherited external DB.
