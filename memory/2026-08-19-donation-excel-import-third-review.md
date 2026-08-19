# Donation Excel Import Third Review Debug Report

- **Symptom:** Production Happy Sharing headers were rejected; commit rows could be changed after preview; excessively large amounts could lose precision in the browser; preview and commit performed per-row database lookups and writes.
- **Root cause:** Header rules were based on synthetic labels rather than the two 67-sheet production workbooks. Commit trusted client-resubmitted identity fields without a preview-bound proof. Amount validation only checked positive `int64`. Both service phases called single-row repository methods inside loops.
- **Fix:** Accept department headers containing `과` and validate F by position/value; cap imports at 1 trillion won; add one-hour HMAC-SHA256 preview tokens bound to all row data, donation date, match status, and matched member; batch candidate and duplicate queries; lock unique active account IDs in ascending order; insert all orders with one multi-row statement while retaining the canonical normalization pipeline.
- **Evidence:** Both referenced workbooks have 67 sheets with zero B-E header mismatches under the new rules. Backend `go build ./...` and `go test ./...` pass. Admin `npm run build`, `npm run lint`, and `npm run test` pass.
- **Regression tests:** `backend/internal/service/donation_import_service_test.go`, `backend/internal/repository/donation_import_repo_test.go`.
- **Related:** The earlier hardening commit introduced the production-header regression and removed the manual flag without binding preview data to commit data.
- **Status:** DONE
