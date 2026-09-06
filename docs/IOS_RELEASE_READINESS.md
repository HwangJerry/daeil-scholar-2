# iOS release fixes — September 6, 2026

These changes are prepared locally. They are not a production deployment or App Store submission.

## Implemented

- Public `/register` and `/register/complete` routes restore the app's email signup. Both bypass the temporary web maintenance gate without unlocking other routes. The signup page initializes its own auth state. Completion tells users to return to the native app after operator approval.
- `POST /api/message-reports` accepts a message ID, reason, and optional details. Only approved, authenticated recipients can report visible, undeleted incoming messages. Evidence is selected in SQL, never supplied by the caller. Repeated reports retain the original record.
- `/admin/message-reports` provides an authenticated administrator queue, pagination, processing history, mandatory decision notes, and content removal. Original evidence remains restricted to the moderator queue. Existing member management handles repeat offenders.
- The message service filters an initial set of abusive phrases before accepting new messages. NFKC normalization and removal of punctuation/spacing catch simple obfuscation. `MESSAGE_BLOCKED_PHRASES` adds comma-separated phrases without disabling the baseline. This is a limited rule-based filter, not a complete classifier.
- The Swift changes add reporting from an incoming message's context menu and the conversation menu, plus a public support link. Donation checkout opens in the system browser. The app owns a required-reason manifest and validates the packaged Release configuration.
- The web `/privacy` page follows the existing editorial layout, with an eight-section contents navigation, confirmed operator/contact details, and accurate withdrawal-retention disclosure. Both footers link to it. `/privacy` and `/privacy/` bypass the maintenance gate without unlocking other routes. This source change is not deployed and does not establish policy compliance; unknown external-service/backup details are explicitly identified.

## Deployment order

1. Candidate source manifest SHA-256: `8232f1eff1f4287a36b5fd5823eaa3a31b5f339fc0ca523f953de1f25e4219b1`. The source manifest, regression pin, and environment example include migrations 055 and 056. Production approval/environment settings have not been changed. Review migration numbering against any concurrently developed backend changes; this branch adds `055_create_message_reports.sql` and `056_create_account_deletion_requests.sql`.
2. Apply those additive migrations using the project's normal migration procedure. No production migration has been executed by this task.
3. Deploy the backend and both SPAs together. Confirm signup works with an empty browser session and that `/api/message-reports` exists before distributing the new iOS build.
4. Give a designated moderator an existing operator/root account. Confirm ordinary members cannot load `/api/admin/message-reports`.
5. Create a synthetic conversation between two disposable approved accounts. Report a received message, confirm queue receipt, remove it, refresh both clients, and verify the removal marker. Do not test with real users' messages.
6. Test and distribute a newly signed candidate after the remaining external-service settings and real deletion operations are verified. The previous IPA does not contain these fixes.

## Moderator operation

- Confirmed by the operator on 2026-09-06: the operator personally handles reports, checks `ghkdwp018@gmail.com` and the administrator report queue daily, and processes reports within 48 hours.
- Verify the operator's existing administrator account can access the deployed queue. No password needs to be shared. A backup during absences and a repeat-offender/suspension process remain to be specified.
- The queue refreshes automatically while open; this implementation does not send emails or staff notifications. The daily manual check is part of the confirmed operating procedure. The public support source now states the daily review and 48-hour target; verify it after deployment.
- Review reports, record a decision, and contact the reporter through the approved support process when needed. Reporter contact details are not exposed to the reported user.
- Approve retention/access rules for report evidence and moderator decisions. Resolved report evidence is now purged after 90 days; delete it earlier when no longer needed or on a valid deletion request. Lawfully retained evidence must be separated before resolving a report.
- Review filter misses and false positives; update additional phrases deliberately.

## Decisions blocking release

1. Confirm external-browser donations, or provide evidence of Apple-approved nonprofit fundraising and Apple Pay support before requesting an in-app alternative.
2. Operator details confirmed (2026-09-06): organization `대일외국어고등학교 장학회`, privacy officer `엄은숙`, privacy-request handler `황제철` at `ghkdwp018@naver.com`. General support/moderation still uses `ghkdwp018@gmail.com`. These are reflected in the privacy page and draft. HappyNanum supplies donation records to the foundation; exact received fields and contractual role remain to be confirmed. See `PRIVACY_RETENTION_REVIEW.md` for researched retention duties and their limits.
3. Revised operator decision (2026-09-06): accept deletion requests in the app and have the designated handler actually delete the data manually. This supersedes blanket indefinite retention as the intended policy. Member deletion targets and report/receipt retention are reflected in this release; external-service/backup settings and the foundation's tax status remain unverified.
4. Manual deletion is implemented locally: in-app durable requests, a root-only processing workflow, private status receipts, database/provider completion guards, and evidence/receipt expiry. The administrator must perform actual DB/files/backups/external-service erasure and individually notify the user; the console does not automatically erase those records or send email. See `MANUAL_ACCOUNT_DELETION_RUNBOOK.md`. Disposable MariaDB tests verify the workflow; production provider revocation, backup/Sentry erasure and operator readiness still require a real release-candidate test.
5. The public privacy page and native policy link are implemented. Resolve external-service contracts/storage/transfers/retention and the foundation's tax status before treating the policy as final. Deploy it with the corresponding backend/admin workflow, not ahead of that workflow. App Store Connect App Privacy remains a separate task. Its displayed date is a revision date, not an asserted production effective date.
6. Verify password/Apple/Kakao login and production APNs on an installed TestFlight candidate; prepare an approved reviewer account.

## Validation

- Manual deletion follow-up: Debug iOS build succeeds and 122 tests pass; frontend 139 unit tests, both SPA builds, changed-file ESLint and 5 public Playwright regressions pass. Backend tests/vet and the pinned MariaDB 10.1.38 manual-deletion lifecycle test pass. Tests use synthetic data; actual Apple/Kakao revocation, production delivery and external/backup deletion are unverified.
- iOS token compliance has zero violations. The workspace-wide design-system gate still reports existing web literals and the unrelated feed heading contract mismatch; those sources were not changed in this task.

- Privacy-page follow-up: frontend production build and changed-file ESLint pass; 12 footer/public-route/maintenance tests and two Playwright navigation/reload regressions pass. Browser checks at 375/768/1440px pass for footer access, section links, hash reload, trailing-slash entry, maintenance isolation, and horizontal overflow. The visit beacon was stubbed for isolated browser checks; without a running backend, the skill captures report its expected local 500 response. No production API was used.
- Full backend `go test ./...` and `go vet ./...` pass, including migration-source approval, invalid input, recipient authorization, admin authorization, and rollback checks.
- Browser fixture checks pass: fresh-session signup and completion, maintenance isolation, moderator queue and recorded removal. These use synthetic responses, not production accounts.
- Frontend and admin production builds pass. Both report bundle-size warnings.
- iOS Debug build and unsigned Release device build pass. Packaged manifest/config validation passes; five negative/positive release-validator tests pass.
- iOS design-system compliance: zero violations. Workspace-wide design validation has pre-existing web literal violations. The visual guard reports four missing capture/baseline pairs.
- Full frontend suite: 133 tests passed. iOS simulator suite: 119 tests passed on iOS 18.4 after integrating the concurrently completed logout-response fix; an iOS 26 runner failed during bootstrap before tests started.
- Pinned MariaDB 10.1.38 report lifecycle integration passed: unauthorized/hidden/deleted-message reporting rejected; duplicate evidence preserved; content removed and audit evidence retained; repeat resolution rejected.
- This is not physical-device or App Store sign-off.
