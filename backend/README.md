# dflh-saf-v2 Backend

## Mobile Push

### Environment

Configure APNs token-based auth with values from Apple Developer:

- `APNS_KEY_ID`: Key ID for the APNs Auth Key.
- `APNS_TEAM_ID`: Apple Developer Team ID.
- `APNS_BUNDLE_ID`: iOS app bundle id used as the default `apns-topic`.
- `APNS_PRIVATE_KEY`: Auth Key `.p8` content. Escaped `\n` is accepted.
- `APNS_PRIVATE_KEY_PATH`: Path to the Auth Key `.p8` file. Used when `APNS_PRIVATE_KEY` is empty.
- `APNS_ENVIRONMENT`: Default endpoint, `sandbox` or `production`.
- `APNS_REQUEST_TIMEOUT`: APNs HTTP request timeout, for example `5s`.

The server still accepts the previous `PUSH_APNS_*` names as fallbacks. Token-level `apnsEnvironment` takes precedence over `APNS_ENVIRONMENT`.

The backend also contains an FCM provider for Android push delivery, configured with Firebase service-account credentials:

- `FCM_PROJECT_ID`: Firebase project id. `FIREBASE_PROJECT_ID` is also accepted.
- `FCM_CREDENTIALS_JSON`: Firebase service account JSON. Do not commit this value.
- `FCM_CREDENTIALS_FILE`: Path to the service account JSON. `FIREBASE_SERVICE_ACCOUNT_FILE` and `GOOGLE_APPLICATION_CREDENTIALS` are accepted as fallbacks.
- `FCM_REQUEST_TIMEOUT`: FCM request timeout, for example `5s`.

When no push credentials are configured, the server keeps the existing no-op local behavior. When FCM credential env vars are present but invalid, startup fails during dependency wiring.

### Device API

`POST /api/push/device/register` requires authentication.

```json
{
  "platform": "ios",
  "deviceToken": "apns-device-token",
  "apnsEnvironment": "sandbox",
  "bundleId": "kr.dflh.saf",
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

Payload data intentionally omits message body, sender name, notice subject, phone, address, and other PII. The app treats push as a routing/sync hint and refetches backend data.

APNs headers set by the provider:

- `apns-topic`: token bundle id, or `APNS_BUNDLE_ID`
- `apns-push-type: alert`
- `apns-priority: 10`
- `apns-expiration`: `sent_at + ttl_sec`
- `apns-collapse-id`: `collapse_key`

### Operations

To verify production setup, register a device token from the iOS app, create a message or admin notice, and check logs for `push: send result` with `event_type`, `event_id`, `user_id`, token id/hash, APNs status, and reason. Private keys and raw device tokens must not appear in logs.

Registration-only smoke tests for Android do not require Firebase credentials because they only exercise authenticated token storage. Delivery smoke tests require Firebase service-account credentials, a Firebase project, app Firebase config, and a real FCM registration token.

APNs failure handling:

- `BadDeviceToken`, `Unregistered`, `DeviceTokenNotForTopic`: revoke the token.
- `TooManyRequests`, `ServiceUnavailable`, `InternalServerError`: transient; keep the token active and retry at the caller/job level when a queue is introduced.
- Auth/config errors such as `InvalidProviderToken`, `ExpiredProviderToken`, `MissingTopic`: check APNs key, team id, key id, bundle id, and server clock.

Current delivery is best-effort and in-process: `message.new` and `admin.notice` enqueue a goroutine and call APNs directly. There is no durable outbox, retry queue, dead-letter table, or delivery status table yet. A production hardening pass should add durable event storage, bounded retries with backoff for transient APNs responses, worker observability, and replay tooling.

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
