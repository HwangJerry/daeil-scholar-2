# Verification review notifications

Successful canonical approve/reject operations enqueue a `verification.reviewed`
notification through the existing push delivery notifier. Failed/stale writes
do not notify. This service notification is independent of message preferences.
APNs and FCM carry recipient `user_id`, unique event ID and a one-day TTL.
Detailed rejection reasons remain in the authenticated verification response;
the lock-screen text directs the member to review them in the app.

Push registration/unregistration requires authentication, including pending and
rejected members, but no longer requires alumni approval. Community/message
APIs keep their approval gate. Clients refresh current status rather than trusting
the notification as authorization. A device list/state guard suppresses deliveries
for deleted users or an obsolete review status. Existing transient retries and
invalid-token cleanup also apply to these notifications.

Delivery uses the existing bounded in-memory worker; it is best effort and is
not a durable notification outbox. The authoritative status remains visible
on login/foreground/manual refresh if the push is missed. Production delivery
requires PUSH_ENABLED and configured APNs/FCM credentials, registered devices
and user permission. No production notification was sent during implementation.

Regression coverage includes review write failures, both platform targets,
message preference independence, stale/deleted eligibility suppression, payload
recipient/TTL, and allowing pending device registration without opening
community routes. The prior migration 065 and its deployment prerequisites
remain documented in profile-edit-approval.md; this change adds no migration.
