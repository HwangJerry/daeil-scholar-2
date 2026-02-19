# Backend Review — Cycle 3
## Reviewer: reviewer-backend
## Date: 2026-02-20
## Commit: e39488c

## Task #3: auth.go Legacy Session Status Fix

### Review Result: APPROVED

### Checklist
- [x] Status codes correct (AAA removed, BBB/CCC/ZZZ present)
- [x] Consistent with auth_repo.go IN ('BBB','CCC','ZZZ') pattern
- [x] Only auth.go changed (no scope creep)
- [x] No regression introduced
- [x] Matches IC-1 specification

### Findings

No issues — all checks passed.

**1. Status codes verified against authoritative source (`admin_auth.go` header, lines 1–13):**
- `AAA` = 탈퇴회원 (withdrawn) — correctly **removed** from legacy session check
- `BBB` = 승인대기 (pending) — present ✓
- `CCC` = 승인회원 (approved) — correctly **added** ✓
- `ZZZ` = 운영자 (admin) — correctly **added** ✓

**2. Consistency with `auth_repo.go`:**
All three member-lookup queries filter with `USR_STATUS IN ('BBB','CCC','ZZZ')`:
- `FindMemberByNamePhone` (line 61)
- `FindMemberByFNName` (line 78)
- `FindMemberByPhone` (line 138)

The fix at `auth.go:54` now matches this exact pattern: `BBB || CCC || ZZZ`.

**3. Scope — single file, single line:**
```
backend/internal/middleware/auth.go | 2 +-
1 file changed, 1 insertion(+), 1 deletion(-)
```
No other files modified. No scope creep.

**4. Regression analysis:**
- `USRStatus == "AAA"` (withdrawn): previously authenticated via legacy session → now correctly returns 401. **Security fix.**
- `USRStatus == "CCC"` (approved): previously blocked → now authenticates. **Bug fix.**
- `USRStatus == "ZZZ"` (admin): previously blocked → now authenticates. **Bug fix.**
- `USRStatus == "BBB"` (pending): unchanged, still authenticates. **No regression.**

**5. IC-1 compliance:**
The committed code exactly matches the "Required Fix" block in `IMPLEMENTATION_CHECKLIST.md` IC-1:
```go
if legacyUser.USRStatus == "BBB" || legacyUser.USRStatus == "CCC" || legacyUser.USRStatus == "ZZZ" {
    return legacyUser, nil
}
```

### Verdict

**APPROVED.** Commit `e39488c` correctly fixes the CRITICAL security bug where withdrawn members (AAA) could authenticate via legacy session fallback, while approved members (CCC) and admins (ZZZ) were incorrectly blocked. The fix is minimal (1 line), consistent with all repository queries, and matches the IC-1 specification exactly. No regressions introduced.

---

## Task #6: Payment Transaction + Idempotency Fix (commit 7cbb3cf)

### Reviewer: reviewer-payment
### Date: 2026-02-20

### Review Result: APPROVED

### Checklist
- [x] Idempotency check runs BEFORE ep_cli.Approve()
- [x] Transaction: Beginx → defer Rollback → InsertPGDataTx → UpdateOrderPaymentTx → Commit
- [x] PGAuditLogger calls are OUTSIDE the transaction
- [x] Only donate_service.go and donate_repo.go changed
- [x] New Tx methods use *sqlx.Tx parameter correctly
- [x] Original non-Tx methods preserved (backward compat)
- [x] Matches IC-2 specification

### Findings

No blocking issues — all checks passed.

**1. Idempotency check order verified (`donate_service.go:115-123`):**
The `GetOrderPrice()` call and `paymentStatus == "Y"` short-circuit now run at lines 116–123, **before** `epSvc.Approve()` at line 125. Previously, this check was after `Approve()` (the diff confirms it was moved from lines 132–138 in the old code to lines 116–123). This prevents duplicate PG charges for already-paid orders.

