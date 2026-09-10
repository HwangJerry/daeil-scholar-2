# Profile edits and initial approval

Admin approval applies to initial membership only. Approved users save academic
changes immediately; profile writes preserve status, reviewer and historical
approved academic snapshots. Initial unsubmitted/rejected submissions still
become pending, and pending applicants remain pending.

iOS uses PUT /profile plus the alumni-verification update endpoint. Android
sends cohort/department to PUT /profile. That endpoint now atomically updates
WEO_MEMBER and the current verification fields for approved members. Omitting
academic fields preserves them. Neither path grants initial approval.

Deploy the updated API, then apply migration 065 to restore members left in
reapproval_pending by the prior policy. The migration is idempotent, does not
promote initial pending/rejected/unsubmitted members, and does not fabricate
review metadata that the old implementation cleared.

Validation: model, repository, service and middleware tests cover approved edits,
legacy approved records without a graduation year, legacy reapproval recovery,
initial submission and rejected resubmission. An isolated MySQL 8 smoke check
with migration 038 triggers verified the exact profile UPDATE, preserved reviewer
metadata, pending/rejected isolation and applying 065 twice.

Deployment caveat: the existing canonical migration source approval packet has
a checksum mismatch for 058 and does not include 064. Migration 065 also needs
to be included in the next reviewed packet. The three existing source-approval
contract checks fail on the 058 mismatch. No deployment approval hash was
rewritten, and no production database was modified by this task.
