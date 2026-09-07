# iOS release fixes — September 6, 2026

These changes are prepared locally. They are not a production deployment or App Store submission.

## Active scope — operator decision

The current task adds automatic account erasure while retaining the existing manual workflow. On 2026-09-07 the operator explicitly brought Korean donation-record retention law back into scope. The current implementation and remaining production integrations are documented in [AUTOMATIC_ACCOUNT_ERASURE.md](AUTOMATIC_ACCOUNT_ERASURE.md). The operator confirmed a December 31 fiscal year end, digital HappyNanum originals and separate bank-transfer Excel originals; this does not establish legal compliance or guarantee App Review approval.

Use Apple's explicit review requirements for the active checklist. A detailed country-by-country transfer table or a universal seven-day backup/30-day Sentry period is not specified by Guideline 5.1.1(i). Accurate collection/use/sharing disclosures, equivalent third-party protection, retention/deletion and consent-withdrawal explanations remain in scope. Existing promises in the app/policy must match the actual candidate and operator workflow.

The repository-specific [Apple account deletion scope review](APPLE_ACCOUNT_DELETION_SCOPE.md) defines the proposed erasure targets, policy wording, and verification gaps. In particular, the existing footprint scan misses `VD_USR_SEQ`, `O_ACCOUNT_USR_SEQ`, indirect identifiers and historical upload ownership; its zero count alone does not prove complete erasure. That review predates the automatic engine. The automatic implementation now covers these explicit columns, durable file work and reviewed donation retention; external/legacy verification remains an integration prerequisite, not a production validation result.

## Implemented

- Automatic account erasure now preserves existing manual requests and permits root takeover/resume. It verifies provider revocation and reviewed donation retention, requires external erasure evidence, deletes app records in a transaction, retries durable file work, and publishes private receipt confirmation. Full production automation is not ready until the real Sentry/backup/legacy processor and organization-specific retention settings are connected. See [the automatic erasure runbook](AUTOMATIC_ACCOUNT_ERASURE.md).

- Public `/register` and `/register/complete` routes restore the app's email signup. Both bypass the temporary web maintenance gate without unlocking other routes. The signup page initializes its own auth state. Completion tells users to return to the native app after operator approval.
- `POST /api/message-reports` accepts a message ID, reason, and optional details. Only approved, authenticated recipients can report visible, undeleted incoming messages. Evidence is selected in SQL, never supplied by the caller. Repeated reports retain the original record.
- `/admin/message-reports` provides an authenticated administrator queue, pagination, processing history, mandatory decision notes, and content removal. Original evidence remains restricted to the moderator queue. Existing member management handles repeat offenders.
- The message service filters an initial set of abusive phrases before accepting new messages. NFKC normalization and removal of punctuation/spacing catch simple obfuscation. `MESSAGE_BLOCKED_PHRASES` adds comma-separated phrases without disabling the baseline. This is a limited rule-based filter, not a complete classifier.
- The Swift changes add reporting from an incoming message's context menu and the conversation menu, plus a public support link. Donation checkout opens in the system browser. The app owns a required-reason manifest and validates the packaged Release configuration.
- The web `/privacy` page follows the existing editorial layout, with an eight-section contents navigation, confirmed operator/contact details, and accurate withdrawal-retention disclosure. Both footers link to it. `/privacy` and `/privacy/` bypass the maintenance gate without unlocking other routes. This source change is not deployed and does not establish policy compliance; unknown external-service/backup details are explicitly identified.

## Deployment order

1. Candidate source manifest SHA-256: `e07cc5aed6d8468e04e680a87d5e9fc4f232edff3e951fd0374b67decedfff43`. The source manifest, regression pin, and environment example include migrations 055–063. Production approval/environment settings have not been changed. Review migration numbering against any concurrently developed backend changes; this branch adds `055_create_message_reports.sql` and `056_create_account_deletion_requests.sql`.
2. Apply the reviewed migrations using the project's normal procedure; 058 converts storage engines and needs a lock/time/disk review. No production migration has been executed by this task.
3. Deploy the backend and both SPAs together. Confirm signup works with an empty browser session and that `/api/message-reports` exists before distributing the new iOS build.
4. Give a designated moderator an existing operator/root account. Confirm ordinary members cannot load `/api/admin/message-reports`.
5. Create a synthetic conversation between two disposable approved accounts. Report a received message, confirm queue receipt, remove it, refresh both clients, and verify the removal marker. Do not test with real users' messages.
6. Test and distribute a newly signed candidate after the active App Review checks below are verified. The previous IPA does not contain these fixes.

