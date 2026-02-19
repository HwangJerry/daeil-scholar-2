# Implementation Checklist — Security & Validation

## IC-1. Legacy Auth Session Fallback — USR_STATUS Filter (CRITICAL)

**Source:** Cycle 3 Strategy Review
**Affected file:** `backend/internal/middleware/auth.go:54`
**Spec reference:** TECH_DESIGN_DOC.md §4.1

### Status Code Reference (Authoritative)

| Code | Meaning | Auth Allowed? |
|------|---------|---------------|
| `AAA` | 탈퇴회원 (withdrawn) | **NO** |
| `BBB` | 승인대기 (pending approval) | YES |
| `CCC` | 승인회원 (approved member) | YES |
| `ZZZ` | 운영자 (operator/admin) | YES |

### Current Code (INCORRECT)

```go
// auth.go:54 — legacy session fallback
if legacyUser.USRStatus == "BBB" || legacyUser.USRStatus == "AAA" {
    return legacyUser, nil
}
```

**Bug 1:** Allows `AAA` (withdrawn members) — security vulnerability.
**Bug 2:** Blocks `CCC` (approved) and `ZZZ` (admin) — breaks legacy auth for active users.

### Required Fix

```go
if legacyUser.USRStatus == "BBB" || legacyUser.USRStatus == "CCC" || legacyUser.USRStatus == "ZZZ" {
    return legacyUser, nil
}
```

This matches the pattern used in `auth_repo.go` for `FindMemberByNamePhone` and `FindMemberByPhone`:
```sql
WHERE USR_STATUS IN ('BBB','CCC','ZZZ')
```

### Test Cases

1. Legacy session with `USR_STATUS='AAA'` (withdrawn) → **must return 401**
2. Legacy session with `USR_STATUS='BBB'` (pending) → must authenticate
3. Legacy session with `USR_STATUS='CCC'` (approved) → must authenticate
4. Legacy session with `USR_STATUS='ZZZ'` (admin) → must authenticate
5. Expired legacy session (LOG_DATE > 24h) → must return 401
6. Invalid session ID → must return 401

---

## IC-2. Payment ConfirmPayment — DB Transaction Requirement (IMPORTANT)

**Source:** Cycle 3 Strategy Review
**Affected file:** `backend/internal/service/donate_service.go:100-181`
**Spec reference:** TECH_DESIGN_DOC.md §16.3

### Current Flow (Non-Atomic)

```
ep_cli Approve() → InsertPGData() → UpdateOrderPayment()
                    ↑ separate op     ↑ separate op
```

If `InsertPGData` succeeds but `UpdateOrderPayment` fails, the DB has inconsistent state: PG record exists but order remains unpaid. Customer is charged but sees failure.

### Required Fix

Wrap `InsertPGData` + `UpdateOrderPayment` in a single DB transaction:

```go
tx, err := s.repo.DB.BeginTx(ctx, nil)
// InsertPGData within tx
// UpdateOrderPayment within tx
// tx.Commit() or tx.Rollback()
```

### Additional Fix — Idempotency Check Order

Move the `paymentStatus == "Y"` check **before** `ep_cli.Approve()` to avoid calling the external PG service for already-paid orders:

```
GetOrderPrice() → check if already paid → ep_cli Approve() → InsertPGData + UpdateOrderPayment (in tx)
```

### Test Cases

1. Normal payment flow → PG data inserted, order marked paid, both in same transaction
2. DB failure after InsertPGData → transaction rolls back, no orphaned PG record
3. Duplicate callback (order already paid) → returns success without calling ep_cli
4. Amount mismatch → rejected before DB write
