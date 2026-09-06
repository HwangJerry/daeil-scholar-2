# iOS release fixes — September 6, 2026

These changes are prepared locally. They are not a production deployment or App Store submission.

## Active scope — operator decision

The current task is iOS App Review readiness. Additional Korean-law/tax analysis, electronic donation receipt verification, and policy version-history features are deferred at the operator's request. Do not request those documents as prerequisites for continuing this task. This scope decision does not establish legal compliance or guarantee App Review approval.

Use Apple's explicit review requirements for the active checklist. A detailed country-by-country transfer table or a universal seven-day backup/30-day Sentry period is not specified by Guideline 5.1.1(i). Accurate collection/use/sharing disclosures, equivalent third-party protection, retention/deletion and consent-withdrawal explanations remain in scope. Existing promises in the app/policy must match the actual candidate and operator workflow.

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
6. Test and distribute a newly signed candidate after the active App Review checks below are verified. The previous IPA does not contain these fixes.

## Moderator operation

- Confirmed by the operator on 2026-09-06: the operator personally handles reports, checks `ghkdwp018@gmail.com` and the administrator report queue daily, and processes reports within 48 hours.
- Verify the operator's existing administrator account can access the deployed queue. No password needs to be shared. A backup during absences and a repeat-offender/suspension process remain to be specified.
- The queue refreshes automatically while open; this implementation does not send emails or staff notifications. The daily manual check is part of the confirmed operating procedure. The public support source now states the daily review and 48-hour target; verify it after deployment.
- Review reports, record a decision, and contact the reporter through the approved support process when needed. Reporter contact details are not exposed to the reported user.
- Approve retention/access rules for report evidence and moderator decisions. Resolved report evidence is now purged after 90 days; delete it earlier when no longer needed or on a valid deletion request. Lawfully retained evidence must be separated before resolving a report.
- Review filter misses and false positives; update additional phrases deliberately.

## Active App Review release checks

1. Publish a publicly accessible privacy policy and verify links from the installed app and App Store Connect. Check disclosures of collected data, collection methods, purposes, third-party access/protection, retention/deletion and consent withdrawal against the actual app and SDK configuration. The current policy's third-party-protection and withdrawal explanations need this focused check; do not replace unknown facts with unsupported assurances. Source: https://developer.apple.com/app-store/review/guidelines/#privacy (5.1.1).
2. Match App Store Connect App Privacy responses to app/backend/SDK data collection. Inspect the new archive's app-owned PrivacyInfo.xcprivacy and bundled SDK manifests. Local source/build tests do not replace archive inspection.
3. Deploy the deletion/reporting backend and admin workflow before distributing the corresponding candidate. On a disposable account verify in-app deletion request, access termination, actual manual erasure, applicable Apple-token revocation, communicated timing and completion confirmation. Manual processing is permitted; status-only disablement is insufficient. Source: https://developer.apple.com/support/offering-account-deletion-in-your-app/ . The admin console tracks and verifies work; it does not automatically erase all records or send result emails.
4. Verify filtering, reporting, timely moderation, blocking and reachable support in the installed candidate (Guideline 1.2). The operator checks the report queue and ghkdwp018@gmail.com daily and handles reports within 48 hours.
5. Verify donation collection opens outside the app in the system browser and does not unlock digital benefits. The app must remain free for the external-fundraising route under 3.2.2(iv). Apple-approved in-app nonprofit fundraising would be a separate route; Korean public-benefit designation alone does not establish Apple approval.
6. Verify password/Apple/Kakao login, production API configuration and production APNs in the candidate. Provide an approved working reviewer account and clear review notes, including the manual deletion steps and timing.

Retained operator details: 대일외국어고등학교 장학회; privacy contact 엄은숙; request handler 황제철 at ghkdwp018@naver.com. Detailed Korean-law research remains reference material in the privacy documents, not an additional active research task. Do not resume production deployment merely because the review scope changed; the earlier deployment cancellation remains effective.

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
