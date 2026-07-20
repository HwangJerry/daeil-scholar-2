# dflh-saf-v2 Backend

## Mobile Push

### Environment

Configure APNs token-based auth with values from Apple Developer:

- `APNS_TEAM_ID`: Apple Developer Team ID.
- `APNS_BUNDLE_ID`: iOS app bundle id used as the default `apns-topic`.
- `APNS_SANDBOX_KEY_ID`: Key ID for the sandbox APNs Auth Key.
- `APNS_SANDBOX_PRIVATE_KEY`: Sandbox Auth Key `.p8` content. Escaped `\n` is accepted.
- `APNS_SANDBOX_PRIVATE_KEY_PATH`: Path to the sandbox Auth Key `.p8` file. Used when the inline value is empty.
- `APNS_PRODUCTION_KEY_ID`: Key ID for the production APNs Auth Key.
- `APNS_PRODUCTION_PRIVATE_KEY`: Production Auth Key `.p8` content. Escaped `\n` is accepted.
- `APNS_PRODUCTION_PRIVATE_KEY_PATH`: Path to the production Auth Key `.p8` file. Used when the inline value is empty.
- `APNS_ENVIRONMENT`: Default endpoint, `sandbox` or `production`.
- `APNS_REQUEST_TIMEOUT`: APNs HTTP request timeout, for example `5s`.

Token-level `apnsEnvironment` selects both the APNs endpoint and the matching environment-specific signing key. Credentials never fall back across sandbox and production. The server can run with one environment configured, but a notification for an unconfigured environment fails with a configuration error and the outbox marks that job `DEAD`; configure both keys before production tokens can enqueue jobs.

The server still accepts the previous `APNS_KEY_ID`, `APNS_PRIVATE_KEY*`, and `PUSH_APNS_*` names as fallbacks. Legacy credentials are assigned only to the environment selected by `APNS_ENVIRONMENT` (or `PUSH_APNS_USE_SANDBOX`) and are never shared across both environments. Environment-specific variables take precedence. On a systemd host, store `.p8` files outside the application directory and use the `*_PRIVATE_KEY_PATH` variables instead of inline private key values.

APNs credential files are read and parsed during server startup. A partial credential set, unreadable file, invalid `.p8`, missing team id, or missing bundle id prevents startup instead of silently disabling delivery.

iOS delivery uses the DB-backed push outbox worker:

- `PUSH_OUTBOX_BATCH_SIZE`: due jobs claimed per tick. Default `50`.
- `PUSH_OUTBOX_POLL_INTERVAL`: worker poll interval. Default `5s`.
- `PUSH_OUTBOX_MAX_ATTEMPTS`: max delivery attempts before `DEAD`. Default `8`.
- `PUSH_OUTBOX_BASE_BACKOFF`: first transient retry delay. Default `30s`.
- `PUSH_OUTBOX_MAX_BACKOFF`: retry delay cap. Default `15m`.
- `PUSH_OUTBOX_RECOVERY_TIMEOUT`: `PROCESSING` jobs older than this are recovered. Default `5m`.
- `PUSH_OUTBOX_REQUEST_TIMEOUT`: per-APNs send timeout from the worker. Default `10s`.

The backend also contains an FCM provider for Android push delivery, configured with Firebase service-account credentials:

- `FCM_PROJECT_ID`: Firebase project id. `FIREBASE_PROJECT_ID` is also accepted.
- `FCM_CREDENTIALS_JSON`: Firebase service account JSON. Do not commit this value.
- `FCM_CREDENTIALS_FILE`: Path to the service account JSON. `FIREBASE_SERVICE_ACCOUNT_FILE` and `GOOGLE_APPLICATION_CREDENTIALS` are accepted as fallbacks.
- `FCM_REQUEST_TIMEOUT`: FCM request timeout, for example `5s`.

When APNs credentials are not configured, iOS push jobs can still be enqueued but the outbox worker does not start and logs that APNs is not configured. When FCM credential env vars are present but invalid, startup fails during dependency wiring.

### Device API

`POST /api/push/device/register` requires authentication.

```json
{
  "platform": "ios",
  "deviceToken": "apns-device-token",
  "apnsEnvironment": "sandbox",
  "bundleId": "com.daeil.dflhsafv2",
  "locale": "ko-KR"
}
```

Validation rules:

- `platform` must be `ios` or `android`; otherwise the API returns `400 INVALID_PLATFORM`.
- `deviceToken` is required and must be at most 512 characters; otherwise the API returns `400 INVALID_TOKEN`.
- Device tokens are stored up to 512 ASCII characters with binary collation and must never be truncated.
- `apnsEnvironment` is required for iOS. `environment` is accepted as a request alias.
- `debug`, `dev`, and `development` normalize to `sandbox`.
- `release`, `prod`, `production`, `testflight`, and `appstore` normalize to `production`.
- Missing or unknown iOS APNs environments return `400 INVALID_APNS_ENVIRONMENT`.
- `bundleId` and `appBundleId` are both accepted. Registration is an upsert and reactivates the token.

Android registration uses the same endpoint:

```json
{
  "platform": "android",
  "deviceToken": "fcm-registration-token",
  "locale": "ko-KR"
}
```

Android tokens are stored without APNs routing metadata; `APNS_ENVIRONMENT` and `BUNDLE_ID` remain `NULL` for FCM registrations. Migration `033_backfill_android_push_token_metadata_and_length.sql` backfills existing Android rows to clear any APNs metadata and expands token storage for longer FCM registration tokens.

`POST /api/push/device/unregister` requires authentication.

```json
{
  "deviceToken": "apns-device-token"
}
```