## Moderator operation

- Confirmed by the operator on 2026-09-06: the operator personally handles reports, checks `ghkdwp018@gmail.com` and the administrator report queue daily, and processes reports within 48 hours.
- Verify the operator's existing administrator account can access the deployed queue. No password needs to be shared. Use the confirmed daily review/48-hour procedure and existing member management for repeat offenders.
- The queue refreshes automatically while open; this implementation does not send emails or staff notifications. The daily manual check is part of the confirmed operating procedure. The public support source now states the daily review and 48-hour target; verify it after deployment.
- Review reports, record a decision, and contact the reporter through the approved support process when needed. Reporter contact details are not exposed to the reported user.
- Approve retention/access rules for report evidence and moderator decisions. Resolved report evidence is now purged after 90 days; delete it earlier when no longer needed or on a valid deletion request. Lawfully retained evidence must be separated before resolving a report.
- Review filter misses and false positives; update additional phrases deliberately.

## Active App Review release checks

1. Publish a publicly accessible privacy policy and verify links from the installed app and App Store Connect. Check disclosures of collected data, collection methods, purposes, third-party access/protection, retention/deletion and consent withdrawal against the actual app and SDK configuration. The current policy's third-party-protection and withdrawal explanations need this focused check; do not replace unknown facts with unsupported assurances. Source: https://developer.apple.com/app-store/review/guidelines/#privacy (5.1.1).
2. Match App Store Connect App Privacy responses to app/backend/SDK data collection. Inspect the new archive's app-owned PrivacyInfo.xcprivacy and bundled SDK manifests. Local source/build tests do not replace archive inspection.
3. Deploy the deletion/reporting backend and admin workflow before distributing the corresponding candidate. On a disposable account verify in-app deletion request, access termination, actual automatic erasure and existing manual takeover, applicable Apple-token revocation, communicated timing and completion confirmation. Manual processing is permitted; status-only disablement is insufficient. Source: https://developer.apple.com/support/offering-account-deletion-in-your-app/ . The automatic worker deletes app data/files; external verification and receipt work can still require an operator. The private receipt is the completion channel.
4. Verify filtering, reporting, timely moderation, blocking and reachable support in the installed candidate (Guideline 1.2). The operator checks the report queue and ghkdwp018@gmail.com daily and handles reports within 48 hours.
5. Verify donation collection opens outside the app in the system browser and does not unlock digital benefits. The app must remain free for the external-fundraising route under 3.2.2(iv). Apple-approved in-app nonprofit fundraising would be a separate route; Korean public-benefit designation alone does not establish Apple approval.
6. Verify password/Apple/Kakao login, production API configuration and production APNs in the candidate. Provide an approved working reviewer account and clear review notes, including automatic deletion, operator follow-up and timing.

Retained operator details: 대일외국어고등학교 장학회; privacy contact 엄은숙; request handler 황제철 at ghkdwp018@naver.com. Korean donation retention is in scope under the latest instruction; see the automatic erasure runbook for the implemented conditional rules. Do not resume production deployment merely because the review scope changed; the earlier deployment cancellation remains effective.

## Latest release validation — September 7

See [the current handoff](ACCOUNT_ERASURE_RELEASE_HANDOFF.md) for migration 062, historical paths, minimized diagnostics, prepared keys/PG preview and the signed App Store IPA. No production app/DB deployment or App Store submission has occurred.

## Earlier validation

- Automatic-erasure follow-up: Go suite and vet pass, including disposable MariaDB 10.1.38 automatic/manual lifecycle tests, rollback on unknown references, retained archive expiry and donation aggregate preservation. Frontend 139 tests, both SPA builds and changed-file ESLint pass. Five public Playwright checks and administrator automatic/manual transition checks pass on mobile and desktop; a long error-code overflow was fixed. The iOS Debug simulator build passes. Real provider APIs, external processor and production deployment remain unverified.

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
