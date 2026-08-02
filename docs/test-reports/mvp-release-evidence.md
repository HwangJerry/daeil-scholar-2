# MVP Release Evidence

- Observation cutoff: `2026-08-02 21:31:16 KST`
- Branch: `feature/mvp-implementation`
- Scope: `TEST-01` contract, integration, security, and release-gate evidence
- Result: **automated implementation gates GREEN; product release acceptance BLOCKED**

## Snapshot

| Repository | Worktree | HEAD |
|---|---|---|
| Web/backend/admin | `.worktrees/dflh-saf-v2-mvp/web` | `415d1a666a7081280df38100fcb686cc61680b7f` |
| Android | `.worktrees/dflh-saf-v2-mvp/android` | `c25fdad8f3671bf906ad57ed81e9b83dcb9e3860` |
| iOS | `.worktrees/dflh-saf-v2-mvp/ios` | `fbfce50674631f3a89b1575cbb4263a120eabaa4` |

All three worktrees contain broad approved, uncommitted MVP changes. Results in this report apply to those dirty closing snapshots, not to the committed HEADs alone. No commit, push, stash, checkout, reset, or clean was performed.

## Automated Gate Matrix

| Area | Command/evidence | Result |
|---|---|---|
| Backend tests | `GOMAXPROCS=1 GOMEMLIMIT=80MiB go test -count=1 -json -p 1 ./...` | **482 leaf tests passed**; 518 parent-inclusive test actions, 0 failed, 0 skipped; exit 0 |
| Backend race detector | `GOMAXPROCS=1 GOMEMLIMIT=160MiB go test -race -count=1 -p 1 ./...` | pass; exit 0 |
| Backend static/build | `go vet ./...`; `go build -o /tmp/dflh-saf-v2-mvp-server ./cmd/server` | pass; exit 0 |
| Frontend unit | `npm run test` | **15 files / 64 tests passed**; parent-route navigation state and foundation-detail return-link contracts included; exit 0 |
| Frontend lint/build | `npm run lint`; `npm run build` | pass; Vite production build completed |
| Frontend browser smoke | `npm run test:e2e` | Chromium **5 passed**; desktop/mobile navigation exposes only `소식` and `장학회 소개`, nested news/about routes retain their parent active state, direct-page and modal news details link directly back to `/`, foundation details link directly back to `/about`, keyboard focus remains correct, canonical donation content remains on home, and retired routes redirect home |
| Frontend local runtime | `npm run dev`; HTTP/browser check at `http://127.0.0.1:5173` | Vite remains running; root returns HTTP 200 and renders production feed/donation data through an OS-temporary local reverse proxy to `https://daeilfoundation.or.kr`; proxy allows only GET/HEAD/OPTIONS and rejects mutation methods with HTTP 405 |
| Admin unit | clean dependency bootstrap with `npm ci`, then `npm run test` | **2 files / 25 tests passed**; exit 0 |
| Admin lint/build | `npm run lint`; `npm run build` | pass; Vite production build completed |
| Android JVM/lint/build | closing rerun: `./gradlew :app:testDebugUnitTest --rerun-tasks`; `./gradlew :app:lintDebug :app:assembleDebug` | **32 suites / 164 tests**, 0 failures/errors/skips; `BUILD SUCCESSFUL` |
| Android device tests | closing rerun: `./gradlew :app:connectedDebugAndroidTest --rerun-tasks` on `emulator-5554`, API 36 | **25 tests**, 0 failures/errors/skips; `BUILD SUCCESSFUL` |
| iOS XCTest | project/scheme-based `xcodebuild ... '-only-testing:DflhSafV2SwiftTests' test` | **104 tests**, 0 failures; `TEST SUCCEEDED` |
| iOS release build | Release generic iOS Simulator, `CODE_SIGNING_ALLOWED=NO` | `BUILD SUCCEEDED`; compiler warnings 0 |
| iOS physical-device architecture build | Connected iOS device, Debug `iphoneos` arm64, `CODE_SIGNING_ALLOWED=NO` | `BUILD SUCCEEDED`; compile/link/bundle validation passed; no install or execution |
| iOS generic signed device build | Debug `generic/platform=iOS` with `-allowProvisioningUpdates` after account/capability setup | `BUILD SUCCEEDED`; strict code-signature verification passed; embedded entitlements contain `aps-environment=development` and `com.apple.developer.applesignin=[Default]` |
| iOS connected-device run | Debug signed build targeting attached iOS 26.5.2 device; `devicectl` install/launch/process inspection | `BUILD SUCCEEDED`; bundle `com.daeil.dflhsafv2` installed; launch succeeded; running executable confirmed after launch. The later console tunnel was invalidated and CoreDevice now reports the device unavailable, so current-process reinspection requires unlock/reconnect; the installed app remains on-device |
| iOS generated-project audit | XcodeGen `2.45.3` audit project from `project.yml`, generic simulator build | `BUILD SUCCEEDED` |
| Formatting/config | `git diff --check`; Android/iOS config lint; iOS `plutil -lint` | pass in all three worktrees |

