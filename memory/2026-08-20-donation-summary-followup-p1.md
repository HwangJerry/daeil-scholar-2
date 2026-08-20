# DEBUG REPORT: donation summary follow-up P1 fixes

- **Symptom:** A stale snapshot cache entry could bypass live fallback for five minutes, and committed admin ledger writes skipped summary recovery when the response read failed.
- **Root cause:** `GetSummary` read the cache before its atomic stale flag. Admin create/update methods also combined the committed write with the response-only `GetDonationOrder` query, hiding the commit boundary from the orchestrator.
- **Fix:** Check stale state before cache access. Return the committed write result first, mark the snapshot stale, run a best-effort refresh, and only then query the response record.
- **Evidence:** The new stale-cache and create/update response-read failure tests failed on the original ordering and pass after the changes.
- **Regression tests:** `TestDonationSummaryIgnoresCacheWhenSnapshotIsStale`, `TestDonationOrderCreateResponseReadFailureStillRefreshesSummary`, and `TestDonationOrderUpdateResponseReadFailureStillRefreshesSummary`.
- **Related:** Four intentionally deferred limitations are documented inline: refresh ordering, second-precision update tokens, donor-count refresh after identity unlinking, and admin query invalidation.
- **Status:** DONE
