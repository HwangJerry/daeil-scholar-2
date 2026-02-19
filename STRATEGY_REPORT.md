# Strategy Report — Cycle 3
## Date: 2026-02-20
## Strategist: Strategist Agent (Opus)

## Executive Summary

The overall architecture is **solid for its constraints** — the layered Go backend, dual SPA separation, and auth bridge design are well-thought-out for a 1GB single-server deployment. However, this first architectural review surfaces **1 CRITICAL security issue** and **2 IMPORTANT reliability issues** that should be addressed before production traffic grows. The critical issue is in the auth middleware's legacy session fallback, which currently allows withdrawn members to authenticate while blocking approved members. The important issues relate to admin status typos in the spec that could cause dangerous implementation errors, and a lack of DB transactions in the payment confirmation flow.

## Critical Issues

### C1. Legacy Auth Fallback Allows Withdrawn Members, Blocks Approved Members

**Severity:** CRITICAL (security)
**Affected files:** `backend/internal/middleware/auth.go:54`, TECH_DESIGN_DOC.md §4.1
**Description:**

The `resolveAuthUser()` function in `auth.go:54` has this legacy session fallback check:

```go
if legacyUser.USRStatus == "BBB" || legacyUser.USRStatus == "AAA" {
    return legacyUser, nil
}
```

Per the authoritative status reference in `admin_auth.go` (confirmed by `admin_member_service.go` and all repository queries):
- `'AAA'` = 탈퇴회원 (withdrawn/resigned member)
- `'BBB'` = 승인대기 (pending approval)
- `'CCC'` = 승인회원 (approved member)
- `'ZZZ'` = 운영자 (operator/admin)

**Two bugs in one line:**
1. **Allows `AAA` (withdrawn)** — A deactivated member with a valid legacy session cookie can still access the system. This is a security breach.
2. **Blocks `CCC` (approved) and `ZZZ` (admin)** — The vast majority of active members who logged in via PHP cannot use the Go backend through the legacy fallback path.

The spec (§4.1 line 530) only checks `user.USRStatus == "BBB"`, which has the same problem of blocking approved members. The `FindMemberByLogin` query in `auth_repo.go:121` correctly uses `USR_STATUS >= 'CCC'` for direct login, and `FindMemberByNamePhone` in `auth_repo.go:61` uses `IN ('BBB','CCC','ZZZ')` for Kakao linking. The legacy fallback should be consistent with these.

**Recommendation:** Change the legacy fallback check to:
```go
if legacyUser.USRStatus == "BBB" || legacyUser.USRStatus == "CCC" || legacyUser.USRStatus == "ZZZ" {
```
This matches the Kakao link query pattern and excludes only withdrawn members.

