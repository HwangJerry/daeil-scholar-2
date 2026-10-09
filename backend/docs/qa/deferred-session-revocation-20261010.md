# Ended-session refresh-proof revocation

Based exactly on deployed `d470410a00429daaf66ad88f5249dc1639c9dbc6`. This lane did not deploy, restart, change production configuration or accounts, interact with Android or modify other worktrees. No schema migration is included. Existing atomic signup code is unchanged and its MariaDB regression cases are rerun.

## Contract

`POST /api/auth/logout/deferred`, `Content-Type: application/json`, body `{"refreshToken":"<original refresh proof>"}`. The endpoint is public and exempt from the app-version gate so blocked builds can still end sessions; an access token or a caller-selected account/session is not accepted as authorization. It issues no cookies, access/refresh tokens or rotated rows, and does not load profile or provider credentials.

- 204, empty body: revoked, original row absent, already revoked or original proof expired. Missing rows and expired proofs never authorize a new revocation.
- 401 `INVALID_REFRESH_TOKEN`: invalid signature/type/metadata or retained-row account/SID/original-expiry mismatch.
- 400 `INVALID_BODY` / `INVALID_REQUEST`: invalid/trailing/unknown-field JSON or missing proof.
- 413 `REQUEST_TOO_LARGE`: body exceeds 16 KiB. 415 `UNSUPPORTED_MEDIA_TYPE`: non-JSON media type.
- 429 `RATE_LIMITED`: the existing limiter is explicitly **10 attempts per 15 minutes**, per trusted client IP and endpoint bucket. This is not a new 10/minute policy. Revocation attempts do not spend login/refresh buckets.
- 500 `LOGOUT_FAILED`: DB query/update/commit or cancellation failure; never reported as successful cleanup. Log only the bounded event code, never the proof or parameter-bearing DB error.

The client retains the original ended-session proof in protected storage only until its original refresh expiry. Retry 429/5xx/network failures within that bound; do not refresh or mint credentials merely to perform cleanup. Legacy access-only sessions retain the existing access-protected best-effort logout path.

## Existing-proof security binding

Deployed refresh tokens are signed HS256 JWTs, not opaque random tokens; their retained table stores random JTI, SID, account and expiry, not the raw token hash. Root approved the compatible signature-plus-retained-row binding rather than a wider migration that cannot backfill previously issued raw token hashes. Clients treat the JWT as an opaque proof.

The server authenticates the signature and exact existing issuer/audience/type/version, requires positive canonical account and 32-character lowercase hex JTI/SID, and validates required expiry/issued/not-before timestamps. It verifies metadata before accepting an expired no-op. A still-valid proof locks its exact original JTI row, checks account/SID/expiry against that row, and updates only that account/SID family in the same transaction. Both canonical and legacy revocation columns are marked. The original signed expiry is compared to DB expiry at its original second precision; it is never extended to a successor expiry.

A consumed rotation ancestor remains a valid revocation proof until its own original expiry; it cannot revoke a later unrelated family or account, even if a synthetic other-account row has the same SID. No row is inserted and no consumed/rotated state is spent by this endpoint. Request-context cancellation rolls back the transaction. A concurrent refresh may win before revocation, but its resulting same-family rows are revoked before successful cleanup. Existing refresh lock-order conflicts can produce a retryable transaction failure rather than a false 204.

## RED to GREEN verification

Actual MariaDB HTTP RED returned 404 for the new route after issuing a session, rotating it and issuing an unrelated session. Desired-contract unit RED showed the ended-session operation was missing. A further RED proved the exact-path app-version gate blocked deferred cleanup with 426; the new path is explicitly exempt and verified through the actual HTTP router. Logs are preserved under the coordinating `build/auth-ios-device-regression-20261010` directory.

GREEN checks cover invalid signer/algorithm/type/issuer/audience/version/claims, deterministic tamper, original issuer proof compatibility, missing/revoked/expired rows, account/SID/expiry mismatch, request-body limits and zero credential cookies. Repository fakes inject query/update/commit failure and cancelled context. Actual disposable MariaDB HTTP tests cover consumed ancestry, replay, unrelated new family and other-account isolation, original-expired ancestor with a live successor, per-endpoint limiter separation, concurrent successor refresh, UPDATE-trigger fault and cancelled HTTP rollback.

The commit-boundary fault test wraps the disposable MariaDB driver and rolls back actual pending writes while returning a synthetic commit error; it then verifies active rows remain and a later retry succeeds. This verifies returned-failure atomicity, not the outcome of an unknowable network failure after the DB committed. For an ambiguous commit, return 500 and retry the same proof; idempotence handles either persisted state safely.

Final suite counts and exact names are in `deferred-logout-verification-summary.json` in the coordinating artifact directory. Counts from overlapping suites are not summed. All DB tests use isolated Docker MariaDB 10.1.38, never an external DSN or production secret. Core HTTP/transaction and race suites used the unchanged 15-second baseline-import harness. The final blocked-build HTTP case hit that existing deadline during baseline import on this ARM Mac; only that run used a temporary 120-second import deadline, restored before commit. The pre-existing broad-race password-hasher wall-clock limitation remains untouched; the focused revocation/session race suite and actual MariaDB HTTP race run independently.

Final GREEN groups, reported separately:

- Full normal suite: 1,608 test/subtest passes, 100 opt-in skips, zero failures.
- Focused session/logout/rate-limit and all app-version gate race suite: 61 passes, zero skips/failures.
- Core actual MariaDB HTTP/transaction plus deployed atomic signup regression: 32 passes, zero skips/failures.
- Actual HTTP refresh/deferred-revoke race: 1 test with three concurrency scenarios, zero failures.
- Final-source actual blocked-build HTTP cleanup: 1 pass, zero skips/failures.

The last group and final-source full/race runs cover the exact-path gate exemption added after the core HTTP run. The test-support import deadline was restored byte-for-byte, with no harness diff.
