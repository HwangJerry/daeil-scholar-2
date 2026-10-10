# Social signup terminal handling and new-password policy

Implemented on `fix/auth-signup-terminal-20261009`, based on deployed `567b6e1`. No production requests, account changes, schema migrations, device changes, push, merge, or deployment were performed by this lane.

## Changes

- Add `POST /api/auth/social/link/cancel {token}`. Ready → cancelled clears all provider data; repeated/missing/expired cancellation returns 204. Processing returns 409 `TOKEN_IN_PROGRESS`; consumed returns 409 `SIGNUP_ALREADY_COMPLETED`. Begin/Cancel/Consume share a mutex; cancelled prefill/photo/signup are rejected.
- Begin occurs before phone-grant validation. Retryable precommit failures release the lease. Processing cache entries retain status/lease/upload metadata without provider credentials; only the active request lease holds provider data. Release restores that data only before the original expiry.
- Immediately after member commit, consume the continuation and preserve the committed photo. Bind the phone grant and record accepted consent even if continuation finalization fails. Session/finalization errors return 500 `SIGNUP_COMPLETED_LOGIN_REQUIRED` and a Korean signup-complete/relogin message. Consumed replay returns 409 `SIGNUP_ALREADY_COMPLETED` before a spent phone-grant check, without minting another session. Fresh verified provider login recovers the existing account.
- Centralize `ValidateNewPassword`: at least 8 UTF-8 bytes, an ASCII letter, an ASCII digit, and a character matching `[^a-zA-Z0-9]`. Apply before registration dependencies/hash/writes and reuse in change/reset. Preserve legacy login/hash-transition behavior and change/reset error messages. Commit the byte-identical shared 18-case JSON fixture.
- Track only actual UploadResults. Cancellation/expiry and losing upload races discard only tracked results; provider/form URLs are never treated as upload ownership. Processing expiry postpones cleanup until commit/rollback, protecting a member photo even when commit exceeds TTL. Cleanup has bounded retries.
- Before profile unlink, lock its exact WEO_FILES primary key under default REPEATABLE READ; official profile claims and managed signup-photo account commits acquire the same key before member/owner writes. Reuse erasure's canonical surviving owner/history/member/content checks as nonlocking reads whose snapshot starts after the exact file lock. Exclude only the tracked F_SEQ from the file-reference scan. Absolute/relative/HTTP/www/query/percent aliases and existing joined/owned/current/historical references are preserved. Cleanup is serialized without blocking unrelated writes. Uploaded multipart temporary files are removed.

## Initial verification (`c493798`)

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

## Opus review follow-up

Independent CLI Opus 5.5 high returned CHANGES_REQUIRED (no P0/P1). Its complete original feedback is preserved in the root review packet. This backend lane fixed P2-1, P3-1, P3-3 and the concrete observability part of P3-4:

- Social-photo cleanup no longer uses broad FOR UPDATE range scans. Only the exact tracked WEO_FILES PK is locked. The first follow-up used READ COMMITTED; the second-review fix below uses the default REPEATABLE READ. Official `AssignProfileUpload` and new managed social signup photo commits lock that file before account/owner writes, reject a missing/mismatched result and roll back when cleanup won. The file-row lock survives reference verification and metadata retirement commit; the orchestrator serializes signup cleanup work. Existing erasure retains its original broad locking behavior.
- Persistent reference scans still examine every existing owner/history/member/content/banner/file alias using the canonical normalizer. They are nonlocking current reads, avoiding raw LIKE filters that miss percent-encoded/base64/entity aliases. Candidate filtering decodes valid percent pairs independently before deciding whether a malformed URL could refer to this exact random filename. A related ambiguous path is retained for review; an unrelated malformed path does not globally disable cleanup. Full reference read cost remains linear in stored reference data; only cleanup is serialized.
- Processing prefill/photo now return 409 TOKEN_IN_PROGRESS.
- Web LoginWithBridge performs fallible login-log/last-login DB writes before setting JWT or PHP cookies. Both injected DB failures issue zero cookies.
- Blocked cleanup returns a typed reason to the orchestrator. A bounded OTLP counter `daeil_social_signup_photo_cleanup_blocked{reason=referenced|review_required|ownership_changed|other}` and a reason/FSeq-only warning distinguish preserved ownership from review-required paths. Tokens, provider credentials, profiles, filenames and URLs are not logged by this event.

