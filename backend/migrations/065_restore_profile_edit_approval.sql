-- Profile edits no longer require a second admin review.
-- Only the legacy reapproval state proves an earlier approval; initial
-- pending/unsubmitted/rejected applications must not be promoted.
-- Deploy the updated API before applying this idempotent repair.
-- Preserve existing academic information and approval snapshots; do not
-- manufacture reviewer identities or timestamps cleared by the old flow.
UPDATE ALUMNI_VERIFICATION
SET STATUS = 'approved', UPDATED_AT = NOW()
WHERE STATUS = 'reapproval_pending';