Total executed leaf/runner-reported automated tests represented above: **869**, failures **0**.

## Security and Contract Evidence

- Backend canonical route/handler/service/repository tests are included in the fresh 482-leaf-test gate.
- Auth redaction tests exercise recursive access/refresh/identity-token and authorization-code removal and malformed-input fail-closed behavior through production `RedactAuthJSON`.
- Backend tests cover approved-only middleware, root/operator authorization, verification transitions, block suppression, SSE replay/cursor handling, push preferences/device lifecycle, account lifecycle, and donation ledger paths.
- Android and iOS contract fixtures are consumed through production DTO/parser/request seams rather than test-only mirror decoders.
- Android and iOS message SSE tests cover accepted event IDs, stale/duplicate suppression, `Last-Event-ID`, one-refresh/one-retry, resync, and ordered burst handling.
- No credential values, provider payloads, cookies, payment data, or connection strings are reproduced in this report.

## Warnings and Dependency Findings

- Backend vet/build, frontend lint/build, Android gates, and iOS source compilation completed without source compiler/linter failures.
- Admin production build emits an existing Rollup chunk-size warning: the main minified JavaScript chunk is approximately `2,566.85 kB` (`856.01 kB` gzip). This is a performance/release-budget issue, not a compile failure.
- Before remediation, full-tree `npm audit --json` reported 16 findings per SPA and production-only audit reported 3 package findings. After the approved non-breaking remediation, fresh full-tree audit reports **13 high package nodes** per SPA while production-only audit reports **2 high package nodes** representing one React Router RSC-mode advisory.
- Before remediation, Go 1.25.2 `govulncheck ./...` reported **33 reachable vulnerabilities** from the standard library and five modules. After the approved toolchain/dependency remediation, fresh `govulncheck` reports **0 reachable vulnerabilities**.
- Xcode emits only the informational AppIntents metadata-skip warning because the app has no AppIntents dependency; app source compiler warnings are 0.
- The physical Debug run showed that Kakao SDK informational logging can emit sensitive provider token-response fields to an attached development console. No token value is reproduced in this report, and console-attached execution was stopped. Suppress or redact provider SDK auth logging before collecting or sharing further device logs, and rotate/revoke the observed test session.

## Dependency Remediation and Risk Acceptance

OS-temporary spikes first validated the following version set. The project owner then approved its application to the implementation worktree.

- **Backend — applied and GREEN:** `go 1.25.0` directive with deployed verification on Go 1.25.12; `chi/v5 5.2.2`, `jwt/v5 5.2.2`, `goldmark 1.7.17`, `x/image 0.43.0`, `x/net 0.53.0`, `x/sys 0.43.0`, and `x/text 0.38.0`. The fresh full race-enabled suite, vet, server build, and `govulncheck` all pass; reachable vulnerability count is 0.
- **Frontend/Admin — applied and GREEN:** exact `dompurify 3.4.12` and `react-router-dom 7.18.2`, followed by non-breaking `npm audit fix` lockfile updates without `--force`. Fresh frontend 64 tests, lint, production build, Chromium E2E 5/5 and admin 25 tests, lint, production build all pass.
- **Router residual acceptance:** both SPAs are Vite client applications and do not configure React Server Components. The remaining production audit result is two package nodes for one RSC-mode CSRF advisory. No safe `react-router-dom` v8 package is currently published. The project owner accepts this specific non-RSC exposure through **2026-08-29 or the publication of a supported safe `react-router-dom` release, whichever occurs first**. Re-audit is mandatory at that boundary. This acceptance does not cover new advisories or RSC/SSR enablement.
- **Dev/build residual acceptance:** full-tree audit retains 13 high package nodes. Eleven are the `brace-expansion` advisory propagated through ESLint/TypeScript-ESLint dependency paths; two are the accepted Router package nodes. A temporary global `brace-expansion 5.0.9` override spike was rejected because both SPA lint commands crashed inside Minimatch with `TypeError: expand is not a function`. A supported ESLint 10 spike required the latest React lint plugins and then surfaced 13 new frontend rule failures, so it is a source-refactor/lint-policy migration rather than a lockfile-only fix. The project owner accepts the CI/build-only `brace-expansion` exposure through **2026-08-29**. Before expiry, either complete the dedicated ESLint 10 source migration or renew the risk decision from a fresh audit. No incompatible override or disabled lint rule was applied.

