# Retained review accounts hidden from directory — 2026-11-13

The explicitly provisioned accounts `review261112` (app review, member 3) and `signupqa261112` (signup QA, member 4) remain in production for future releases. No member, password, approval, authentication identity or role records were modified. The review account remains approved; QA remains pending.

The shared alumni eligibility SQL excludes those exact IDs alongside canonical administrator accounts. It applies to search rows/counts, detail lookup, member-derived filter values and widget names/counts. It also excludes the QA identity if approved later. Names and ID prefixes are not used for classification, so ordinary members with similar names/IDs are unaffected. Future review identities must be added explicitly to this policy after identification.

Validation:

- `go test ./...` passed.
- Read-only synthetic approved-member fixtures executed against MariaDB using the exact repository SQL: review and QA identities excluded, ordinary identity and `review261112-real` retained. No fixture records inserted into production.
- Built clean commit `70ebc4e`; server SHA-256 `db4f464b8e01fb27e6fd592365009496fbd22f1a1354eddfaa98fcc6b0af9168` installed.
- Previous binary retained privately at `/app/maintenance/hide-review-20261113-70ebc4e/server.before`; loopback health and external HTTPS verified; Apache gracefully reloaded to refresh proxy connections.
- Real review login with iOS and Android headers succeeded; authenticated self and approved-member directory/filter APIs and feed remained accessible. Both test identities and administrator return 404 from directory detail.
- With only operator/review/QA/pending members currently present, directory and widget correctly return zero. No public member was removed or approval changed.
- Hall of Fame remains 155 public entries. Successful verification sessions logged out.
- Database verification confirmed both review/QA records still exist with nonempty password hashes and their original approval statuses.

This is a backend-only change; existing installed apps receive it without a store release.
