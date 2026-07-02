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

The server still accepts the previous `PUSH_APNS_*` names as fallbacks. Token-level `apnsEnvironment` takes precedence over `APNS_ENVIRONMENT`: debug/dev app tokens use `sandbox`, TestFlight and App Store tokens use `production`.

Configure Android FCM with Firebase service-account credentials:

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

`apnsEnvironment` must be `sandbox` or `production`. `environment` is also accepted as an alias. `bundleId` and `appBundleId` are both accepted. Registration is an upsert and reactivates the token.

Android registration uses the same endpoint:

```json
{
  "platform": "android",
  "deviceToken": "fcm-registration-token",
  "locale": "ko-KR"
}
```

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

APNs failure handling:

- `BadDeviceToken`, `Unregistered`, `DeviceTokenNotForTopic`: revoke the token.
- `TooManyRequests`, `ServiceUnavailable`, `InternalServerError`: transient; keep the token active and retry at the caller/job level when a queue is introduced.
- Auth/config errors such as `InvalidProviderToken`, `ExpiredProviderToken`, `MissingTopic`: check APNs key, team id, key id, bundle id, and server clock.

Android real-device smoke is unblocked at the backend registration/provider layer, but still requires a Firebase project, Android app Firebase config, configured service-account credentials, and a physical device or emulator with a real FCM registration token.