## Backend Container Build Contract

- The production builder was aligned from `golang:1.23-alpine` to `golang:1.25.12-alpine` so the deployed binary does not silently use the vulnerable standard library remediated above.
- A clean Docker build exposed an inherited private-module dependency on `github.com/cherryberryyogurt/debug-agent-go-client`. The original image had neither Git nor a private-module credential path. Git and CA certificates are now explicit builder-only prerequisites.
- The approved Dockerfile contract uses a required BuildKit secret named `github_token`, `GOPRIVATE`, and a temporary mode-0600 `.netrc` created and removed in the same `RUN` layer. The token is not accepted as an ARG/ENV value and is not written to the final image.
- A new `.dockerignore` excludes `.env` variants except `.env.example`, Git metadata, logs, credential/key file patterns, build/coverage/temp/upload artifacts, and local server output from the BuildKit context. The focused ad-hoc verifier first observed a valid RED because the file was absent, then passed after the exclusions were added.
- Ad-hoc fail-closed verification after the final Docker context edit confirms that a build without the secret exits non-zero with `secret github_token: not found` before dependency download. All OS-temporary verifiers were removed after execution.
- A separate synthetic-invalid secret check confirms that BuildKit mounts the secret and reaches the private-module fetch path without printing the synthetic value. The expected authentication failure is wiring evidence only, not an authenticated image-build success; the synthetic secret file was removed.
- A clean authenticated image build remains **BLOCKED** until an authorized CI/staging secret is connected. No credential was read, printed, or substituted from the host during this verification.

## Acceptance Blockers

`TEST-01` product acceptance remains **BLOCKED** until all required items below have current evidence:

1. **Staging integration/E2E:** the currently reachable local Kubernetes namespace contains only MariaDB `10.11` and no backend deployment; it is not the target MariaDB `10.1.38` staging topology. Approved/rejected/reapproval account flows, alumni search, two-party conversation, block/unblock, push settings, account deletion, and root/operator capability matrix against the deployed backend and target schema remain unexecuted.
2. **Resource/load gate:** concurrent SSE connections, message send, donation summary, and import job under the target `1 core / 1 GB` staging topology. Backend unit/build execution with `GOMAXPROCS=1` and `GOMEMLIMIT=80MiB` does not substitute for this load test.
3. **Physical-device/provider gate:** only an Android emulator is attached. The iOS account, profile, Sign in with Apple entitlement, APNs development entitlement, signed physical-device build, installation, launch, and live process are now verified. Android FCM and iOS APNs delivery at least once each, foreground/background/terminated behavior, tap routing, preview/notification-off behavior, token rotation, and blocked-recipient suppression remain unexecuted.
4. **Provider authentication:** real Kakao and Apple login/link flows with configured provider credentials; `KAKAO_NATIVE_APP_KEY_CONFIGURED=no` remains unresolved for mobile release smoke.
5. **HappyNanum device flow:** personal-information entry, identity-verification popup, external payment-app transition, and success/cancel return on a physical iOS device without executing an unapproved financial transaction.
6. **Account-deletion reconciliation:** server-side personal-data anonymization/deletion, refresh-session removal, push-token removal, social credential revoke/outbox state, and retained statutory transaction fields verified after a real staging deletion.
7. **Secret-remediation gate:** migrate historically plaintext deployment credentials to the approved secret store, rotate them, connect an authorized BuildKit `github_token`, and complete a clean backend image build before release; values remain `[REDACTED]`.

## Decision

The cross-platform automated implementation snapshot is reproducibly GREEN, but the TEST-01 acceptance criterion requiring physical-device push and staging integration/load evidence is not met. Do not authorize `TEST-02` production-data rehearsal or `RELEASE-01` from this report alone.
