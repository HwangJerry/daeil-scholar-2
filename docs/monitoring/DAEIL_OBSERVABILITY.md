# Production observability and login security dataset

Production sends OTLP JSON over authenticated HTTPS to the dedicated cluster
collector. The collector sends metrics to Prometheus/VM, request logs to Loki
and spans to Tempo. `/api/health` retains its separate HTTPS/DB readiness probe.
The backend exports every 15 seconds; the procfs host agent every 30 seconds.
No listening monitoring port is added to production.

Backend configuration (systemd drop-in, root-readable secret file):
`TELEMETRY_ENABLED=true`, `TELEMETRY_ENDPOINT` (HTTPS origin only),
`TELEMETRY_TOKEN` (>=32 characters), `LOGIN_EXPORT_KEY` (>=32 characters),
`LOGIN_AUDIT_SPOOL=/app/logs/security-login-outbox` (backend-owned mode 0700).
Do not put these values into git, mobile applications or ordinary logs.

## Metrics, request traces and mobile breadcrumbs

Every registered API request contributes to counters and duration histograms.
Health, uploads and telemetry ingestion are excluded from application latency.
5xx errors and login attempts are traced, plus about 1/16 of ordinary requests
and sampled mobile journeys. Route templates replace personal path parameters.
Headers, body, query, IP, user IDs, tokens and error payloads are never exported.
The request log contains route, method, status, duration and trace ID.

Mobile `/api/mobile/telemetry` accepts only a closed schema and static screen,
action and route categories. Unknown fields, arbitrary paths, old events,
invalid IDs and large batches are rejected. Maximum 20 events / 16 KiB per
request, 20 requests per IP per minute. This diagnostic endpoint intentionally
works before login; the payload is client-reported and must not be used as
security evidence. API spans propagate `traceparent` to the backend. Device
identifiers never appear in breadcrumbs. Bounded queues drop oldest mobile
events after four minutes; diagnostic export cannot block login or UI actions.

## Login security dataset (schema version 1)

Migration 082 creates `LOGIN_SECURITY_EVENTS`. Password, Kakao and Apple login
attempts (including rate limits and version rejection) record server UTC time,
provider, outcome, HTTP status, duration, IP + IP source, installation UUID,
platform, app version/build and trace ID. Successful authentication, link-required
signup, invalid credentials, policy rejection, malformed requests, rate limits
and server errors are separate labels. Social browser callback redirects have
explicit success/link/rejection outcomes. Refresh is not a new login attempt.

Only loopback peers may supply forwarding headers; Apache appends its client IP
to XFF, so the rightmost valid address is used. Direct callers cannot spoof IP
through XFF. Native apps generate an installation UUID; this is not a hardware
identifier or reliable proof of a physical device. Missing metadata is empty,
never fabricated. Version/device claims are untrusted client reports.

The DB write has a 300 ms deadline. Failed writes fall back to mode-0600 files
in the persistent outbox, max 10,000 events, replayed in bounded batches.
`daeil_login_audit_write_failures_total` detects DB audit failures; outbox-full
and filesystem failures are also recorded without personal data in journal.
Events older than 90 days are deleted hourly (up to 20 batches of 10,000 within a five-second deadline) and discarded
from the outbox. The security table is excluded from ordinary deployment/weekly DB dumps so
older copies cannot extend this retention. Exported training files also require
a 90-day deletion schedule controlled by their recipient.

Root-authenticated export:
`GET /api/admin/security/login-events/export?from=<RFC3339>&to=<RFC3339>&after_id=0&limit=1000`
Returns NDJSON, maximum 31 days / 10,000 rows per page, `X-Next-Cursor` keyset
pagination. Date end is exclusive. By default IP and installation ID are
HMAC-SHA256 pseudonyms, domain-separated using a persistent export key. Use a
single key for reproducible datasets; rotate deliberately and keep outside git.
`raw=true` is available only through the same root-only route for incident work.
All exports have `Cache-Control: private, no-store`, and row-count audit logging.
No security source rows are forwarded to Lighthouse/Loki. Lighthouse receives
only provider/outcome counters, with no IP or device labels.

## Deployment and rollback

Use the immutable backend release process and apply migration 082. Preserve the
existing account-erasure feature gates after deploying. The migration is
additive; rolling back the application leaves the dataset intact. Disabling
`TELEMETRY_ENABLED` stops new login security logging and signal export. Stop the
independent host service to stop host signals. Keep the dataset for its defined
retention window; do not drop it as part of an application rollback.

The new mobile instrumentation requires an updated app binary. Existing 1.3.1
installations continue to provide server-side login result/time/IP, but cannot
supply the new installation UUID or client breadcrumbs. App Store privacy and
Play Data Safety declarations must describe installation identifiers, product
interaction and diagnostic collection when releasing the new binaries.