RED evidence: processing endpoints returned404, both web DB faults emitted6cookies, and an unrelated member update stalled behind photo cleanup. These were captured before fixes.

Final verification is recorded in the coordinating review summary. The real HTTP/MariaDB/PNG test creates two actual uploads, commits a member, calls the normal ConsumeWithPhoto handler transition, checks the selected member photo and row survive, and waits for the unused disk file plus WEO_FILES row to disappear. The managed-claim race checks an official claim observes committed metadata removal before disk unlink completes, returns ErrManagedUploadUnavailable and writes neither ownership nor a dangling member photo. Alias tests cover encoded filenames, absolute aliases, base64 content, related malformed paths and unrelated malformed paths. A concurrent unrelated member write completes while unlink is paused.

### Atomic ownership boundary

Fresh upload filenames contain 96 random bits (24 hex characters) and are immutable. Cleanup accepts an actual tracked UploadResult and verifies its exact FSeq, gate, join status and URL. Same-token processing/commit serialization plus the exact-file claim lock protects official lifecycle writers. All existing persistent references are checked at cleanup decision time. Legacy content/direct copied-URL writes that do not participate in this managed claim protocol can introduce a new reference concurrently or after deletion; this remains an explicit unmanaged/dangling-reference boundary. This lane does not claim atomic protection for every arbitrary legacy writer and does not alter unrelated content-write protocols or add a schema migration.

Follow-up GREEN suites (reported separately; overlapping cases are not summed):

- Full normal Go suite: 1,562 test/subtest passes, 85 opt-in skips, 0 failures.
- Focused remediation race: 46 passes, 2 DB opt-in skips, 0 failures; those database cases run in the next suite.
- Final-source repository/file/erasure MariaDB gate suite: 37 passes, 0 skips/failures. This re-runs the existing file-safety, hostless, banner/entity regressions, automatic erasure, ownership/nonblocking/managed-claim/encoded-reference checks and candidate unit test.
- Real HTTP recovery/cancel/photo-commit MariaDB suite: 11 passes, 0 skips/failures.

Integration baseline imports temporarily used the same 120-second local ARM harness workaround. The original 15-second harness file was restored; no test-support deadline or schema/config modification is included in the follow-up commit. Exact suite names and top-level cases are in the root `backend-review-verification-summary.json` artifact.


## Second-review backend follow-up

Opus round 2 returned PASS and identified P3-A. Metadata retirement now uses the database default REPEATABLE READ, with the exact WEO_FILES locking read before the first consistent reference read. This avoids READ COMMITTED DML compatibility issues with legacy statement binlogging while retaining visibility of managed claims that committed before the lock was acquired.

Cleanup deletes the exact verified metadata row and commits before filesystem unlink. DELETE failure, commit failure or an indeterminate commit outcome never authorize unlink; the intact file may remain an orphan, which is safer than restoring a row after its file disappeared. An official claim arriving after retirement receives ErrManagedUploadUnavailable and rolls back even while disk unlink is paused. Existing canonical owner/history/content/URL aliases and unrelated-write behavior remain unchanged.

A receipt with a private repository-owned path is minted only after successful metadata commit. The orchestrator retains it under the exact tracked FSeq/URL solely for disk retries: at most four attempts and five minutes, with abandoned receipts removed by a timer. A missing database row never grants arbitrary unlink permission. Exhaustion, expiry or process restart can leave an orphan; no schema or durable cleanup ledger was introduced.

RED tests reproduced filesystem deletion before DELETE and commit faults, and the inability to retry a real disk failure after retiring metadata. GREEN adds zero-unlink assertions for both database fault stages, actual MariaDB trigger failure, late official claim rejection before disk erase, real filesystem retry through the committed receipt, bounded exhaustion and missing-row authorization denial. Final suite counts and exact names are recorded in the coordinating backend-round3-verification-summary.json artifact.


Final-source round 3 GREEN suites (overlap is not summed):

- Full normal Go suite: 1,568 passes, 86 opt-in skips, zero failures.
- Focused remediation race: 52 passes, 5 database opt-in skips, zero failures; the database tests pass in the gated suites.
- Repository/file/erasure MariaDB gate: 41 passes, zero skips/failures, including both DELETE/commit unit fault stages and all existing gated reference groups.
- Real HTTP recovery/cancel/photo-commit MariaDB suite: 11 passes, zero skips/failures.

These final integration runs used the unchanged 15-second harness. No temporary harness workaround remains in source.
