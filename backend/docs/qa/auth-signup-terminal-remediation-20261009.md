# Social signup terminal handling and new-password policy

Implemented on `fix/auth-signup-terminal-20261009`, based on deployed `567b6e1`. No production requests, account changes, schema migrations, device changes, push, merge, or deployment were performed by this lane.

## Changes

- Add `POST /api/auth/social/link/cancel {token}`. Ready → cancelled clears all provider data; repeated/missing/expired cancellation returns 204. Processing returns 409 `TOKEN_IN_PROGRESS`; consumed returns 409 `SIGNUP_ALREADY_COMPLETED`. Begin/Cancel/Consume share a mutex; cancelled prefill/photo/signup are rejected.
- Begin occurs before phone-grant validation. Retryable precommit failures release the lease. Processing cache entries retain status/lease/upload metadata without provider credentials; only the active request lease holds provider data. Release restores that data only before the original expiry.
- Immediately after member commit, consume the continuation and preserve the committed photo. Bind the phone grant and record accepted consent even if continuation finalization fails. Session/finalization errors return 500 `SIGNUP_COMPLETED_LOGIN_REQUIRED` and a Korean signup-complete/relogin message. Consumed replay returns 409 `SIGNUP_ALREADY_COMPLETED` before a spent phone-grant check, without minting another session. Fresh verified provider login recovers the existing account.
- Centralize `ValidateNewPassword`: at least 8 UTF-8 bytes, an ASCII letter, an ASCII digit, and a character matching `[^a-zA-Z0-9]`. Apply before registration dependencies/hash/writes and reuse in change/reset. Preserve legacy login/hash-transition behavior and change/reset error messages. Commit the byte-identical shared 18-case JSON fixture.
- Track only actual UploadResults. Cancellation/expiry and losing upload races discard only tracked results; provider/form URLs are never treated as upload ownership. Processing expiry postpones cleanup until commit/rollback, protecting a member photo even when commit exceeds TTL. Cleanup has bounded retries.
- Before profile unlink, lock its exact WEO_FILES row and reuse erasure's surviving owner/history/member/content reference ranges through filesystem unlink and transaction commit. Exclude only the tracked F_SEQ from the file-reference scan. Absolute/relative/HTTP/www/query aliases use the existing canonical reference resolver; joined/owned/current/historical references are preserved. Uploaded multipart temporary files are removed.

## Verification

Baseline RED:

- Unit desired-contract tests failed: weak signup password reached repository dependencies and the atomic Cancel operation was absent.
- Disposable MariaDB HTTP tests failed: cancellation route returned 404 and postcommit refresh-insert fault returned generic `LOGIN_FAILED`.

GREEN:

- Full normal Go suite: 1,556 test/subtest passes, 82 opt-in skips, 0 failures.
- Focused remediation race suite: 43 test/subtest passes, 0 failures/skips.
- Disposable MariaDB HTTP/ownership checks: 12 test/subtest passes, including member rollback, weak password rejection with unconsumed phone grant, committed-account refresh-insert failure, recovery to the same member, spent-token terminal replay, 0 refresh sessions on failure, member-owned consumed proof, consent persistence, cancellation/idempotence/expiry, and profile ownership/lock checks.
- Existing erasure alias/reference regressions plus new profile checks with their explicit integration gate: 33 test/subtest passes, 0 failures/skips.
- Follow-up focused photo/finalization handler checks passed after multipart temporary-file cleanup.

The legacy full-schema import exceeded the harness's 15-second query deadline once under Docker amd64 emulation on this ARM Mac. Integration runs used a temporary 120-second test-harness deadline; the original harness file was restored and is not part of this commit.

A broad race selection also included the pre-existing `TestPasswordHasherUsesGlobalSemaphore`: its 2-second wall-clock completion assertion failed under race instrumentation, and its cleanup left later tests waiting on that semaphore. Only the owned test process was stopped for a diagnostic stack; the test was not changed. The full normal suite passes it. Relevant remediation race tests pass independently. Raw logs and stack are private artifacts under `build/auth-server-remediation-20261009` in the workspace root.

## Limits

Continuation state, timers, and orphan retry metadata remain process-local, matching the existing architecture. If the server restarts after an upload, this lane cannot guarantee durable orphan cleanup; existing owned files remain protected. Restart/expiry invalidates continuation proof and requires fresh provider authentication. No provider consent revocation is triggered by signup cancellation. Processing metadata stays until the active request finalizes; credentials have already been removed from the cache. Physical-device and reviewed deployment validation are owned by the coordinating session.
