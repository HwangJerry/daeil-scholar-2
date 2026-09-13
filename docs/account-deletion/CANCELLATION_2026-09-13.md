# Account deletion cancellation — 2026-09-13

## Implemented contract

- POST `/api/account-deletion/cancel` accepts separate `receiptToken` and `cancelToken`, each 32 random bytes encoded as 64 hex characters. No login is required; only hashes are retained by the server. Receipt lookup never requires the cancellation secret.
- Updated apps persist both secrets before submitting. Android uses its encrypted receipt store; iOS uses Keychain and passes capabilities in the fragment of the existing receipt web page. The page immediately removes the fragment from browser history and retains secrets only in memory.
- Receipt provides `canCancel` and `cancelledAt`. Cancel confirmation is required. Response loss triggers a lookup; unconfirmed cancellation is not reported as success.
- Pending requests can be cancelled until execution starts, including after scheduling/expediting if the worker has not started. Cancel and execution share the erasure lock. Start records `STARTED_AT` before social outbox enqueue in the same transaction. Cancellation rejects processing, started, or already-progressed target state.
- Cancellation restores the captured member state (CCC/BBB/BAA). Former ZZZ root administrators become CCC. Sessions and administrator roles remain revoked; fresh login is required. Any intervening member update while disabled blocks automatic restoration.
- `cancelled` is terminal. Worker selection, execution, schedule/mode changes and evidence mutations exclude terminal requests. Reapplication creates a new request and schedule. The cancelled request releases its user association and remains readable for 30 days; public completion date is omitted in favor of cancellation date.
- Administrators can invoke `cancel_verified` only after entering identity verification evidence. It uses the same cancellation guards; evidence is recorded before attempting cancellation. A denied attempt can therefore retain its verification evidence.

Cancellation preserves receipt-work evidence as a historical snapshot. It does not cancel independent donation receipt issuance/correction; operators must continue that work and contact cleanup through the existing accounting workflow, as explicitly stated in the admin UI. Cancelled user receipts do not claim erasure work remains pending.

## Compatibility and rollout

Apply migration 067 after 066 before deploying the server. Candidate lineage/contract fixtures include 067; no live migration approval was changed. Deploy backend and receipt web page before mobile updates. Cancellation endpoint intentionally remains available when new deletion intake is paused.

Preexisting requests without a captured original state or cancellation secret cannot be automatically restored. Old clients remain receipt-only. Lost local secrets require administrator identity verification for requests that have the new server metadata; no OTP/self-service recovery has been added. Existing social unlink operations cannot be undone by cancellation.

No production migration, deletion activation, real account cancellation, app upload or server deployment was performed as part of this change. Previous production pause remains in effect until a separately verified rollout.

## Verification

- Backend `go test ./...`: passed, including migration lineage contract.
- Docker MariaDB integration: cancellation, scheduled erasure and automatic erasure suites passed. Covers wrong secret, duplicate request/cancellation, original member states, revoked roles, reapplication, expedited-but-not-started cancellation, post-start denial, intervening member changes and erasure lock contention.
- Android debug build and account-deletion repository tests passed. Added independent durable secret, successful cancellation, response-loss recovery and failed cancellation cases.
- Web cancellation tests passed: confirmation, fragment removal, separate capabilities, cancelled terminal UI and refreshing execution status after failure.
- Admin schedule/cancellation tests passed, including mandatory verification evidence.
- iOS generic simulator build passed. Receipt-store XCTest tests passed on iOS Simulator (6 tests); final secret-write interruption guard is verified in the final test run. The first simulator bootstrap attempt failed before test execution; retry succeeded.

Device visual QA and production end-to-end cancellation are not claimed by these automated checks.
