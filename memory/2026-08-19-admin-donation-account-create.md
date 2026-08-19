# Admin donation account linkage on create

## Debug report

- **Symptom:** Admin-created donations with `accountUsrSeq` populated `O_ACCOUNT_USR_SEQ` but persisted legacy `USR_SEQ` as `0`, so legacy donor aggregation and member joins could miss or misclassify the donation.
- **Root cause:** `CreateDonationOrder` hardcoded the first INSERT argument (`USR_SEQ`) to `0`, unlike `UpdateDonationOrder`, which synchronizes `USR_SEQ` with `O_ACCOUNT_USR_SEQ` and falls back to `0` when the link is cleared.
- **Fix:** Initialize `USR_SEQ` from `accountUsrSeq` when present and use `0` when absent, while continuing to bind `O_ACCOUNT_USR_SEQ` directly.
- **Evidence:** The linked-account regression test failed before the fix with `expected 42, actual 0` and passed after the fix. `go build ./... && go test ./...` passed from `backend/`.
- **Regression test:** `backend/internal/repository/admin_donation_ledger_test.go` verifies both linked (`42`, `42`) and unlinked (`0`, `NULL`) INSERT arguments.
- **Related:** `DonationRepository.CountDonors` counts distinct `USR_SEQ`; `AdminDonationRepository.GetDonationOrders` joins `WEO_ORDER.USR_SEQ` to `WEO_MEMBER.USR_SEQ`. The synchronized create behavior now matches both legacy consumers and the existing update policy from `c9a7817`.
- **Status:** DONE