**2. Transaction pattern is correct (`donate_service.go:163-190`):**
```
Line 164: tx, err := s.repo.DB.Beginx()
Line 170: defer tx.Rollback()
Line 172: pgSeq, err := s.repo.InsertPGDataTx(tx, pgData)
Line 179: affected, err := s.repo.UpdateOrderPaymentTx(tx, ...)
Line 186: if err := tx.Commit(); err != nil { ... }
```
- `Beginx()` uses sqlx's transaction starter (correct for `*sqlx.DB`).
- `defer tx.Rollback()` immediately follows — safe no-op after successful Commit per `database/sql` spec.
- Both DB operations use the same `tx`.
- `Commit()` at the end with error handling.
- No nested transactions, no partial commits.

**3. PGAuditLogger placement:**
- `approve_success` log: line 143 — **before** `tx` starts. ✓
- `db_tx_begin_fail` log: line 167 — `tx` failed to start, not inside tx. ✓
- `db_insert_fail` log: line 175 — within tx code block, but PGAuditLogger writes to a **file** (not DB), so it's functionally independent of the DB transaction. The function returns an error here, triggering `defer tx.Rollback()`. Audit record persists correctly. ✓
- `db_update_fail` log: line 182 — same reasoning as above. ✓
- `db_tx_commit_fail` log: line 188 — after `Commit()` failed. ✓

All audit writes are to `PG_AUDIT_LOG_PATH` (a file), never via the DB transaction. They correctly persist regardless of transaction outcome.

**4. Scope confirmed — exactly 2 files changed:**
```
backend/internal/repository/donate_repo.go | 26 ++++++++++++++++++++
backend/internal/service/donate_service.go | 39 ++++++++++++++++++++++--------
2 files changed, 55 insertions(+), 10 deletions(-)
```

**5. Tx method signatures verified (`donate_repo.go`):**
- `InsertPGDataTx(tx *sqlx.Tx, data *model.PGData)` — line 51: uses `tx.Exec(...)` ✓
- `UpdateOrderPaymentTx(tx *sqlx.Tx, orderSeq int, amount int, pgSeq int64, ip string)` — line 76: uses `tx.Exec(...)` ✓

Both methods use the `tx` parameter for queries, not `r.DB`. SQL statements are identical to their non-Tx counterparts.

**6. Backward compatibility — original methods preserved:**
- `InsertPGData` (lines 39–48) — unchanged, still uses `r.DB.Exec`. ✓
- `UpdateOrderPayment` (lines 62–73) — unchanged, still uses `r.DB.Exec`. ✓

**7. IC-2 compliance:**
The implementation matches all three requirements from `IMPLEMENTATION_CHECKLIST.md` IC-2:
- ✅ DB transaction wrapping `InsertPGData` + `UpdateOrderPayment`
- ✅ Idempotency check moved before `ep_cli.Approve()`
- ✅ Flow: `GetOrderPrice() → already-paid check → ep_cli Approve() → InsertPGData + UpdateOrderPayment (in tx)`

### Observation (non-blocking)

There is a TOCTOU race window between the idempotency check (line 116, reads `O_PAYMENT`) and the `UpdateOrderPaymentTx` (line 179, `WHERE O_PAYMENT = 'N'`). If two PG callbacks arrive concurrently for the same order, both could pass the initial check. The secondary guard (`WHERE O_PAYMENT = 'N'` + `affected == 0` at line 192) mitigates this, but the second callback would still call `ep_cli.Approve()` and insert a duplicate `WEO_PG_DATA` row. A `SELECT ... FOR UPDATE` lock on the order row would fully close this gap, but that's a separate enhancement — not a regression from this commit.

### Verdict

**APPROVED.** Commit `7cbb3cf` correctly implements both IC-2 fixes: (1) the idempotency check now prevents unnecessary PG gateway calls for already-paid orders, and (2) `InsertPGData` + `UpdateOrderPayment` are wrapped in a proper sqlx transaction with `Beginx/defer Rollback/Commit`, preventing inconsistent state where a customer is charged but the order remains unpaid. Original non-Tx methods are preserved for backward compatibility. PGAuditLogger correctly writes to file outside the DB transaction. The TOCTOU race is a pre-existing condition noted for future hardening.
