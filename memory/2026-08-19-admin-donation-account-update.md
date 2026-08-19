# DEBUG REPORT: admin donation account reassignment

- **Symptom:** `PUT /api/admin/donation/orders/{orderSeq}` accepted `accountUsrSeq` and returned success without changing `WEO_ORDER.O_ACCOUNT_USR_SEQ`.
- **Root cause:** The update service omitted CreateOrder's account existence validation, the repository UPDATE omitted `O_ACCOUNT_USR_SEQ`, and the request's `*int` representation collapsed omitted and explicit JSON `null` into the same value.
- **Fix:** Reused a shared service validator for create and update, introduced the handler's existing `value + set + UnmarshalJSON` nullable-field pattern for `accountUsrSeq`, and conditionally updates the account column with `CASE WHEN ? THEN ? ELSE O_ACCOUNT_USR_SEQ END`.
- **Evidence:** Handler tests prove omitted/null/value decode separately. Repository SQL mock tests prove value changes the column, omission preserves it, and explicit null clears it. A service test proves an unknown account returns `ErrDonationAccountNotFound` before persistence.
- **Regression tests:** `backend/internal/handler/admin_donation_ledger_test.go`, `backend/internal/service/admin_donation_service_test.go`, and `backend/internal/repository/admin_donation_ledger_test.go`.
- **Related:** Commit `8c8471c` added account linkage on create but did not carry the field through the full update path.
- **Verification:** `go build ./... && go test ./...` passed from `backend/` on 2026-08-19.
- **Status:** DONE
