# MVP migration rollback runbook (`030`–`034`)

## Safety gates

1. Stop writes or put the API into maintenance mode, and stop the message purge worker.
2. Take and verify a full database **backup** before any forward or rollback DDL.
3. Record row counts and schema metadata from `information_schema` without exporting personal values.
4. Roll back application binaries before removing columns they reference.
5. Execute rollback in reverse order: `034` → `033` → `032` → `031` → `030`.
6. Use `SHOW TABLE STATUS` and filesystem monitoring to confirm table size, engine,
   free disk space, and acceptable copy/lock time on a production-like clone.

MariaDB 10.1 DDL causes implicit commits. The steps below are an operator runbook,
not one transactional script. Before every `DROP`, confirm presence in
`information_schema.COLUMNS`, `information_schema.STATISTICS`, or
`information_schema.TABLES`.

## Forward preflight and expected failures

- Run numbered migrations exactly once. A direct rerun may fail with `1060`
  (duplicate column) or `1061` (duplicate index); that is expected and must not be
  bypassed with `--force`.
- `033` prints legacy donation error counts and intentionally fails with `1062`
  before persistent DDL when it finds payment/status disagreement, negative values,
  completed `O_PRICE <> O_PAY`, missing dates, or ambiguous legacy `O_STATUS='N'`.
  Reconcile those rows against approved source records and re-run the preflight;
  never coerce them into `completed` or silently clamp amounts.
- `apply_all.sql` records the one-time `021` privacy backfill in
  `ALUMNI_SCHEMA_MIGRATION`; a rerun changes column defaults but does not overwrite
  a member's subsequent privacy choice. Existing index names are accepted only when
  their ordered columns and unique property match the expected definition.

## `034` personal-data retention

Before removal, confirm there is no pending legal-retention purge and preserve any
required legal transaction records in the approved backup location.

- Drop `IDX_WO_ACCOUNT_DATE` and `IDX_WO_LEGAL_RETENTION` from `WEO_ORDER`.
- Drop `O_DONOR_NAME`, `O_DONOR_PHONE`, `O_DONOR_COHORT`,
  `O_DONOR_DEPARTMENT`, `O_LEGAL_RETENTION_UNTIL`, and
  `O_ACCOUNT_UNLINKED_AT`, and `O_ACCOUNT_USR_SEQ` from `WEO_ORDER`.
- Drop `IDX_WM_PERSONAL_PURGE`, `USR_ANONYMIZED_AT`, and `USR_PURGE_AT` from
  `WEO_MEMBER`.
- Drop `IDX_AM_SENDER_ACCOUNT` and `IDX_AM_RECVR_ACCOUNT`, then drop
  `AM_SENDER_ACCOUNT_SEQ`, `AM_RECVR_ACCOUNT_SEQ`,
  `AM_SENDER_ANONYMIZED_YN`, and `AM_RECVR_ANONYMIZED_YN` from `ALUMNI_MESSAGE`.

These columns may contain state that cannot be reconstructed after account
anonymization. Prefer restoring the backup over destructive rollback.

The canonical account links are `O_ACCOUNT_USR_SEQ`,
`AM_SENDER_ACCOUNT_SEQ`, and `AM_RECVR_ACCOUNT_SEQ`. Account withdrawal clears
the appropriate canonical link and sets the corresponding anonymized/unlinked
timestamp or flag. During the legacy-reader transition, the same operation also
sets the old `USR_SEQ`, `AM_SENDER_SEQ`, or `AM_RECVR_SEQ` link to sentinel `0`;
the new readers must use `LEFT JOIN` and the canonical link columns. Never rerun a
backfill that relinks rows marked anonymized or unlinked.

## `033` donation ledger

- Drop indexes `IDX_WO_LIFECYCLE_DATE`, `UK_WO_COMPOSITE_KEY`, and
  `UK_WO_SOURCE_TRANSACTION` from `WEO_ORDER`.
- Drop `O_SOURCE`, `O_TRANSACTION_NO`, `O_COMPOSITE_KEY`, `O_DONATION_DATE`,
  `O_GROSS_AMOUNT`, `O_REFUNDED_AMOUNT`, `O_NET_RECEIVED_AMOUNT`,
  `O_LIFECYCLE_STATUS`, and `O_PAYMENT_METHOD`.
- Re-enable the legacy aggregate only after reconciling canonical net amounts
  against `O_PRICE`, `O_STATUS`, and `O_PAYMENT`.

The canonical status/amount backfill is not reversibly encoded in legacy fields.
Use the backup for exact restoration.

## `032` push

- If the pre-migration schema contained a pre-existing `ALUMNI_PUSH_PREFERENCE`,
  preserve the table and its existing notice/message columns and indexes; drop only
  the added `MESSAGE_PREVIEW_ENABLED` column after the rolled-back application no
  longer references it.
- Drop `ALUMNI_PUSH_PREFERENCE` only when the pre-migration schema report proves
  migration 032 created the whole table.
- Drop `ALUMNI_PUSH_DEVICE`.

Revoke or invalidate provider tokens before deleting the device table if the
rollback is part of an incident response.

## `031` block and message retention

- Drop indexes `IDX_AM_PURGE`, `IDX_AM_RECVR_VISIBLE`, and
  `UK_AM_SENDER_CLIENT` from `ALUMNI_MESSAGE`.
- Drop `AM_CLIENT_MESSAGE_ID`, `AM_VISIBLE_RECVR`, `AM_SUPPRESSION_REASON`, and
  `PURGE_AT` from `ALUMNI_MESSAGE`.
- Drop `ALUMNI_MEMBER_BLOCK`.

Do not roll back while suppressed messages remain unless product and privacy
owners accept that legacy readers cannot enforce the recipient-hidden rule.
Until every reader filters `AM_VISIBLE_RECVR`, blocked-message writes must also set
legacy `AM_DEL_RECVR='Y'`; otherwise inbox, unread, and conversation queries can
expose a suppressed message.

## `030` roles and alumni verification

- Restore the legacy administrator status only for accounts confirmed from the
  pre-migration backup; do not infer it from current role rows after arbitrary
  role changes.
- Drop `ALUMNI_VERIFICATION` only after the application has returned to legacy
  `USR_STATUS` authorization.
- Drop `ALUMNI_ADMIN_ROLE` last.

Legacy `BAA` rows are migrated as `rejected` with a NULL rejection reason because
there is no reliable legacy reason source. Existing `ZZZ` administrators are
initially `root` to preserve current access; a root must explicitly review and
downgrade non-root operators before enabling role-management UI.

## Verification

After each stage:

- query `information_schema` to verify intended indexes, columns, and tables;
- run legacy member, message, and donation read smoke tests;
- verify no public donor endpoint or schema was introduced;
- compare row counts and aggregate donation totals with the pre-migration report.
