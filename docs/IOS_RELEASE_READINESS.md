# iOS release fixes — September 6, 2026

These changes are prepared locally. They are not a production deployment or App Store submission.

## Implemented

- Public `/register` and `/register/complete` routes restore the app's email signup. Both bypass the temporary web maintenance gate without unlocking other routes. The signup page initializes its own auth state. Completion tells users to return to the native app after operator approval.
- `POST /api/message-reports` accepts a message ID, reason, and optional details. Only approved, authenticated recipients can report visible, undeleted incoming messages. Evidence is selected in SQL, never supplied by the caller. Repeated reports retain the original record.
- `/admin/message-reports` provides an authenticated administrator queue, pagination, processing history, mandatory decision notes, and content removal. Original evidence remains restricted to the moderator queue. Existing member management handles repeat offenders.
- The message service filters an initial set of abusive phrases before accepting new messages. NFKC normalization and removal of punctuation/spacing catch simple obfuscation. `MESSAGE_BLOCKED_PHRASES` adds comma-separated phrases without disabling the baseline. This is a limited rule-based filter, not a complete classifier.
- The Swift changes add reporting from an incoming message's context menu and the conversation menu, plus a public support link. Donation checkout opens in the system browser. The app owns a required-reason manifest and validates the packaged Release configuration.

## Deployment order

1. Candidate source manifest SHA-256: `979a1fe9f4a2416bbb4cdada329fe7d6a1a163d5d403d522656cea954770029a`. The source manifest, regression pin, and environment example include migration 055. Production approval/environment settings have not been changed. Review migration numbering against any concurrently developed backend changes; this branch adds `055_create_message_reports.sql`.
2. Apply that additive migration using the project's normal migration procedure. No production migration has been executed by this task.
3. Deploy the backend and both SPAs together. Confirm signup works with an empty browser session and that `/api/message-reports` exists before distributing the new iOS build.
4. Give a designated moderator an existing operator/root account. Confirm ordinary members cannot load `/api/admin/message-reports`.
5. Create a synthetic conversation between two disposable approved accounts. Report a received message, confirm queue receipt, remove it, refresh both clients, and verify the removal marker. Do not test with real users' messages.
6. Test and distribute a newly signed candidate after the remaining policy/deletion decisions are resolved. The previous IPA does not contain these fixes.

## Moderator operation still required

- Assign a person and backup to monitor the queue and the public support channel.
- Agree a response target and a repeat-offender/suspension process. The queue refreshes automatically while open; this implementation does not send emails or staff notifications.
- Review reports, record a decision, and contact the reporter through the approved support process when needed. Reporter contact details are not exposed to the reported user.
- Approve retention/access rules for report evidence and moderator decisions. There is no automatic evidence purge until that policy is specified.
- Review filter misses and false positives; update additional phrases deliberately.

## Decisions blocking release

1. Confirm external-browser donations, or provide evidence of Apple-approved nonprofit fundraising and Apple Pay support before requesting an in-app alternative.
2. Confirm the responsible organization and public privacy/support contact. The existing support page uses `ghkdwp018@gmail.com`; the foundation footer uses `dflhs.scholar@gmail.com`. These may have different owners.
3. Approve retention per data category: ordinary profiles and uploaded images; sent/received messages; donation receipts/ledger; security records; report evidence; backups. Specify purpose, duration, and responsible person for any retained category.
4. Implement and verify actual account erasure and provider-token revocation. Current `AnonymizeAccountForDeletion` only changes account status, and the legacy revocation worker does not receive new deletion jobs. Do not call this fixed.
5. Finalize the privacy-policy draft, publish at a public HTTPS URL, add native/web links, and complete App Store Connect App Privacy separately.
6. Verify password/Apple/Kakao login and production APNs on an installed TestFlight candidate; prepare an approved reviewer account.

## Validation

- Full backend `go test ./...` and `go vet ./...` pass, including migration-source approval, invalid input, recipient authorization, admin authorization, and rollback checks.
- Browser fixture checks pass: fresh-session signup and completion, maintenance isolation, moderator queue and recorded removal. These use synthetic responses, not production accounts.
- Frontend and admin production builds pass. Both report bundle-size warnings.
- iOS Debug build and unsigned Release device build pass. Packaged manifest/config validation passes; five negative/positive release-validator tests pass.
- iOS design-system compliance: zero violations. Workspace-wide design validation has pre-existing web literal violations. The visual guard reports four missing capture/baseline pairs.
- Full frontend suite: 133 tests passed. iOS simulator suite: 114 tests passed on iOS 18.4; an iOS 26 runner failed during bootstrap before tests started.
- Pinned MariaDB 10.1.38 report lifecycle integration passed: unauthorized/hidden/deleted-message reporting rejected; duplicate evidence preserved; content removed and audit evidence retained; repeat resolution rejected.
- This is not physical-device or App Store sign-off.
