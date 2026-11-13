# Administrator directory exclusion — 2026-11-13

Changed the production administrator account (`jerryhwang018`, member 1) phone to canonical `01000000000`, displayed as `010-0000-0000`. Updated its existing `AUTH_PHONE_CLAIM` in the same transaction. Confirmed no other member used that phone. Login ID, administrator role and approval remain intact.

The alumni repository excludes canonical `ALUMNI_ADMIN_ROLE` accounts from search rows and counts, detail lookup, member-derived filter options and widget preview/counts. This preserves administrative access without listing operators as alumni. No client release or database schema change is required.

Validation:

- `go test ./...` passed; repository search expectations updated for the exclusion.
- Built Linux artifacts from clean commit `f0f271e` with release-bundle provenance verification.
- Installed server SHA-256 `08e886889f81d83d144a8eb32170dd0ac7a93aae1820970494e3b96215d85600`.
- Prior binary retained privately at `/app/maintenance/hide-admin-20261113-f0f271e/server.before`.
- Verified internal HTTP health and external HTTPS endpoints. Server-local public HTTPS checks are unsuitable on this host (routing/trust); initial attempt rolled back automatically, then deployment passed using loopback HTTP plus external HTTPS verification. Apache graceful reload cleared stale proxy connections after restart.
- Approved review account, with both iOS and Android request headers: search and widget each return the one nonadministrator approved member; filters and ordinary member detail return 200; administrator directory detail returns 404. Successful verification session logged out.
- Public Hall of Fame still returns 155 entries.

Only backend binary and the requested phone/claim were changed in production. Existing administrative authorization and public-content restoration remain intact.