Unregister marks the token inactive. APNs permanent failures such as `BadDeviceToken`, `Unregistered`, and `DeviceTokenNotForTopic` mark tokens revoked. FCM permanent registration-token failures such as not-registered or sender/project mismatch also mark tokens revoked; transient FCM failures keep tokens active.

### Push Preferences

`GET /api/push/preferences` requires authentication and returns the current user's mobile push preferences. Missing rows default to all push types enabled for backwards compatibility.

```json
{
  "noticeEnabled": true,
  "messageEnabled": true
}
```

`PUT /api/push/preferences` requires both fields and persists the same response shape:

```json
{
  "noticeEnabled": false,
  "messageEnabled": true
}
```

`message.new` delivery checks `messageEnabled`; `admin.notice` delivery checks `noticeEnabled` before direct send or iOS outbox enqueue. Preference lookup failures fail closed and are logged so opt-out users are not accidentally notified.

### Payload Contract

Every APNs custom payload and FCM data payload includes:

- `event_type`: `message.new` or `admin.notice`
- `event`: same value as `event_type`
- `event_id`
- `template_key`
- `template_version`
- `ttl_sec`
- `collapse_key`
- `user_id`
- `args`
- `deep_link`
- `sent_at`

`message.new` uses `args.sender_seq`, `args.recvr_seq`, and `deep_link: "/messages/{senderSeq}"`.

`admin.notice` uses `args.post_seq` and `deep_link: "/feed/{postSeq}"`.

Custom routing data intentionally omits message body, sender name, notice subject, phone, address, and other PII. For `message.new`, the user-visible APNs/FCM alert title is the sender name and the body is a whitespace-normalized preview truncated to 80 Unicode characters. For `admin.notice`, the alert title is `새 소식` and the body is the whitespace-normalized notice subject. Blank values fall back to generic copy. iOS outbox rows retain this alert title/body until operational cleanup because retries must reproduce the same notification. The app still treats the custom payload as a routing/sync hint and refetches backend data.

APNs headers set by the provider:

- `apns-topic`: token bundle id, or `APNS_BUNDLE_ID`
- `apns-push-type: alert`
- `apns-priority: 10`
- `apns-expiration`: `sent_at + ttl_sec`
- `apns-collapse-id`: `collapse_key`

### Durable Outbox

`message.new` and `admin.notice` iOS pushes are inserted into `ALUMNI_PUSH_OUTBOX` once per active iOS device token. The unique key on `(EVENT_ID, MDT_SEQ)` prevents duplicate jobs for the same event/token pair.

Outbox statuses:

- `PENDING`: job is stored and ready when `NEXT_ATTEMPT_AT <= NOW()`.
- `PROCESSING`: a worker claimed the job using a claim token.
- `SENT`: APNs accepted the notification.
- `FAILED`: a transient error occurred and retry is scheduled.
- `DEAD`: delivery should not be retried automatically.

Transient APNs errors (`TooManyRequests`, `ServiceUnavailable`, `InternalServerError`, network timeouts, context deadlines) use exponential backoff from `PUSH_OUTBOX_BASE_BACKOFF` up to `PUSH_OUTBOX_MAX_BACKOFF`. Invalid token errors (`BadDeviceToken`, `Unregistered`, `DeviceTokenNotForTopic`) revoke the device token and mark the job `DEAD`. Payload/config/auth errors are also marked `DEAD` and logged for operator action.

The worker recovers stuck `PROCESSING` rows older than `PUSH_OUTBOX_RECOVERY_TIMEOUT` back to retryable `FAILED`.

Operational queries:

```sql
SELECT COUNT(*) AS pending_count
FROM ALUMNI_PUSH_OUTBOX
WHERE STATUS IN ('PENDING','FAILED') AND NEXT_ATTEMPT_AT <= NOW();

SELECT COUNT(*) AS dead_count
FROM ALUMNI_PUSH_OUTBOX
WHERE STATUS = 'DEAD';

SELECT TIMESTAMPDIFF(SECOND, MIN(CREATED_AT), NOW()) AS oldest_pending_age_sec
FROM ALUMNI_PUSH_OUTBOX
WHERE STATUS IN ('PENDING','FAILED');
```

Dead jobs should be replayed only after confirming the cause is fixed, the payload contract is still valid, and the device token was not revoked for an invalid-token reason.

### Operations

To verify production setup, register a device token from the iOS app, create a message or admin notice, confirm rows appear in `ALUMNI_PUSH_OUTBOX`, and check logs for `push outbox delivery result` with `outbox_id`, `event_type`, `event_id`, `user_id`, token id/hash, attempt count, status, and APNs reason. Private keys, raw device tokens, and full payload bodies must not appear in logs.

Registration-only smoke tests for Android do not require Firebase credentials because they only exercise authenticated token storage. Delivery smoke tests require Firebase service-account credentials, a Firebase project, app Firebase config, and a real FCM registration token. Android notifications use the high-importance channel id `dflh_push_v2`, which must remain aligned with the app manifest default channel and its application-startup channel creation.

APNs failure handling:

- `BadDeviceToken`, `Unregistered`, `DeviceTokenNotForTopic`: revoke the token.
- `TooManyRequests`, `ServiceUnavailable`, `InternalServerError`: transient; keep the token active and retry through the outbox worker.
- Auth/config errors such as `InvalidProviderToken`, `ExpiredProviderToken`, `MissingTopic`: check APNs key, team id, key id, bundle id, and server clock.

### Tests

Run targeted backend tests with:

```sh
go test ./internal/service ./internal/handler
```

This repository currently uses a local module replacement:

```go
replace github.com/dflh-saf/social-auth => ../../dflh-social-auth
```

The sibling `../../dflh-social-auth` checkout must exist relative to `backend/`, or Go setup fails before package tests start. Do not remove or rewrite this replace just to run push tests; prepare the sibling checkout or adjust the workspace consistently with the project setup.