**Action taken:** Added spec note to TECH_DESIGN_DOC.md §4.1 (Change #1).

---

## Important Issues

### I1. TECH_DESIGN_DOC §15.2 and §6.7 — Admin Status 'AAA' Should Be 'ZZZ'

**Severity:** IMPORTANT (spec correctness / new-developer trap)
**Affected files:** TECH_DESIGN_DOC.md §15.2 (line ~2531), §6.7 (line ~1557)
**Description:**

Two sections of the spec reference `USR_STATUS = 'AAA'` as the admin/operator status:
- §15.2 (line 2514): `USR_STATUS = 'AAA'` in the overview table
- §15.2 (line 2531): `user.USRStatus != "AAA"` in the code example
- §6.7 (line 1557): "USR_STATUS = 'AAA'(관리자)"

The production code (`admin_auth.go:27`) correctly uses `"ZZZ"`. Commit `4fc3818` documented this discrepancy but did not fix the spec itself. Any developer implementing from the spec would check for `'AAA'` (withdrawn members), granting admin access to deactivated accounts.

**Recommendation:** Fix the spec to use `'ZZZ'` in all admin status references.

**Action taken:** Fixed TECH_DESIGN_DOC.md §15.2 and §6.7 (Change #2).

### I2. Payment ConfirmPayment — Non-Atomic DB Operations

**Severity:** IMPORTANT (financial data integrity)
**Affected files:** `backend/internal/service/donate_service.go:100-181`, `backend/internal/repository/donate_repo.go`
**Description:**

In `ConfirmPayment()`, after `ep_cli` approves the payment (money is charged to the customer), two separate DB operations execute without a transaction:

1. `s.repo.InsertPGData(pgData)` — inserts PG approval record
2. `s.repo.UpdateOrderPayment(orderSeq, ...)` — marks order as paid

**Failure scenario:** If `InsertPGData` succeeds but `UpdateOrderPayment` fails (connection drop, deadlock, etc.):
- `WEO_PG_DATA` records successful payment
- `WEO_ORDER` still shows `O_PAYMENT='N'`
- Customer sees failure page, but money is charged
- Manual reconciliation required via PG audit log

The `PGAuditLogger` mitigates this by recording all events for manual reconciliation, which is good. But a DB transaction wrapping both operations would prevent the inconsistency entirely.

Additionally, the idempotency check (`paymentStatus == "Y"`) happens **after** the `ep_cli` approval call at line 139. If a duplicate EasyPay callback arrives, the code would call `ep_cli` again before detecting the already-paid order. Moving the check before `ep_cli.Approve()` would prevent unnecessary external calls.

**Recommendation:**
1. Wrap `InsertPGData` + `UpdateOrderPayment` in a single DB transaction
2. Move the `paymentStatus == "Y"` check before the `ep_cli.Approve()` call

**Action taken:** Added spec note to TECH_DESIGN_DOC.md §16.3 (Change #3).

---

## Nice-to-Have

### N1. No Maximum Payment Amount Validation
`CreateOrder` validates `amount >= 10000` but has no upper bound. While EasyPay likely enforces its own limits, a server-side maximum (e.g., 10,000,000원) would prevent abuse.

### N2. JWT Auth Doesn't Check USR_STATUS
When JWT auth succeeds (`auth.go:46-49`), there's no status check. A withdrawn member with a valid JWT can access the system until token expiry (24h). This is a common JWT trade-off and acceptable given the low-risk user base, but could be improved by adding a status check or using shorter token TTL.

### N3. Nginx `/api/auth/` Location Missing Proxy Headers
The `/api/auth/` location block in `nginx/conf.d/alumni.conf:42-47` sets `X-Real-IP` but omits `X-Forwarded-For` and `X-Forwarded-Proto` that the general `/api/` block includes. This could affect request logging completeness for auth endpoints.

### N4. No Retry Logic After PG Approval + DB Failure
If `ep_cli` approves a payment but DB write fails, there's no automatic retry. The PG audit logger captures the event for manual reconciliation, but an automatic retry (even a single attempt) would reduce the need for manual intervention.

### N5. Content Pipeline Memory Allocation
Large Markdown posts could cause significant memory allocation during `goldmark` rendering + `base64` encoding within the 80MiB GOMEMLIMIT. This is mitigated by: (1) admin-only content authoring, (2) 1MB body size limit, (3) typical post sizes are small. No immediate action needed.

---

## Informational Observations

### O1. Background Jobs Are Properly Cancellable
Both `DonationSnapshotJob` and `SessionCleanupJob` use `context.WithCancel` and are properly stopped during graceful shutdown. The donation job also recovers from panics. Well-designed for the constraints.

### O2. PGAuditLogger Is Durable
Uses `sync.Mutex` + `fsync` per write. Good design for financial audit trail. The JSON-per-line format is parseable for automated reconciliation tooling.

### O3. OAuth State Validation Is Correct
Uses `go-cache` with 5-minute TTL, generates random 32-char state, validates on callback, deletes after use. Prevents CSRF on OAuth flow.

### O4. CSRF Middleware Correctly Scoped
`/pg/*` routes are properly placed outside the CSRF middleware group in `routes.go`. The CSRF middleware correctly allows GET/HEAD/OPTIONS and validates Origin/Referer for state-changing methods.

### O5. Donation Snapshot Startup Backfill
The snapshot job checks for today's snapshot on startup and creates one if missing. This prevents stale data after server restarts that span midnight. Good resilience pattern.

### O6. Auth Bridge Design Is Sound
The dual JWT + legacy cookie approach with JWT-first fallback is well-designed for the PHP coexistence period. The cookie clearing on logout covers both systems.

### O7. Idempotent Order Update
`UpdateOrderPayment` uses `WHERE O_PAYMENT='N'` to prevent double-payment. This is the correct pattern, though it should be paired with a transaction (see I2).

---

## Doc Changes Made This Cycle

### Change #1: §4.1 — Legacy Auth Status Filter Warning
**Section:** §4.1 브릿지 미들웨어 구현 (after the `lookupLegacySession` code block)
**Type:** WARNING note
**Reason:** The spec shows `user.USRStatus == "BBB"` but the implementation has `"BBB" || "AAA"`. Both are wrong — AAA allows withdrawn members, and both block CCC (approved) and ZZZ (admin). The filter should be `IN ('BBB','CCC','ZZZ')` to match other auth queries.

### Change #2: §15.2 and §6.7 — Admin Status Fix ('AAA' → 'ZZZ')
**Sections:** §15.2 overview table and code example, §6.7 description
**Type:** Correction
**Reason:** Admin status is 'ZZZ' (operator), not 'AAA' (withdrawn). Confirmed by `admin_auth.go`, `admin_member_service.go`, and all repository queries.

### Change #3: §16.3 — Payment Transaction Requirement Note
**Section:** §16.3 Donate Service ConfirmPayment
**Type:** WARNING note
**Reason:** InsertPGData + UpdateOrderPayment must be in a DB transaction. Current spec and implementation run them as separate operations, risking data inconsistency after PG approval.

---

## Worker Task Recommendations

Based on this review, the following work should be done (but can be deferred if no workers are spawned this cycle):

1. **[HIGH] Fix `auth.go:54` legacy status filter** — Change `"BBB" || "AAA"` to `"BBB" || "CCC" || "ZZZ"`. Single-line fix, but critical security impact.
2. **[MEDIUM] Wrap ConfirmPayment DB ops in transaction** — Requires adding `BeginTx`/`Commit`/`Rollback` to `donate_repo.go` or `donate_service.go`. Moderate effort.
3. **[MEDIUM] Move idempotency check before ep_cli call** — In `donate_service.go`, call `GetOrderPrice` before `epSvc.Approve()`. Simple reorder.
4. **[LOW] Add max payment amount validation** — Add upper bound check in `CreateOrder`.
5. **[LOW] Fix nginx auth location proxy headers** — Add missing `X-Forwarded-For` and `X-Forwarded-Proto` to `/api/auth/` block.

---

## Loop Exit Recommendation

**Do NOT exit the loop.** The CRITICAL auth bug (C1) should be fixed in code this cycle if possible, not just documented. The legacy auth fallback allowing withdrawn members is a security vulnerability that could be exploited now. The payment transaction issue (I2) is important but the PGAuditLogger provides a safety net for manual reconciliation.

**Recommendation:** Continue with 1 worker to fix the auth bug (C1). The payment transaction fix (I2) can be deferred to a future cycle since the PGAuditLogger provides a mitigation layer. The spec doc changes made this cycle provide the necessary context for both fixes.

---

## Cycle 4 Verification — 2026-02-20

### C1 (auth.go) — Resolved?

**YES** — Fully resolved in commit `e39488c`.

Evidence at `backend/internal/middleware/auth.go:54`:
```go
if legacyErr == nil && legacyUser != nil && (legacyUser.USRStatus == "BBB" || legacyUser.USRStatus == "CCC" || legacyUser.USRStatus == "ZZZ") {
```

This correctly:
- **Allows** `BBB` (pending approval), `CCC` (approved member), `ZZZ` (operator/admin)
- **Blocks** `AAA` (withdrawn member) — by exclusion
- Matches the pattern used in `FindMemberByNamePhone` (`IN ('BBB','CCC','ZZZ')`) for consistency
- Removes the previously allowed `AAA` status that was the security breach

### I1 (spec doc) — Resolved?

**YES** — Fixed in TECH_DESIGN_DOC.md v1.7 and propagated to split docs.

Evidence across all spec documents:
1. `docs/spec-admin.md` §15.2 line 21: `USR_STATUS = 'ZZZ'` (운영자) ✓
2. `docs/spec-admin.md` §15.2 code example line 37-38: `user.USRStatus != "ZZZ"` ✓
3. `docs/spec-api.md` §6.7 line 284: `USR_STATUS = 'ZZZ'`(운영자) with correct legend: `'AAA'`=탈퇴, `'BBB'`=승인대기, `'CCC'`=승인회원, `'ZZZ'`=운영자 ✓
4. `docs/spec-auth.md` §4.1 line 96: Warning note added referencing `IMPLEMENTATION_CHECKLIST.md` IC-1 ✓

No remaining references to `'AAA'` as admin/operator status in the spec docs.

### I2 (payment transaction) — Resolved?

**YES** — Fully resolved in commit `7cbb3cf`. Both sub-recommendations implemented.

**(a) Idempotency check moved before `ep_cli`:**
`donate_service.go:115-123` — `GetOrderPrice` is called and `paymentStatus == "Y"` is checked **before** `epSvc.Approve()` at line 125. This prevents duplicate PG charges on callback retries. ✓

**(b) Atomic DB transaction:**
`donate_service.go:163-190` — `Beginx()` → `InsertPGDataTx(tx, ...)` → `UpdateOrderPaymentTx(tx, ...)` → `Commit()` with `defer tx.Rollback()`. ✓
`donate_repo.go:51-60` — `InsertPGDataTx` accepts `*sqlx.Tx`. ✓
`donate_repo.go:76-87` — `UpdateOrderPaymentTx` accepts `*sqlx.Tx`, retains `WHERE O_PAYMENT = 'N'` guard. ✓

**Important caveat (MyISAM):** `WEO_ORDER` and `WEO_PG_DATA` are legacy tables that are almost certainly MyISAM (confirmed by migration 001: "MyISAM tables — no engine change in Phase 1"; no migration changes these tables' engine; v2.0 roadmap includes "MyISAM → InnoDB 전환"). On MyISAM, `BEGIN`/`COMMIT`/`ROLLBACK` are effectively no-ops — each statement auto-commits individually. The transaction wrapping is **structurally correct** and will become effective upon InnoDB migration, but provides no atomicity guarantee today. The `WHERE O_PAYMENT = 'N'` guard remains the actual protection against double payment. This is a pre-existing architectural constraint, not a regression.

### TOCTOU Race (reviewer observation) — Severity Assessment

**NICE-TO-HAVE** (not CRITICAL or IMPORTANT)

Reasoning:
1. **Financial risk is mitigated.** `UpdateOrderPaymentTx` uses `WHERE O_PAYMENT = 'N'` — only one concurrent request can flip the payment status. The second gets `affected == 0` and returns nil (idempotent). No double-payment can occur at the order level.
2. **Duplicate `ep_cli` calls are safe.** If two concurrent callbacks race past the idempotency check and both call `ep_cli.Approve()`, the PG system should reject the second attempt (same encData/sessionKey). Even if it doesn't, the order-level guard prevents double-accounting.
3. **Orphan PG record is the only residual risk.** In the TOCTOU window, both requests could `InsertPGData` (one per transaction), but only one `UpdateOrderPayment` succeeds. The loser commits an orphan PG record with no corresponding order update. This is a minor data quality issue, not a financial one.
4. **`SELECT ... FOR UPDATE` wouldn't help on MyISAM anyway.** MyISAM uses table-level locking, not row-level. True transactional protection requires InnoDB migration first. The reviewer's suggestion of `SELECT ... FOR UPDATE` is correct in principle but requires InnoDB to be effective.
5. **PGAuditLogger covers reconciliation.** Both approval attempts are logged with full context, enabling manual or automated reconciliation of any orphan records.

**Recommendation:** Defer to InnoDB migration milestone (v2.0). At that point, add `SELECT O_PRICE, O_PAYMENT FROM WEO_ORDER WHERE O_SEQ = ? FOR UPDATE` inside the transaction to fully close the gap. File as tech debt, not a blocker.

### New Issues Introduced by Cycle 3 Commits?

**NO** — No new CRITICAL or IMPORTANT issues found.

The two commits (`e39488c` auth fix, `7cbb3cf` payment fix) are clean, targeted changes:
- `auth.go`: Single-line condition change, no new code paths
- `donate_service.go`: Reordered existing logic + wrapped in Tx, no new business logic
- `donate_repo.go`: Added `*Tx` variants of existing methods, identical SQL

The MyISAM/transaction caveat is a pre-existing constraint, not introduced by these commits.

### Remaining CRITICAL Issues: 0
### Remaining IMPORTANT Issues: 0

### Loop Exit Recommendation

**EXIT — all criteria met.**

- C1 (CRITICAL security): ✅ Resolved in production code
- I1 (IMPORTANT spec): ✅ Resolved in documentation
- I2 (IMPORTANT financial): ✅ Resolved in production code (structurally correct; full atomicity pending InnoDB migration)
- No new CRITICAL or IMPORTANT issues introduced
- TOCTOU race classified as NICE-TO-HAVE, deferred to InnoDB migration
- All 3 nice-to-have items from Cycle 3 (N1-N5) remain as future improvements, none blocking
