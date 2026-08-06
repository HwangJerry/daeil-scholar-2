# Kakao Canonical Auth Production Rollout Plan

> 상태: **승인된 실행 기준 — T00/T05/T10/T20 및 T30 local/source 완료, P10 inactive pre-stage attempt 2 완료, maintenance sentinel·DB migration·backend deployment 미수행, SMTP/permission 사용자 소유 예외, 전체 rollout NO-GO**<br>
> 작성 기준일: 2026-08-04<br>
> Plan path: `docs/plans/kakao-canonical-auth-production-rollout.md`<br>
> Ouroboros interview: `interview_20260803_050358`<br>
> Seed: `seed_65da4507f693`<br>
> Seed SHA-256: `0ec39d2389e8bd64b2d7f13460d1e0c944f9a285c27606962bbd9710fb466632`<br>
> Seed generation authorization: **`y`** — `Proceed to generate Seed specification?` prompt at `2026-08-04T06:09:31+09:00`<br>
> Seed generated at: `2026-08-04T06:11:17.309937+09:00` — Seed `metadata.created_at`<br>
> Post-generation Whole-Seed approval response: **완성된 Whole-Seed를 승인한다**<br>
> Post-generation Whole-Seed approval recorded at: `2026-08-04T07:36:50+09:00`
> Supplemental decisions: `~/.ouroboros/decisions/interview_20260803_050358.md`

> **2026-08-06 T90 approval amendment:** 사용자는 T90 production 실행을 `read-only preflight → backend deployment → controlled smoke → maintenance release` 네 단계로 분리하고, 각 단계가 이전 단계 PASS와 별도의 exact-command 명시 승인을 요구하도록 확정했다. 이 amendment는 아래의 single-gate/automatic-release 표현을 대체한다. Deployment 또는 smoke PASS만으로 다음 단계가 자동 승인되지 않으며, release는 deployment와 full smoke가 모두 PASS한 뒤에도 별도 승인을 받아야 한다.

## 0. 문서 사용 규칙

이 문서는 production push/outbox를 보존하면서 Kakao mobile auth를 canonical 계약으로 전환하는 전체 실행 기준이다. 이 문서의 존재는 DB mutation, commit, deployment 또는 mobile channel upload 승인이 아니다.

실행자는 자신이 배정받은 Task ID의 **Scope / Exact files / Dependencies / Verification / Stop conditions**만 선택적으로 읽되, 아래 공통 규칙은 항상 적용한다.

1. 각 수직 작업은 `RED → 최소 GREEN → package regression → full regression → REFACTOR` 순서를 지킨다.
2. Production DB migration, backend deployment, Android Play 내부 테스트, iOS TestFlight는 각각 별도 사용자 명시 승인 gate를 갖는다.
3. Commit, push, amend, reset, clean, stash는 별도 승인 없이 하지 않는다.
4. Dirty worktree에서 production artifact를 build/deploy하지 않는다.
5. Credential, provider token, native app key, signing key hash, DB credential, smoke token은 source·log·manifest에 출력하지 않는다.
6. Production 단계의 evidence는 credential을 제거한 command, exit code, timestamp, checksum, invariant 결과만 기록한다.
7. Gate가 실패하면 다음 dependency로 진행하지 않는다.

---

## 1. 목표와 최종 Definition of Done

### 1.1 목표

현재 production의 push/outbox 동작과 legacy PHP 공존을 보존하면서 다음 canonical Kakao mobile auth를 backend, Android, iOS에 동일하게 배포한다.

```text
Android/iOS Kakao SDK
  → provider access token
  → POST /api/auth/kakao/mobile
     {"grantType":"access_token","accessToken":"[REDACTED]"}

linked identity
  → status="authenticated"
  → nested session
  → session.user.verification.status

unlinked identity
  → status="linkRequired"
  → nested linkRequired
  → provider email은 UI prefill일 뿐 identity proof가 아님
```

### 1.2 최종 완료 조건

다음 조건이 모두 참일 때만 완료로 판정한다.

- Canonical backend contract, central authorization, refresh rotation, logout, social-link continuation regression PASS.
- Migration `036–039`가 exact MariaDB `10.1.38`에서 clean apply, reapply, backfill, partial-failure 검증 PASS.
- Clean commit 기반 artifact와 deployment manifest의 provenance/checksum 검증 PASS.
- Full logical DB dump, checksum, off-host 보호 사본, 격리 restore 검증 PASS.
- 별도 승인 후 production migration `036 → 037 → 038 → 039`와 단계별 post-check PASS.
- 별도 승인된 production read-only preflight PASS.
- 별도 승인 후 backend deployment PASS.
- 별도 승인 후 server-side smoke PASS.
- Deployment와 smoke가 모두 PASS하고 별도 release 승인을 받은 경우에만 maintenance write 차단 해제.
- 별도 승인 후 Play Console 내부 테스트 Android release와 TestFlight iOS release E2E PASS.
- Maintenance 해제 직후 시작한 24시간 push/outbox 관찰 PASS.
- Rollout 기인 데이터 유실, 중복 발송, stuck `PROCESSING`, 예기치 않은 `DEAD`, worker 중단, 구조적 Android/iOS push 실패가 0건.

최종 논리식:

```text
FinalCompletion =
  ServerRolloutAccepted
  AND AndroidAccepted
  AND iOSAccepted
  AND PushOutboxObservation24hPassed
```

---

## 2. 현재 기준선과 provenance 한계

### 2.1 Backend worktree

```text
Repository: /Users/jerryhwang/Workspace/03_daeil/.worktrees/kakao-canonical-auth-hotfix/web
Branch: fix/kakao-canonical-auth
HEAD: cd9f8f0807c0292091a127e2fbc68ff99e6a2e1d
State: modified + untracked
```

Production binary metadata는 `vcs.modified=true`다. 따라서 `cd9f8f0807c0292091a127e2fbc68ff99e6a2e1d` clean revision을 exact production source라고 주장하지 않는다. Push/outbox 보존 여부는 source diff만이 아니라 배포 전후 production black-box 동작, DB 지표, worker 로그와 실제 device 수신을 비교해 판정한다.

### 2.2 Android worktree

```text
Repository: /Users/jerryhwang/Workspace/03_daeil/.worktrees/dflh-saf-v2-mvp/android
Branch: feature/mvp-implementation
HEAD: c25fdad8f3671bf906ad57ed81e9b83dcb9e3860
State: modified + untracked
```

현재 emulator evidence는 pre-deploy smoke일 뿐 최종 acceptance가 아니다.

### 2.3 iOS worktree

```text
Repository: /Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2-swift
Branch: feature/social-auth-security
HEAD: fbfce50674631f3a89b1575cbb4263a120eabaa4
State: modified + untracked
```

현재 연결 iPhone 직접 설치 evidence는 pre-deploy smoke일 뿐 최종 TestFlight acceptance가 아니다.

### 2.4 2026-08-04 T05 scope amendment

사용자는 SMTP credential 값 노출 여부와 대응을 현재 rollout에서 판단하지 않고 직접 기억·대응하기로 명시했다. 이어서 production EnvironmentFile permission 변경도 보류하고 현재 `0640 root:alumni-backend` 구성을 유지한 채 T05를 종료하도록 승인했다.

따라서 이 amendment는 기존 Whole-Seed와 아래 T05/G0 기술 중 충돌하는 부분을 다음과 같이 대체한다.

- Agent scope에서 SMTP AUTH probe, replacement 생성/교체, `SMTP_PASSWORD` runtime 변경, old credential revoke, 그 목적의 restart/smoke를 제외한다.
- SMTP finding은 evidence에 관측 사실과 사용자 소유 deferred risk로 남기되 G1B/T60 또는 rollout 진행을 차단하지 않는다.
- Production EnvironmentFile permission/owner 변경도 현재 scope에서 제외하고 현재 구성을 유지한다.
- T05 acceptance는 current source/artifact plaintext 제거, external EnvironmentFile 계약, `deploy.sh` 비밀 비노출 remote validation/query, scanner/test/static verification PASS로 닫는다.
- 이후 새로 확인되는 non-SMTP active credential 노출은 이 예외에 포함되지 않으며 별도 G0 판단을 다시 요구한다.

---

## 3. 불변 계약

### 3.1 Canonical outcome

- Endpoint가 Kakao 전용이므로 request에 별도 `provider`를 넣지 않는다.
- 내부 DB provider code는 `KT`다.
- 신규 backend는 top-level `pending` 또는 `rejected` auth outcome을 만들지 않는다.
- Pending/rejected는 `authenticated` session 내부 authorization state다.
- Legacy flat response fallback decoder를 Android/iOS에 추가하지 않는다.
- Provider email 일치만으로 기존 회원에게 자동 연결하지 않는다.

### 3.2 Central fail-closed authorization

`ALUMNI_VERIFICATION`은 동문 승인 이력이고 `WEO_MEMBER.USR_STATUS`는 현재 account eligibility다. 둘을 한 의미로 강제 변환하지 않는다.

```text
protected access = legacy account eligible AND verification approved
```

신규 login/session, refresh, `/api/auth/me`, 모든 protected API가 중앙화된 동일 policy를 사용한다. 기존 access token도 매 요청 DB principal을 다시 읽고 legacy account가 ineligible이면 즉시 차단한다. `CCC → AAA`와 같은 전환에서도 verification 이력은 보존하되 현재 접근은 fail-closed다. `ZZZ` 이탈 시 root role은 제거한다.

### 3.3 Social-link security

```text
SELECT continuation FOR UPDATE
→ exact account password 재인증
→ provider subject owner 검사
→ active social link attach
→ continuation one-time consume
→ COMMIT
```

DB에는 raw continuation token 또는 provider access token을 저장하지 않고 SHA-256 token hash만 저장한다.

---

## 4. Dependency graph와 승인 gate

### 4.1 Task index

| Task | Anchor | Primary outcome |
|---|---|---|
| T00 | [Production 기준선](#t00) | Provenance, push/outbox baseline, test inventory |
| T05 | [Credential source remediation](#t05) | Tracked plaintext 제거·external injection·비노출 검증 |
| T10 | [Backend auth closure](#t10) | Canonical contract와 central authorization |
| T20 | [Migration hardening](#t20) | 036–039 exact-engine evidence |
| T30 | [Maintenance gate](#t30) | All-writer freeze와 controlled smoke |
| T40 | [Android readiness](#t40) | Release AAB 전 code/config gate |
| T50 | [iOS readiness](#t50) | App Store archive 전 code/config gate |
| T60 | [Immutable artifacts](#t60) | Review, approved clean commits, checksums |
| T70 | [Backup/restore](#t70) | Production preflight와 isolated restore |
| T80 | [DB migration](#t80) | 승인된 036→039 production apply |
| T90 | [Backend rollout](#t90) | Separately approved preflight, deploy, smoke, maintenance release |
| T100 | [Android E2E](#t100) | Play internal final acceptance |
| T110 | [iOS E2E](#t110) | TestFlight final acceptance |
| T120 | [24h observation](#t120) | Push/outbox final observation |

### 4.2 Dependency graph

```text
T00 baseline/provenance
 ├─→ T05 credential remediation
 ├─→ T10 central authorization closure
 ├─→ T20 migration hardening
 ├─→ T30 maintenance/write-freeze implementation
 ├─→ T40 Android readiness
 └─→ T50 iOS readiness

T05 + T10 + T20 + T30 + backend full regression + independent review
 → G1B clean backend commit/artifact 생성 승인
 → T60(B) backend artifact/manifest

T40 + Android full regression + independent review
 → G1A clean Android commit/artifact 생성 승인
 → T60(A) Android artifact/manifest

T50 + iOS full regression + independent review
 → G1I clean iOS commit/artifact 생성 승인
 → T60(I) iOS artifact/manifest

T20 + T30 + T60(B) + production read-only preflight
 → T70 backup/isolated restore
 → G2 production DB migration 승인
 → T80 production migration

T80 PASS + T60(B) immutable artifact 재검증
 → G3-P 승인
 → T90 read-only preflight 실행/PASS
 → G3-D 승인
 → T90 backend deployment 실행/PASS
 → G3-S 승인
 → T90 controlled server-side smoke/cleanup 실행/PASS
 → G3-R 승인
 → T90 maintenance release 실행/PASS

T60(A) + T90
 → G4 Android Play 내부 테스트 배포 승인
 → T100 Android E2E

T60(I) + T90
 → G5 iOS TestFlight 배포 승인
 → T110 iOS E2E

T90 release 직후
 → T120 24시간 push/outbox observation

T100 + T110 + T120
 → FinalCompletion
```

---

<a id="t00"></a>
## 5. T00 — Production 기준선 및 테스트 자산

> **Task contract**
> - **Goal/Scope:** Production provenance, auth black-box, push/outbox 지표와 전용 테스트 자산을 read-only로 고정한다.
> - **Explicit exclusions:** DB mutation, test-row 생성, maintenance 진입, deployment를 하지 않는다.
> - **Repository/files:** Backend worktree, production read-only runtime, 아래 `.hermes/evidence` artifacts.
> - **Dependencies:** 없음.
> - **API/DB/platform impact:** 관측만 수행하며 runtime 동작을 변경하지 않는다.
> - **Implementation steps:** Redacted baseline query/log/device inventory를 수집해 timestamp와 provenance를 기록한다.
> - **Test steps:** 동일 read-only query를 재실행해 count 안정성과 secret 미노출을 확인한다.
> - **Acceptance criteria:** Required baseline 전 항목과 테스트 계정 분리 증거가 존재한다.
> - **Read before starting:** §0–§5, §19.

### Scope

- Production black-box push/outbox, auth response와 DB 지표 기준선을 credential 없이 수집한다.
- `pending`, `rejected`, `approved`, `linkRequired` 전용 테스트 계정을 실제 회원 데이터와 분리해 준비한다.
- 테스트 device와 test row 식별 규칙을 정한다.

### Exact files / artifacts

- `.hermes/evidence/kakao-canonical-auth-baseline.md`
- 새 evidence: `.hermes/evidence/kakao-rollout/pre-deploy-baseline.md`
- 새 evidence: `.hermes/evidence/kakao-rollout/test-account-inventory.redacted.md`
- Runtime read-only source: production DB, worker logs, current `/api/auth/kakao/mobile` black-box response

### Required baseline

- Production revision과 `vcs.modified=true` provenance.
- Outbox `PENDING/PROCESSING/SENT/FAILED/DEAD` counts, oldest pending age, retry 분포, duplicate invariant.
- Push worker 실행 상태와 최근 provider error category counts.
- Android/iOS 전용 test device 등록 상태; token 값은 기록하지 않는다.
- 기존 iPhone 관측: HTTP 200이나 top-level `status` 누락으로 canonical decode 실패.

### Stop conditions

- 실제 회원 계정 또는 기존 production row를 테스트에 사용해야만 하는 상태.
- 테스트 데이터가 production 데이터와 구분되지 않는 상태.
- Baseline query나 로그가 credential/device token을 노출하는 상태.

---

<a id="t05"></a>
## 5.1 T05 — Credential source remediation

> **Task contract**
> - **Goal/Scope:** Git 추적 deployment 파일의 평문 credential 후보를 current source에서 제거하고 runtime 값을 external EnvironmentFile에서만 읽는 비노출 경로를 검증한다.
> - **Explicit exclusions:** 값 출력, history rewrite, SMTP 유효성 판단/probe/rotation/revoke, production EnvironmentFile 값·permission·owner 변경, 그 목적의 service restart를 하지 않는다.
> - **Repository/files:** `deploy/alumni-backend.service`, `deploy/alumni-backend.env.example`, `deploy.sh`, `scripts/kakao-auth-rollout/secret-scan.sh`, `scripts/kakao-auth-rollout/secret-scan_test.sh`, `scripts/kakao-auth-rollout/deploy-env-contract_test.sh`.
> - **Dependencies:** T00.
> - **API/DB/platform impact:** Source remediation과 read-only preflight만 수행하며 production runtime mutation은 없다.
> - **Implementation steps:** 값 재출력 없이 tracked/history/artifact scan → source를 external `EnvironmentFile` 주입으로 전환 → `deploy.sh`를 remote-only validation/query로 전환 → SMTP finding은 사용자 소유 deferred risk로 기록한다.
> - **Test steps:** Git history/current tree/build artifact/log secret scan, scanner self-test, deploy contract test, shell syntax/ShellCheck, systemd config validation, production read-only preflight.
> - **Acceptance criteria:** Current source/artifact plaintext 0건, raw 값 출력 0건, external injection/deploy preflight/static tests PASS. SMTP provider-side 상태와 production permission 변경은 명시적 예외이므로 T05를 차단하지 않는다.
> - **Read before starting:** §0–§5.1, §8, §11–§14, §18–§19.

### Gate G0 — Deferred non-SMTP credential remediation only

현재 알려진 SMTP finding은 §2.4의 사용자 소유 deferred risk이며 이 rollout의 G0/G1B blocker가 아니다. Agent는 SMTP 관련 production secret 변경·restart·revoke를 수행하지 않는다. 이후 별도의 non-SMTP active credential 노출이 확인될 때만 값 자체가 아닌 credential 종류, active match 여부, 영향, 교체 순서, rollback과 smoke를 제시하고 G0를 다시 연다.

JWT signing key가 실제 active leaked value로 확인되면 세션 강제 만료 영향과 dual-key transition 가능성을 먼저 증명한다. 증명되지 않으면 fail-closed rotation으로 모든 기존 session을 무효화하고 영향 범위를 G0 승인 자료에 명시한다.

---

<a id="t10"></a>
## 6. T10 — Backend canonical auth와 authorization closure

> **Task contract**
> - **Goal/Scope:** Canonical outcome, session lifecycle, social-link와 중앙 authorization을 TDD로 닫는다.
> - **Explicit exclusions:** Migration 실행, maintenance 운영, mobile channel upload와 production deploy를 하지 않는다.
> - **Repository/files:** Backend worktree와 아래 Exact files.
> - **Dependencies:** T00.
> - **API/DB/platform impact:** Auth response/authorization runtime과 DB principal read policy; production mutation 없음.
> - **Implementation steps:** RED regression을 추가하고 shared eligibility policy로 최소 GREEN 후 refactor한다.
> - **Test steps:** Focused packages, full `go test`, `go vet`, `go build`를 순서대로 실행한다.
> - **Acceptance criteria:** Contract/security regression 전체 PASS 및 sensitive log 0건.
> - **Read before starting:** §0–§3, §5–§6, §18–§19.

### Exact files

- `backend/internal/model/mobile_auth.go`
- `backend/internal/model/auth_principal.go`
- `backend/internal/model/request.go`
- `backend/internal/model/user.go`
- `backend/internal/service/login_eligibility.go`
- `backend/internal/service/auth_jwt.go`
- `backend/internal/service/mobile_auth_service.go`
- `backend/internal/service/auth_service.go`
- `backend/internal/service/auth_session.go`
- `backend/internal/repository/auth_principal_repo.go`
- `backend/internal/repository/refresh_rotation_repo.go`
- `backend/internal/repository/social_link_continuation_repo.go`
- `backend/internal/repository/auth_repo.go`
- `backend/internal/handler/auth_kakao_mobile_handler.go`
- `backend/internal/handler/auth_social_link_handler.go`
- `backend/internal/handler/auth_mobile_response.go`
- `backend/internal/middleware/alumni_approved.go`
- `backend/internal/middleware/auth.go`
- `backend/internal/middleware/admin_auth.go`
- `backend/internal/middleware/auth_revalidation_test.go`
- `backend/internal/middleware/admin_auth_revalidation_test.go`
- `backend/cmd/server/routes.go`
- `.hermes/evidence/kakao-t10-auth-closure.md`

### T10.1 Central eligibility RED

다음 regression을 먼저 실패시킨다.

- `CCC/approved` access token 발급 후 DB에서 legacy status를 ineligible로 변경하면 protected API가 거부된다.
- 동일 상태에서 `/api/auth/me`가 유효 authenticated principal을 반환하지 않는다.
- Refresh가 거부되고 신규 token이 생기지 않는다.
- Verification 이력은 `approved`로 보존된다.
- `ZZZ` 이탈 시 root role이 제거된다.
- JWT cookie, legacy PHP session, mobile JWT 각각에서 DB principal 재조회 없이 접근하는 현재 경로가 RED로 재현된다.
- Admin, optional-auth, 일반 authenticated route별 policy matrix에서 ineligible principal이 fail-closed다.
- `auth_revalidation_test.go`는 JWT cookie, legacy session, mobile access token과 optional-auth를 table-driven으로 검증하고, `admin_auth_revalidation_test.go`는 admin route에서도 legacy eligibility와 current role을 DB에서 다시 읽는지 검증한다.

Test files:

- `backend/internal/middleware/alumni_approved_test.go`
- `backend/internal/service/mobile_access_session_test.go`
- 필요 시 새 `backend/internal/service/login_eligibility_test.go`

### T10.2 Minimal GREEN

- `login_eligibility.go`를 login/refresh/me/protected middleware가 공유하는 단일 account policy로 만든다.
- Middleware가 JWT claim을 authorization source로 사용하지 않고 DB principal의 legacy eligibility와 verification을 함께 확인한다.
- `/api/auth/me`와 refresh도 같은 policy를 사용한다.
- Migration 038의 verification 이력을 인위적으로 rejected로 변경하지 않는다.

### T10.3 Contract/security regression

- Linked Kakao → `authenticated` + nested `session`.
- Unlinked Kakao → `linkRequired` + nested payload.
- Provider email auto-link 금지.
- Unsupported grant/redirect mismatch 거부.
- `/api/auth/kakao/mobile`은 `access_token` grant만 허용하고 `authorization_code`를 포함한 다른 grant는 negative contract test에서 거부한다.
- Android/iOS decoder는 top-level `pending` 또는 `rejected` response를 canonical outcome으로 받아들이지 않고 decode/contract error로 처리한다.
- Pending/rejected principal은 top-level `authenticated`와 nested verification status로만 routing한다.
- Current-session logout과 logout-all 분리.
- Refresh one-time rotation과 consumed-token replay 시 same-SID family revoke.
- Canonical `{linkToken,email,password}`만 허용하고 legacy phone merge payload 거부.
- Sensitive OAuth/provider/token 값 logging 금지.

Commands:

```bash
cd backend
go test ./internal/service ./internal/repository ./internal/handler ./internal/middleware -count=1
go test ./... -count=1
go vet ./...
go build ./...
```

### T10.4 Execution closure — 2026-08-04

T10 execution evidence는 `.hermes/evidence/kakao-t10-auth-closure.md`에 기록한다. Latest post-fix independent review `deleg_9f00b831`은 `passed=true`, logic error 0, Blocker/High test gap 0으로 종료됐다.

Pre-fix review `deleg_cee206a8`이 발견한 서로 다른 continuation token의 MariaDB `ERROR 1213` deadlock과 실패 횟수 누락은 provider+subject별 단일 InnoDB guard row로 remediation했다. MariaDB 10.1.38 exact repository probe 결과는 다음과 같다.

```text
concurrent_mismatches=2
counted=2
deadlocks=0
fifth_attempt=locked
sibling/new tokens=locked
expired_guard=reset
creation=atomic
one_time_consume=preserved
owner_check=preserved
```

Post-fix review의 Low test gaps는 T20 exact-engine harness와 T60 pre-commit regression hardening에 보존한다. 이는 T10 PASS를 차단하지 않지만 누락된 것으로 표현하지 않는다.

T10 PASS는 production rollout GO가 아니다. T10 종료 당시 open이었던 `WEO_MEMBER_SOCIAL=MyISAM` attach/consume 원자성 `B-T00-01`과 migration 037 legacy `Y→ACTIVE` backfill `B-T00-02`는 아래 T20 exact-engine closure에서 해결했다. T10 동안 production DB/service/deployment/credential/commit/channel mutation은 수행하지 않았다.

---

<a id="t20"></a>
## 7. T20 — Migration 036–039 hardening

> **Task contract**
> - **Goal/Scope:** Migration 036–039와 dedicated runner를 MariaDB 10.1.38에서 재현 가능하게 검증한다.
> - **Explicit exclusions:** Production DB apply, dump/restore, backend deploy를 하지 않는다.
> - **Repository/files:** Backend worktree, exact MariaDB fixture와 아래 SQL/scripts.
> - **Dependencies:** T00; T10의 authorization policy는 038 최종 판정 전에 필요하다.
> - **API/DB/platform impact:** Fixture schema/backfill/runner만 변경하며 production DB에는 영향이 없다.
> - **Implementation steps:** Migration별 RED fixture, 최소 SQL 수정, reapply/partial-failure hardening 순으로 수행한다.
> - **Test steps:** 028–035 prerequisite부터 036→039 clean/reapply/failure fixture와 postcheck를 실행한다.
> - **Acceptance criteria:** Migration-specific invariant와 runner safety가 모두 PASS한다.
> - **Read before starting:** §0–§4, §7, §12–§13, §18–§19.

### Exact files

- `backend/migrations/030_create_mobile_refresh_token_table.sql` — prerequisite only
- `backend/migrations/036_extend_mobile_refresh_token_for_rotation.sql`
- `backend/migrations/037_harden_member_social_links.sql`
- `backend/migrations/038_create_auth_principal_tables.sql`
- `backend/migrations/039_create_social_link_continuation.sql`
- `backend/migrations/apply_all.sql`
- `migrate.sh`
- `deploy.sh`
- 새 `scripts/kakao-auth-rollout/preflight.sql`
- 새 `scripts/kakao-auth-rollout/postcheck.sql`
- 새 `scripts/kakao-auth-rollout/apply-migrations.sh`
- 새 `scripts/kakao-auth-rollout/test-migrations.sh`
- 새 `backend/migrations/testdata/kakao_auth_028_035_fixture.sql`
- 새 `backend/migrations/testdata/kakao_auth_edge_cases.sql`
- 새 `backend/migrations/testdata/mariadb-10.1.38.image`
- 새 `backend/migrations/kakao-auth-036-039.sha256`

### T20.1 Exact MariaDB fixture

MariaDB `10.1.38` container/fixture에서 production prerequisite schema `028–035`를 만든 뒤 다음을 검증한다.

- Tag-only image는 금지한다. Registry reference는 `mariadb:10.1.38@sha256:` 뒤에 정확히 64개의 lowercase hexadecimal 문자가 와야 하며, 실제 immutable reference 전체를 `mariadb-10.1.38.image`에 기록해 review/checksum한다.
- Fixture는 production baseline의 migration `028–035`와 그 prerequisite schema에서 생성하며 source commit/file SHA-256을 fixture header에 기록한다.
- `test-migrations.sh`가 container lifecycle, fixture import, migration별 apply/reapply, destructive preflight failure, warning capture와 postcheck를 bounded command로 재현한다.

```text
clean apply 036 → postcheck
clean apply 037 → postcheck
clean apply 038 → postcheck
clean apply 039 → postcheck
전체 reapply → schema/data invariant 유지
037 duplicate preflight failure → 첫 table ALTER 이전 중단
dedicated runner nonzero/partial history → 다음 migration 미실행
```

### T20.2 Migration-specific invariants

#### 036

- `CONSUMED_AT`, `REVOKED_AT`, `ROTATED_TO_JTI`, `IDX_MRT_SID` 존재.
- 기존 `MRT_REVOKED_AT` 값이 `REVOKED_AT`에 보존.
- 기존 column 삭제 없음.
- First rotation, replay, current logout, logout-all regression PASS.

#### 037

- Production 실행 전에 `(USR_SEQ,NMS_GATE)` duplicate 0건.
- Existing `(NMS_GATE,NMS_ID)` conflicting owner 0건.
- Duplicate가 있으면 ALTER 시작 전에 fail-closed.
- `WEO_MEMBER_SOCIAL`을 InnoDB로 전환해 attach/consume rollback 원자성을 확보.
- Non-conflicting legacy `Y` link는 deterministic `ACTIVE`로 정규화.
- Unique index 추가 후 legacy web Kakao path의 예상 오류 mapping 검증.

#### 038

- `ALUMNI_ADMIN_ROLE`, `ALUMNI_VERIFICATION` schema 존재.
- `BAA→rejected`, `BBB→pending`, `CCC/ZZZ→approved`, `ZZZ→root` row counts가 예상치와 일치.
- BAA rejection reason non-null.
- Eligible `WEO_MEMBER`인데 verification row가 없는 수 0건.
- `USR_FN` 20자, `USR_DEPT` 100자 초과 preflight 0건 또는 승인된 deterministic mapping 존재.
- Insert/update trigger 동작과 `ZZZ` 이탈 root 제거 검증.
- Legacy account ineligible 상태의 기존 access는 T10 central policy로 차단.
- Transition matrix를 RED fixture로 고정한다: `CCC→AAA`는 approved history 보존+access 차단, `ZZZ→CCC`는 approved 보존+root 제거, `CCC→BBB`는 pending+root 제거, `CCC→BAA`는 rejected+reason+root 제거, `AAA→CCC`는 approved.
- `INSERT IGNORE` warning 0건과 source/backfill target row-count equality를 별도 assertion으로 검증한다.

#### 039

- Raw token/provider credential column 없음.
- Hash-backed one-time row, expiry, consumed 상태 검증.
- Concurrent completion에서 정확히 하나만 성공.
- Row lock, reauth, owner-check, attach, consume가 한 transaction에서 수행.
- Provider+subject 단일 InnoDB guard가 서로 다른 active/new continuation의 reauthentication 실패를 직렬화하고 정확히 5번째 실패에 잠금.
- Exact-engine 병렬 mismatch에서 실패 누락과 `ERROR 1213`이 0건이며 expired guard reset과 token-insert failure rollback 검증.
- Production `WEO_MEMBER_SOCIAL=MyISAM`이면 attach/consume 원자성을 InnoDB rollback으로 주장하지 않고 T20에서 engine 전환 또는 승인된 recoverable state machine으로 해결.

### T20.3 Runner/aggregate safety

- `apply_all.sql`은 historical `001–035` aggregate로 명시하고 `036–039`를 의도적으로 제외한다.
- `migrate.sh` no-arg 전체 적용을 production rollout에 사용하지 않는다.
- `deploy.sh`의 migration auto-apply prompt/`APPLY_MIGRATIONS=1` 경로는 이번 rollout에서 금지한다.
- `deploy.sh`는 pending migration이 있으면 fail-closed하고 backend를 재시작하지 않아야 한다.
- Dedicated apply script는 checksummed `036–039` manifest와 exact order만 수용하고 각 postcheck PASS 후 history를 기록한다.
- DB password를 command line, process listing, stdout에 노출하지 않는다.

### T20.4 Execution closure — 2026-08-04

T20 evidence는 `.hermes/evidence/kakao-t20-migration-hardening.md`에 기록한다.

```text
MariaDB=10.1.38-MariaDB-1~bionic
clean apply=036→037→038→039 PASS
reapply=036→037→038→039 PASS
postcheck violations=0
037 duplicate fail-before-ALTER=PASS
038 reapply warnings=0
attach+consume rollback=atomic
runner approval/freeze/backup gates=fail-closed
deploy migration auto-apply=removed
```

`B-T00-01`은 사용자 승인에 따라 migration 037에서 `WEO_MEMBER_SOCIAL`을 InnoDB로 전환해 해결했다. `B-T00-02`는 first-ALTER preflight와 deterministic `Y→ACTIVE` backfill로 해결했다. Migration source checksum manifest, non-mutating preflight, 단계별 postcheck/history runner와 production-shape fixture를 read-back 검증했다.

T20은 PASS지만 production migration/dump/restore/deployment 승인으로 해석하지 않는다. Production DB/service/EnvironmentFile/credential/commit/channel mutation은 수행하지 않았고 전체 rollout은 계속 NO-GO다.

---

<a id="t30"></a>
## 8. T30 — Maintenance write-freeze와 controlled smoke gate

> **Task contract**
> - **Goal/Scope:** Controlled smoke를 제외한 Go/PHP/admin/background 모든 DB writer를 fail-closed로 동결한다.
> - **Explicit exclusions:** Production sentinel 활성화, dump, migration, deployment와 smoke 실행을 하지 않는다.
> - **Repository/files:** Backend worktree, Apache/systemd deployment files와 아래 신규 gate/scripts.
> - **Dependencies:** T00.
> - **API/DB/platform impact:** Mutating HTTP와 background job lifecycle; fixture/staging에서만 동작 검증한다.
> - **Implementation steps:** Gate RED tests, backend middleware, Apache block, job pause, idempotent release 순으로 구현한다.
> - **Test steps:** Public bypass, forged header, worker claim, valid smoke, auto-release success/failure를 검증한다.
> - **Acceptance criteria:** Maintenance ON에서 허용된 smoke 외 write 0건, OFF에서 jobs 정상 재개.
> - **Read before starting:** §0–§4, §5, §8, §12, §14, §18–§19.

### Closure status — 2026-08-05

**T30 local/source PASS.** All-writer maintenance gate, canonical `0644 + root-owned` sentinel, generation-bound deployment/migration/smoke release, control-plane curl isolation, release-time runtime rebinding, rollback fail-closed, push uncertain-delivery isolation과 evidence server-only artifact scope를 구현했다.

- Canonical Go/shell regression과 metadata-only current/artifact secret scan PASS
- First closing delta review `deleg_8bccbdf4`: NO-GO, 실제 재현된 2건 remediation
- Bounded remediation review `deleg_71553cff`: PASS, 두 finding 모두 FIXED
- Evidence: `.hermes/evidence/kakao-t30-maintenance-closure.md`
- Manifest: `.hermes/evidence/kakao-t30-closure-manifest.md`
- Checksums: `.hermes/evidence/kakao-t30-closure.sha256`

이 PASS는 local/source closure다. Production upstream traffic freeze, backend/Apache/PHP 및 DB drain, sentinel 활성화, maintenance 진입·해제, migration, backend deployment, controlled smoke와 release 후 worker/job 재개는 각각 후속 approval/production gate에서 검증한다. 따라서 전체 rollout은 계속 **NO-GO**다.

### Production inactive pre-stage gate — 2026-08-05

Production read-only baseline과 old-systemd `EnvironmentFile` compatibility closure 후, 사용자가 inactive pre-stage의 high-level 범위와 frozen exact command를 각각 승인했다. Reviewed packet `2f25bccb27306fb467389a9af0f367c7f776269bb29e8bf592c3884f9972ed8c`로 실행한 attempt 1은 Apache/PHP inactive prepare 후 EnvironmentFile helper의 `state_parent_owner_invalid`에서 중단됐고, single fail-closed driver가 EnvironmentFile → Apache/PHP 순서 rollback과 service/health 검증을 완료했다.

Read-only post-rollback 결과는 `/run/alumni`, sentinel, proof가 absent이고 EnvironmentFile original metadata/SHA 및 maintenance key 0/3, 기존 Apache/PHP checksums, active httpd/backend, syntax, health/legacy `200`이 모두 복구됐음을 확인했다. Random stage는 0개다. Root-owned mode `0700` HTTP backup generation 1개만 trusted backup base에 보존했으며 저장 파일 checksums는 original baseline과 일치한다. Evidence는 `.hermes/evidence/kakao-rollout/maintenance-prestage-attempt-1.redacted.md` (`38355a5a6815ff2e29bca142445f27a632ed9055c5de536bab2448c0d1fbe4b7`)다.

실패 원인은 GNU/Linux에서 BSD `stat -f`를 먼저 실행해 partial stdout과 fallback UID가 결합된 cross-platform helper 결함으로 재현했다. Production helper를 GNU `stat -c` first / BSD fallback으로 수정했다. 첫 retry review `deleg_a77d7812`는 구현 방향과 rollback evidence는 인정했지만 (A) gid/group 회귀 계약·행 단위 production probe evidence, (B) retained backup generation의 정확한 길이 경계와 negative runtime tests가 부족하다고 판정해 `NO-GO`를 유지했다.

후속 remediation은 mode/uid/gid/group 네 helper를 실제 `0640` group branch까지 실행하는 GNU-stat fixture, production read-only 4-helper probe, exact 6-character generation namespace, non-mutating backup-base validator와 base/generation의 symlink·owner·mode·type·short/long/wrong-namespace negative tests를 추가했다. Production retained backup validator는 `trusted`, focused contracts와 ad-hoc changed-path Bash/ShellCheck verification은 PASS했다. 최신 retry packet은 `25b8e37c1b0abfb51a375114211cb075defa2afcb263403dab78ba554dbf025a`로 재동결했다.

최신 bounded review `deleg_0efd6fe1`는 시작·종료 HEAD 및 frozen SHA-256 14/14 일치를 확인하고 A/B/C를 모두 `FIXED`, 최종 `PASS — GO`로 판정했다. 새 command-level 사용자 승인을 받은 뒤 packet `25b8e37c1b0abfb51a375114211cb075defa2afcb263403dab78ba554dbf025a`의 exact driver를 실행했다. Driver는 `transaction=prepared`를 출력했지만 aggregate terminal status가 `1`이라 즉시 성공으로 간주하지 않았고, mutation command를 반복하지 않은 채 별도 production read-only postcheck와 격리 deploy-preflight verifier를 실행했다. 두 검증은 각각 exit `0`으로 sentinel absent, EnvironmentFile exact 3-key/proof binding, approved Apache/PHP checksums와 syntax, active services, health/legacy `200`, stage absent, trusted backups 및 EnvironmentFile PASS 후 expected migration drift/build·upload·restart 미도달을 확인했다.

따라서 P10-B02 EnvironmentFile 3-key와 P10-B03 inactive Apache/PHP gate는 **PREPARED_VERIFIED / 완료**다. Evidence는 `.hermes/evidence/kakao-rollout/maintenance-prestage-attempt-2.redacted.md` (`a13309b51bdb95c15ee618049ee62f715df00977919c6f67b0f9920131efe53b`)다. Production maintenance sentinel은 계속 inactive/absent이며 DB DDL/DML, migration, backend restart/deployment, credential change, controlled write smoke는 수행되지 않았다. 다음 production phase는 자체 exact scope·rollback·approval 전에는 시작하지 않는다.

### Exact files

- `backend/internal/config/config.go`
- 새 `backend/internal/maintenance/gate.go` — shared sentinel state와 constant-time smoke authorization
- 새 `backend/internal/middleware/maintenance_write.go` — Go mutating HTTP 차단
- 새 `backend/internal/middleware/maintenance_write_test.go`
- `backend/cmd/server/routes.go`
- `backend/cmd/server/main.go`
- `backend/cmd/server/wire.go`
- `backend/internal/job/session_cleanup.go`
- `backend/internal/job/email_worker.go`
- `backend/internal/job/visit_aggregation.go`
- `backend/internal/job/push_outbox_worker.go`
- `backend/internal/job/push_outbox_worker_test.go`
- `backend/internal/repository/session_repo.go`
- `backend/internal/repository/password_reset_repo.go`
- `backend/internal/repository/visit_repo.go`
- `backend/internal/repository/auth_repo.go`
- `backend/internal/repository/push_outbox_repo.go`
- `backend/internal/repository/push_outbox_repo_test.go`
- `backend/internal/service/push_notifier.go`
- `backend/internal/service/push_notifier_test.go`
- `backend/internal/service/push_platform_provider.go`
- `backend/internal/service/push_fcm_provider.go`
- `backend/internal/service/push_fcm_provider_test.go`
- `backend/internal/service/push_apns_provider.go`
- `backend/internal/service/push_apns_provider_test.go`
- `deploy/httpd-alumni.conf` — legacy PHP/admin/public mutating request 차단
- `deploy/alumni-backend.service`
- `deploy.sh`
- 새 `scripts/kakao-auth-rollout/enter-maintenance.sh`
- 새 `scripts/kakao-auth-rollout/server-smoke.sh`
- 새 `scripts/kakao-auth-rollout/release-maintenance.sh`
- 새 `scripts/kakao-auth-rollout/writer-zero.sql`

### Required behavior

```text
Maintenance ON:
  health/read-only 요청 허용
  일반 Go API write 차단
  legacy PHP write 차단
  admin write 차단
  session cleanup/email/visit 등 background DB writer pause
  일반 outbox worker pause
  signed/hash-verified operator-controlled smoke write만 허용

Controlled smoke:
  pre-freeze backlog가 0 또는 승인된 stable snapshot임을 확인
  식별 가능한 test outbox만 생성
  smoke용 worker가 test row만 처리
  일반 production row는 변경하지 않음

Maintenance OFF:
  deployment PASS AND full server-side smoke PASS AND 별도 release 승인인 경우에만 해제
  background jobs/worker 재개 확인
```

### Writer inventory and zero-writer protocol

1. Go routes, admin routes, legacy `/old/` PHP source, payment callback, direct operator SQL, cron/systemd timer, `session_cleanup`, `visit_aggregation`, email side effects와 push worker를 writer ledger에 등록한다.
2. 각 writer는 `pause request → acknowledgement → in-flight drain → last-write timestamp 고정` evidence를 남긴다.
3. Legacy PHP production source와 route를 read-only 조사해 GET mutation과 method override를 포함한 실제 write path를 식별한다. HTTP method 차단만으로 완전성이 증명되지 않으면 dump를 시작하지 않는다.
4. DB `PROCESSLIST`, active transaction, application/job acknowledgement와 table write counter를 함께 확인해 controlled smoke 외 writer 0을 증명한다.
5. Direct DB/operator credential은 maintenance 동안 사용을 통제하고 접속 owner를 기록한다. DB 권한/read-only 계층이 필요하면 별도 production preflight evidence와 승인 없이 적용하지 않는다.

### Security constraints

- Raw smoke token은 repository/systemd log/HTTP access log에 남기지 않는다.
- Config에는 token hash만 두고 비교는 constant-time으로 수행한다.
- Public client가 proxy header 또는 loopback 판정을 위조해 bypass할 수 없어야 한다.
- Release operation은 idempotent하다.
- Deployment 또는 smoke 실패, release script 실패 시 sentinel을 유지하고 fail-closed한다.

### RED/GREEN verification

- 모든 mutating method가 maintenance에서 `503` 또는 계약된 maintenance error를 반환.
- Read-only와 `/api/health`는 유지.
- Missing/invalid smoke proof는 거부.
- Valid controlled smoke만 허용.
- Legacy PHP mutating request도 Apache에서 차단.
- Paused worker가 production row를 claim하지 않음.
- 별도 G3-R 승인된 release 후 일반 writes와 jobs가 재개.

---

<a id="t40"></a>
## 9. T40 — Android readiness

> **Task contract**
> - **Goal/Scope:** Android canonical contract, secure session, Kakao/release config를 channel-ready 상태로 만든다.
> - **Explicit exclusions:** Play upload, production E2E, signing/key 값 출력과 backend 변경을 하지 않는다.
> - **Repository/files:** Android worktree와 아래 Exact files.
> - **Dependencies:** T00; wire contract는 §3을 따른다.
> - **API/DB/platform impact:** Android client wire/session/runtime만 변경하며 DB 영향은 없다.
> - **Implementation steps:** Contract/session RED tests, 최소 client/config 수정, emulator/direct-install pre-smoke 순으로 수행한다.
> - **Test steps:** Unit, lint, debug/release build와 Kakao package/signing configuration presence를 검증한다.
> - **Acceptance criteria:** Clean release build 후보와 strict canonical/session restoration regression PASS.
> - **Read before starting:** §0–§4, §5, §9, §15, §18–§19.

### Repository / exact files

- `/Users/jerryhwang/Workspace/03_daeil/.worktrees/dflh-saf-v2-mvp/android`
- `app/build.gradle.kts`
- `app/src/main/AndroidManifest.xml`
- `app/src/main/kotlin/com/dflh/app/DflhApplication.kt`
- `app/src/main/kotlin/com/dflh/app/core/auth/AuthApi.kt`
- `app/src/main/kotlin/com/dflh/app/core/auth/AuthModels.kt`
- `app/src/main/kotlin/com/dflh/app/core/auth/SessionRepository.kt`
- `app/src/main/kotlin/com/dflh/app/core/auth/TokenSessionManager.kt`
- `app/src/main/kotlin/com/dflh/app/feature/auth/KakaoSdkLoginGateway.kt`
- `app/src/main/kotlin/com/dflh/app/feature/auth/KakaoLoginCoordinator.kt`
- `app/src/main/kotlin/com/dflh/app/feature/auth/SessionViewModel.kt`
- `app/src/main/kotlin/com/dflh/app/feature/push/DflhFirebaseMessagingService.kt`
- `app/src/main/kotlin/com/dflh/app/feature/push/PushApi.kt`
- `app/src/main/kotlin/com/dflh/app/feature/push/PushCoordinator.kt`
- `app/src/main/kotlin/com/dflh/app/feature/push/PushDeduplicator.kt`
- `app/src/main/kotlin/com/dflh/app/feature/push/PushTokenStore.kt`
- `app/src/test/kotlin/com/dflh/app/core/auth/CanonicalAuthContractTest.kt`
- `app/src/test/kotlin/com/dflh/app/core/auth/KakaoAuthContractTest.kt`
- `app/src/test/kotlin/com/dflh/app/core/auth/SocialAccountLinkContractTest.kt`
- `app/src/test/kotlin/com/dflh/app/core/auth/SessionRepositoryTest.kt`

### Verification

```bash
./gradlew :app:testDebugUnitTest
./gradlew :app:lintDebug
./gradlew :app:assembleDebug
./gradlew :app:bundleRelease
```

- Native app key 존재 여부만 검증하고 값/BuildConfig/APK content/key hash를 출력하지 않는다.
- Package `com.dflh.saf.v2`와 production release signing key hash가 Kakao Developers console과 일치.
- Release signing은 repository 밖 credential로 구성하고 Play App Signing/upload-key provenance를 값 노출 없이 기록한다.
- `versionCode`/`versionName`이 Play의 기존 release보다 단조 증가하고 AAB output SHA-256이 manifest와 일치한다.
- RED tests는 top-level pending/rejected 거부, nested routing, encrypted restart restore, cold start callback, FCM payload/dedup을 포함한다.
- `SessionRepository` negative tests에서 top-level `pending`/`rejected` fixture가 성공 state로 변환되지 않음을 직접 assertion한다.
- Direct install은 pre-deploy smoke만 인정.
- Final artifact는 승인된 clean commit의 production-signed AAB다.

---

<a id="t50"></a>
## 10. T50 — iOS readiness

> **Task contract**
> - **Goal/Scope:** iOS canonical contract, Keychain restore, Kakao/logging/release entitlement를 channel-ready로 만든다.
> - **Explicit exclusions:** TestFlight upload, production acceptance, credential 출력과 backend 변경을 하지 않는다.
> - **Repository/files:** iOS repository와 아래 Exact files.
> - **Dependencies:** T00; wire contract는 §3을 따른다.
> - **API/DB/platform impact:** iOS client wire/session/runtime만 변경하며 DB 영향은 없다.
> - **Implementation steps:** Wire/security RED tests, 최소 model/repository/state 수정, simulator/physical pre-smoke 순으로 수행한다.
> - **Test steps:** Focused/full tests, Release generic build, entitlement/archive/log inspection을 실행한다.
> - **Acceptance criteria:** App Store archive 후보와 strict canonical/Keychain/logging regression PASS.
> - **Read before starting:** §0–§4, §5, §10, §16, §18–§19.

### Repository / exact files

- `/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2-swift`
- `dflh-saf-v2-swift.xcodeproj`
- `Sources/App/Feature/Login/LoginModels.swift`
- `Sources/App/Feature/Login/AuthRepository.swift`
- `Sources/App/Feature/Login/SocialLoginCoordinator.swift`
- `Sources/App/AppState.swift`
- `Sources/App/Network/APIClient.swift`
- `Sources/Auth/TokenSessionManager.swift`
- `Packages/KakaoAuthKit/Sources/KakaoAuthKit/KakaoAuthKit.swift`
- `Sources/App/Infrastructure/Push/AppDelegate.swift`
- `Sources/App/Infrastructure/Push/PushNotificationCoordinator.swift`
- `Sources/App/Feature/Push/PushDeviceService.swift`
- `Sources/App/Feature/Push/PushPayloadDecoder.swift`
- `Config/DflhSafV2Swift-Release.entitlements`
- `Tests/DflhSafV2SwiftTests/AuthSecurityTests.swift`

### Verification

- Canonical wire shape, strict outcome decoding, social-link `{linkToken,email,password}`, cancellation, verification routing, Keychain restore tests.
- Simulator test/build 후 physical production-config direct install smoke.
- Release entitlement가 APNs production이고 Sign in with Apple/Kakao URL callback이 archive에 존재.
- Kakao SDK Debug stdout에 token/provider identifier가 없는지 actual log capture로 검증.
- Final artifact는 승인된 clean commit의 App Store distribution-signed archive다.

Representative command:

```bash
xcodebuild -project dflh-saf-v2-swift.xcodeproj \
  -scheme DflhSafV2Swift \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  -archivePath build/release/DflhSafV2Swift.xcarchive \
  archive
xcodebuild -exportArchive \
  -archivePath build/release/DflhSafV2Swift.xcarchive \
  -exportPath build/release/export \
  -exportOptionsPlist Config/ExportOptions-AppStore.plist
```

- 새 `Config/ExportOptions-AppStore.plist`는 App Store distribution export만 담당하며 signing secret을 포함하지 않는다.
- `MARKETING_VERSION`/`CURRENT_PROJECT_VERSION`이 App Store Connect의 기존 build보다 단조 증가해야 한다.
- RED tests는 top-level pending/rejected 거부, nested routing, Keychain restart restore, cold-start Kakao callback, APNs payload handling을 포함한다.
- `AuthSecurityTests` negative decoder fixture에서 top-level `pending`/`rejected`가 정식 outcome으로 남지 않음을 직접 assertion한다.
- `.xcarchive`와 exported IPA의 SHA-256, signing team/bundle/entitlement provenance를 manifest에 기록한다.

### Executable iOS TDD and regression gate

검증 scheme은 `DflhSafV2Swift`, test target은 `DflhSafV2SwiftTests`, simulator destination은 현재 검증된 `platform=iOS Simulator,name=Codex iPhone 15,OS=26.0`으로 고정한다. Destination이 없으면 임의 기기로 대체하지 않고 동일 runtime의 전용 simulator를 생성·기록한 뒤 plan evidence를 갱신한다.

```bash
cd /Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2-swift
RUN_ID="$(date -u +%Y%m%dT%H%M%SZ)"
RESULT_DIR="build/TestResults/${RUN_ID}"
mkdir -p "${RESULT_DIR}"

# RED: 새 contract/security test를 먼저 추가한 상태에서 non-zero와 expected assertion failure를 확인한다.
xcodebuild test \
  -project dflh-saf-v2-swift.xcodeproj \
  -scheme DflhSafV2Swift \
  -destination 'platform=iOS Simulator,name=Codex iPhone 15,OS=26.0' \
  -only-testing:DflhSafV2SwiftTests/SocialAuthResponseTests \
  -only-testing:DflhSafV2SwiftTests/SocialLinkFormStateTests \
  -only-testing:DflhSafV2SwiftTests/AuthenticationDestinationTests \
  -only-testing:DflhSafV2SwiftTests/TokenSessionManagerTests \
  -only-testing:DflhSafV2SwiftTests/AuthBootstrapCoordinatorTests \
  -resultBundlePath "${RESULT_DIR}/AuthContractSecurity-RED.xcresult"

# 최소 GREEN: 동일 focused suite가 exit 0과 TEST SUCCEEDED를 반환해야 한다.
xcodebuild test \
  -project dflh-saf-v2-swift.xcodeproj \
  -scheme DflhSafV2Swift \
  -destination 'platform=iOS Simulator,name=Codex iPhone 15,OS=26.0' \
  -only-testing:DflhSafV2SwiftTests/SocialAuthResponseTests \
  -only-testing:DflhSafV2SwiftTests/SocialLinkFormStateTests \
  -only-testing:DflhSafV2SwiftTests/AuthenticationDestinationTests \
  -only-testing:DflhSafV2SwiftTests/TokenSessionManagerTests \
  -only-testing:DflhSafV2SwiftTests/AuthBootstrapCoordinatorTests \
  -resultBundlePath "${RESULT_DIR}/AuthContractSecurity-GREEN.xcresult"

# Package/full regression: 전체 DflhSafV2SwiftTests target을 실행한다.
xcodebuild test \
  -project dflh-saf-v2-swift.xcodeproj \
  -scheme DflhSafV2Swift \
  -destination 'platform=iOS Simulator,name=Codex iPhone 15,OS=26.0' \
  -only-testing:DflhSafV2SwiftTests \
  -resultBundlePath "${RESULT_DIR}/DflhSafV2SwiftTests-FULL.xcresult"
```

- RED evidence는 command, non-zero exit, 새 assertion의 실패명만 기록하고 token/payload 원문을 기록하지 않는다.
- GREEN/full regression은 exit `0`, log의 `** TEST SUCCEEDED **`, `.xcresult` 존재와 failure count `0`을 모두 요구한다.
- 이미 같은 이름의 result bundle이 있으면 덮어쓰지 말고 삭제가 아닌 새 rollout run ID 경로를 사용한다.
- 순서는 `RED → 최소 GREEN → focused/package regression → full regression → REFACTOR → 동일 full regression 재실행 → archive/export`이며 어느 단계도 건너뛰지 않는다.

---

<a id="t60"></a>
## 11. T60 — Clean commits, independent review, immutable artifacts

> **Task contract**
> - **Goal/Scope:** Verified source를 독립 review하고 사용자 승인 후 clean commit과 immutable artifact/manifest로 고정한다.
> - **Explicit exclusions:** 해당 G1B/G1A/G1I 전 commit, push/amend, DB mutation, deployment와 channel upload를 하지 않는다.
> - **Repository/files:** Backend/Android/iOS repositories, backend의 checksum-bound local replacement module, deployment manifest artifact.
> - **Dependencies:** T60(B)는 T05/T10/T20/T30+G1B, T60(A)는 T40+G1A, T60(I)는 T50+G1I에 각각 독립 의존한다.
> - **API/DB/platform impact:** Source provenance와 build artifact만 고정하며 runtime 영향은 없다.
> - **Implementation steps:** Closing review, finding closure, 해당 G1B/G1A/G1I 승인, clean commit/build/checksum 순으로 수행한다.
> - **Test steps:** Clean status, commit SHA, local replacement exact inventory, effective build environment, reproducible build와 manifest checksum을 독립 재검증한다.
> - **Acceptance criteria:** Blocker/High 0건과 manifest-actual checksum 일치.
> - **Read before starting:** §0–§4, §6–§11, §18–§19.

### Gate G1B / G1A / G1I

- **G1B:** T05/T10/T20/T30과 backend full regression/review 후 backend+migration clean commit/artifact 승인을 받는다. T70은 G1B만 의존한다.
- **G1A:** T40 Android regression/review 후 Android clean commit/AAB 생성 승인을 받는다. G4는 G1A artifact를 요구한다.
- **G1I:** T50 iOS regression/review 후 iOS clean commit/archive 생성 승인을 받는다. G5는 G1I artifact를 요구한다.

각 gate 승인 전 해당 repository를 commit하지 않는다. 한 플랫폼의 artifact 지연이 G1B와 DB preflight를 불필요하게 차단하지 않도록 범위를 분리한다.

### Independent review

- Backend authorization/session/social-link/migration high-risk review.
- Android canonical contract/session persistence/release configuration review.
- iOS canonical contract/Keychain/release entitlement/logging review.
- Findings가 Blocker/High이면 수정 후 review를 다시 수행한다.

### Deployment manifest

새 artifact: `.hermes/evidence/kakao-rollout/deployment-manifest.md`

필수 필드:

```text
repository + branch + clean commit SHA
source status
production baseline cd9f8f0807c0292091a127e2fbc68ff99e6a2e1d + vcs.modified=true provenance limitation
Go/Java/Kotlin/Gradle/Xcode/Swift toolchain
effective GOWORK, GOFLAGS, GOTOOLCHAIN, selected Go version
backend local replacement module path + manifest SHA-256 + exact file SHA-256
exact backend build command and target environment
backend binary SHA-256
Android AAB SHA-256
iOS archive/export SHA-256
migration manifest: 036,037,038,039 exact order + SHA-256
test commands + exit codes
independent review verdict
build timestamp and target architecture
```

Manifest와 실제 업로드 artifact checksum이 다르면 중단한다.

### T60(B) external local replacement provenance

`backend/go.mod`의 다음 local replacement는 backend Git commit 밖의 실제 build input이다.

```text
replace github.com/dflh-saf/social-auth => ../../dflh-social-auth
4013f30cb6f0a253a81cad25b151cb6a52b6c8d788ce2fca55382ddb86e138f5  dflh-social-auth/go.mod
1608b62050c2f016368775a94020dd1a3152889874ca2ce034871b4c2730d9be  dflh-social-auth/kakao/client.go
f7bd19fc61b255cb58d4dfbaeffee86c99971c5c864a48a5517b950c59543573  .hermes/evidence/kakao-rollout/g1b-external-build-input.sha256
```

G1B artifact는 새 disposable bundle 안의 `bundle/web` clean worktree와 `bundle/dflh-social-auth` checksum-bound copy를 함께 사용한다. `bundle/web/backend`에서 `../../dflh-social-auth`가 정확히 bundle 내부 sibling을 가리키는지 확인한다. Destination inventory는 `go.mod`, `kakao/`, `kakao/client.go`만 허용하며 두 파일은 regular non-symlink여야 한다. 예상 밖 filesystem entry, symlink, type 또는 checksum 불일치는 중단 조건이다.

원본 외부 directory를 검증한 뒤 두 파일만 복사하고, build 직전과 직후에 destination의 exact inventory와 checksum을 다시 확인한다. Build는 `GOWORK=off`, empty `GOFLAGS`, `GOTOOLCHAIN=local`, `CGO_ENABLED=0`, `GOOS=linux`, `GOARCH=amd64`를 명시한다. Manifest에는 selected `go version`, effective `go env`, exact command, `go version -m` read-back과 binary SHA-256를 기록한다. Disposable bundle은 artifact read-back 후 전체 제거한다.

---

<a id="t70"></a>
## 12. T70 — Production preflight, backup, isolated restore

> **Task contract**
> - **Goal/Scope:** 모든 writer를 동결한 뒤 full dump의 실제 복원 가능성을 production 외부에서 증명한다.
> - **Explicit exclusions:** Production migration/backend deploy와 production DB restore를 하지 않는다.
> - **Repository/files:** Approved artifacts, production read-only/runtime controls, repository 밖 backup storage, 새 `scripts/kakao-auth-rollout/dispose-backup.sh`.
> - **Dependencies:** T20, T30, T60(B).
> - **API/DB/platform impact:** Maintenance write block을 활성화하고 production은 read-only; restore는 isolated DB만 대상이다.
> - **Implementation steps:** Preflight, backlog drain, freeze, dump, checksum/off-host copy, isolated restore 순으로 수행한다.
> - **Test steps:** Writer 0, import exit 0, schema/object/count/relationship fingerprint를 대조한다.
> - **Acceptance criteria:** Backup/restore evidence PASS와 G2 입력 자료 완결.
> - **Read before starting:** §0–§5, §7–§8, §11–§14, §18–§19.

### Preconditions

- G1B PASS와 immutable backend/migration artifact 존재.
- T30 maintenance gate가 staging/exact fixture에서 검증됨.
- Production read-only preflight가 MariaDB `10.1.38`, schema prerequisites, table engines, sizes, disk space, duplicate/length/count invariant를 기록.
- Existing outbox backlog와 jobs를 확인하고 가능한 범위에서 drain.

### Production read-only preflight status — 2026-08-05

사용자가 첫 production 단계인 read-only preflight만 승인했고, SSH/runtime metadata, aggregate-only DB `SELECT`, remote-reduced journal count와 body-free health/read-only HTTP status를 관찰했다. Production DDL/DML, file/service mutation, dump, maintenance, migration 또는 deployment는 수행하지 않았다.

```text
MariaDB=10.1.38-MariaDB
applied_migrations=028-035
exact_preflight_metrics=6/6 zero
social/provider duplicate groups=0
device token duplicate groups=0
outbox pending/processing/failed/stuck=0
outbox delivery-state-uncertain=0
push/outbox repeated aggregate snapshot=stable
```

P10-B01 old-systemd compatibility는 bounded review `deleg_605c3ce3` PASS로, P10-B02/B03 inactive EnvironmentFile·Apache/PHP pre-stage는 attempt 2 independent read-back과 evidence `a13309b51bdb95c15ee618049ee62f715df00977919c6f67b0f9920131efe53b`로 **PREPARED_VERIFIED** 상태다. Sentinel은 absent이므로 production은 정상 서비스 상태이며 writer freeze는 아직 활성화되지 않았다. T70 진입은 T60(B) immutable backend/migration artifact와 별도 maintenance-entry command·rollback·production 승인을 선행조건으로 유지한다.

Evidence: `.hermes/evidence/kakao-rollout/production-preflight.redacted.md`

### Maintenance entry

1. Apache/Go/legacy/admin write block sentinel 활성화.
2. Background jobs와 일반 outbox worker pause 확인.
3. Controlled smoke 이외 DB writer 0임을 process/log/DB 지표로 확인.
4. 이 상태가 아니면 dump를 시작하지 않는다.

### Full logical dump

포함 범위:

```text
schema + data + triggers + routines + events + migration history
```

- Table engine을 먼저 확인한다. `--single-transaction` 일관성은 InnoDB에만 의존하므로 non-InnoDB가 있으면 write-freeze와 적절한 lock 전략을 사용한다.
- Dump는 repository 밖에 저장하고 최소 권한, checksum, size, timestamp를 기록한다.
- 동일 서버 장애에도 대비해 암호화된 off-host 사본을 만든다.
- Dump 내용과 credential은 출력하지 않는다.

### Retention, key custody, and disposal

- Full dump와 isolated restore 임시 DB는 최종 acceptance 및 T120 PASS 후 30일간 보존한다.
- 미해결 rollout incident/정합성 조사가 있으면 incident 종료 후 30일까지 자동 연장한다.
- 암호화 키는 dump와 분리된 production secret 관리 경로에 저장하고 사용자와 지정 restore operator만 접근한다.
- 접근, restore, 복사, 폐기 event를 redacted evidence로 기록한다.
- 보존기간 종료 후 `dispose-backup.sh`가 local/off-host dump와 임시 restore DB의 폐기를 확인하고 재조회 불가 postcheck를 수행한다.
- 폐기 후에는 dump data 없이 SHA-256, 크기, 생성·폐기 시각, restore 결과와 redacted manifest만 유지한다.

### Isolated restore

Production DB가 아닌 격리 DB에 restore하고 다음을 검증한다.

- Import exit 0.
- 핵심 table/trigger/index/routine/event 존재.
- Production과 table별 row count 일치.
- Auth principal/social identity/refresh/outbox representative 관계 일치.
- `036–039` 적용 전 schema fingerprint 일치.

Restore rehearsal가 실패하면 G2로 진행하지 않는다.

### Gate G2 — Production DB migration approval

Preflight, dump checksum, off-host copy, isolated restore, migration checksum, 예상 lock/row counts, stop/restore 판단표를 사용자에게 제시하고 별도 승인을 받는다.

---

<a id="t80"></a>
## 13. T80 — Production migration 036–039

> **Task contract**
> - **Goal/Scope:** G2 승인된 checksum의 036→039만 production에 순차 적용하고 매 단계 invariant를 확인한다.
> - **Explicit exclusions:** Backend deployment, mobile release, automatic restore와 승인되지 않은 SQL을 실행하지 않는다.
> - **Repository/files:** T20 migration/runner artifacts와 production DB.
> - **Dependencies:** T70 PASS와 G2 명시 승인.
> - **API/DB/platform impact:** Production auth schema/backfill 변경; maintenance write freeze는 유지한다.
> - **Implementation steps:** Checksum 확인 후 migration 하나씩 apply/postcheck/history 기록한다.
> - **Test steps:** 단계별 schema/backfill/trigger/replay/duplicate/push-outbox invariants를 실행한다.
> - **Acceptance criteria:** 036–039 postcheck와 전체 integrity PASS; partial state 0건.
> - **Read before starting:** §0–§4, §7, §11–§13, §18–§19.

Production에서는 `apply_all.sql`, no-arg `migrate.sh`, `deploy.sh APPLY_MIGRATIONS=1`을 사용하지 않는다.

```text
036 apply → 036 postcheck PASS
037 apply → 037 postcheck PASS
038 apply → 038 postcheck PASS
039 apply → 039 postcheck PASS
```

각 단계에서:

- 실행 파일 checksum을 manifest와 재대조.
- Start/end timestamp, exit code, warning과 row count를 credential 없이 기록.
- `_migration_history`는 SQL과 postcheck가 모두 PASS한 뒤에만 기록.
- 실패하면 다음 migration을 실행하지 않고 partial DDL state를 조사.
- DDL 자동 rollback을 가정하지 않는다.

전체 migration 후:

- Backfill counts와 missing principal 0건.
- Social duplicate/conflicting owner 0건.
- Trigger/inserts/status transition 검증.
- Refresh schema와 legacy/canonical revoke coherence.
- Continuation raw credential absence.
- Existing push/outbox schema/data fingerprint 무변경.

Migration PASS만으로 backend를 배포하지 않는다.

---

<a id="t90"></a>
## 14. T90 — Separately approved backend deployment, server-side smoke, and release

> **Task contract**
> - **Goal/Scope:** 단계별 별도 승인으로 immutable backend를 배포하고 full smoke를 검증한 뒤, 별도 release 승인 후에만 maintenance를 해제한다.
> - **Explicit exclusions:** Android/iOS channel upload, production dump restore, unapproved artifact deploy를 하지 않는다.
> - **Repository/files:** Backend artifact, deploy/systemd/Apache/smoke/release scripts와 production runtime.
> - **Dependencies:** T80 PASS, T60(B) checksum revalidation, G3-P/G3-D/G3-S/G3-R 각각의 명시 승인과 선행 단계 PASS.
> - **API/DB/platform impact:** Production backend/runtime 전환과 controlled test writes; 일반 writes는 smoke 종료까지 차단한다.
> - **Implementation steps:** Read-only preflight, rollback artifact, atomic deploy, readiness, full smoke, AND-gated release를 순서대로 수행하되 각 단계 시작 전에 별도 exact-command 승인을 받는다.
> - **Test steps:** Canonical/authz/social-link/refresh/logout/outbox/log scan과 release idempotency를 검증한다.
> - **Acceptance criteria:** G3-P/D/S/R 네 approval record, read-only preflight PASS evidence, deployment+smoke+cleanup+release postcheck PASS, jobs 재개, T120 timer 시작 evidence.
> - **Read before starting:** §0–§4, §6–§8, §11–§14, §17–§19.

### Gate G3 — Four separate production approvals

- **G3-P read-only preflight:** local authority/artifact checksums, reviewed closure SHA-256, 64-hex attempt nonce와 production read-only state query의 exact command를 승인받는다. PASS evidence에는 approved closure, maintenance generation, current backend SHA-256/size와 consumed attempt ID를 기록한다. Production mutation은 금지한다.
- **G3-D backend deployment:** 같은 closure에 결속된 G3-P PASS evidence SHA-256, 동일 maintenance generation, immutable binary checksum, preflight-observed backend SHA-256/size, 64-hex attempt nonce, rollback command와 exact deployment command를 제시하고 별도 승인을 받는다. Build window 이후 rollback copy의 hash/size와 maintenance generation을 final replacement 전에 재검증하며 drift면 replacement/restart 없이 fail closed한다.
- **G3-S controlled smoke:** G3-D PASS 뒤 test run ID, 허용된 writer/proof 범위, exact smoke/cleanup commands와 실패 시 maintenance 유지 조건을 제시하고 별도 승인을 받는다.
- **G3-R maintenance release:** G3-D와 G3-S가 모두 PASS한 뒤 release command, generation binding, worker/job 재개 postcheck를 제시하고 별도 승인을 받는다.

어느 단계의 PASS도 다음 단계 승인이 아니다. 각 approval bit는 해당 exact command와 reviewed evidence에만 유효하다.

각 G3-P/G3-D/G3-S/G3-R approval bit는 single-use이며 reviewed evidence SHA-256과 exact command bytes에 결속된다. Approval은 첫 execution attempt 시작과 동시에 성공/실패와 무관하게 consumed된다. 같은 stage retry 또는 whole-T90 replay에서는 re-executed stage마다 갱신된 evidence를 검토하고 새 applicable approval을 받아야 한다. 각 stage launcher는 approved command에 포함된 64-hex attempt nonce로 canonical payload를 derive하고 trusted local `0600` no-overwrite consumed record를 첫 attempt 전에 생성하며, 동일 record가 이미 있으면 fail closed한다. T90-D1 launcher는 G3-P/G3-D에 이를 강제하고, G3-S/G3-R launcher도 해당 stage 승인 요청 전에 같은 contract를 구현·검증해야 한다.

### Deployment

- 기존 binary를 checksum과 함께 rollback artifact로 보존.
- T60(B)와 같은 clean disposable bundle을 재구성하고 external module의 exact inventory/type/checksum을 build 전후 재검증한다.
- `GOWORK=off`, empty `GOFLAGS`, `GOTOOLCHAIN=local`과 manifest에 기록된 selected Go version이 일치해야 한다.
- `RECORD_MAINTENANCE_DEPLOY_EVIDENCE=1`, `APPROVED_SOURCE_REVISION`, `APPROVED_BACKEND_ARTIFACT_SHA256`를 모두 명시한 backend-only `deploy.sh`만 허용한다.
- `deploy.sh`가 exact source/input으로 재빌드한 binary SHA-256가 T60(B) approved digest와 일치할 때만 업로드한다. 불일치 또는 approval 변수 누락은 upload 전 fail-closed한다.
- Atomic replacement 후 `systemctl restart alumni-backend`.
- Apache config test가 PASS하기 전 reload하지 않는다.
- Deploy script가 migrations/frontend/legacy shim을 암묵적으로 변경하지 않도록 target 범위를 제한한다.

### Controlled server-side smoke

일반 production write는 계속 차단하고 지정된 테스트 계정/operator proof만 허용한다.

필수 PASS:

- `/api/health`와 process readiness.
- Password/mobile/Kakao canonical `authenticated` structure.
- Unlinked identity의 `linkRequired` structure.
- `pending/rejected/approved` 보호 API authorization.
- Legacy-ineligible + approved history가 protected API와 `/api/auth/me`에서 fail-closed.
- Social-link reauth, owner-check, attach, one-time consume와 replay 거부.
- Refresh first rotation, replay family revoke, logout/logout-all.
- iOS/APNs: test outbox 1건 enqueue, worker claim, `SENT` 또는 계약된 terminal state, duplicate 0건.
- Android/FCM: direct provider send 결과, device receipt, duplicate 0건을 outbox와 별도 검증한다. 현재 topology에서 FCM row가 iOS outbox에 생길 것으로 기대하지 않는다.
- FCM transient failure에 durable retry가 실제 code/test로 증명되지 않으면 수렴했다고 간주하지 않고 smoke/observation 실패로 처리한다.
- Existing production outbox row 무변경.
- Sensitive log scan PASS.

### Controlled test-row lifecycle

- 모든 smoke row는 non-secret rollout run ID로 session, continuation, social-link attempt, outbox, push registration inventory에 연결한다.
- 각 단계 전후에 실제 production 회원/social-link/push row와 구분되는 소유권 predicate를 검증한다.
- Smoke 성공 또는 실패 후 해당 run ID로 생성한 row만 정리하고 삭제/consume/revoke count를 evidence에 기록한다.
- Existing member/social-link/device/outbox row는 cleanup 대상이 아니다.
- 정리 postcheck에서 unconsumed continuation, active smoke session, test device registration, nonterminal test outbox가 0건이어야 한다. 보존이 필요한 audit row는 test marker와 retention 근거를 남긴다.

### Separately approved maintenance release

다음 AND와 별도 G3-R exact-command 승인이 모두 참일 때만 `release-maintenance.sh`를 실행한다.

```text
backend deployment PASS
AND full server-side smoke PASS
```

Release script는 sentinel을 제거하고 일반 background jobs/worker를 재개하며 idempotent postcheck를 수행한다. Release 실패 시 write block을 유지하고 성공으로 보고하지 않는다.

Maintenance 해제 즉시 T120의 24시간 timer를 시작한다.

---

<a id="t100"></a>
## 15. T100 — Android Play internal E2E

> **Task contract**
> - **Goal/Scope:** G4 승인된 AAB를 Play 내부 테스트로 설치해 production Android E2E를 완료한다.
> - **Explicit exclusions:** 직접 설치 build를 final acceptance로 인정하지 않고 backend/iOS를 변경하지 않는다.
> - **Repository/files:** Approved Android AAB/manifest, Play Console, 실제 Android device.
> - **Dependencies:** T40, T60(A) Android artifact, T90 PASS, G4 명시 승인.
> - **API/DB/platform impact:** Play test release와 식별 가능한 production test rows/push만 생성한다.
> - **Implementation steps:** Checksum/signing preflight, upload, Play install, state별 E2E와 cleanup 순으로 수행한다.
> - **Test steps:** Kakao callback, canonical routing, authz, encrypted restart restore, FCM actual receipt를 검증한다.
> - **Acceptance criteria:** 아래 Final acceptance 전 항목 PASS와 duplicate 0건.
> - **Read before starting:** §0–§5, §9, §11, §14–§15, §17–§19.

### Gate G4

AAB commit SHA/checksum, Play package/signing/Kakao key-hash preflight, production config와 test account 목록을 제시하고 Play Console 내부 테스트 배포 승인을 받는다.

### Final acceptance

실제 Android device에 Play 내부 테스트 경로로 설치하고 다음을 모두 검증한다.

- Kakao app/web handoff와 callback.
- Linked → `authenticated`, unlinked → `linkRequired`.
- Canonical social link completion.
- Pending/rejected/approved routing.
- Approved의 alumni/messages/push 접근.
- 비승인/ineligible principal의 protected API 차단.
- Encrypted secure session 저장, process kill, 재실행 복원.
- Refresh, current logout, logout-all.
- 실제 FCM production push 수신과 duplicate 없음.

Platform-local 실패면 backend를 유지하고 Android release만 차단·수정한다. Shared backend/security/data 문제면 write를 재차단하고 T90 rollback/forward-fix 판단으로 전환한다.

---

<a id="t110"></a>
## 16. T110 — iOS TestFlight E2E

> **Task contract**
> - **Goal/Scope:** G5 승인된 archive를 TestFlight로 설치해 production iOS E2E를 완료한다.
> - **Explicit exclusions:** 직접 설치 build를 final acceptance로 인정하지 않고 backend/Android를 변경하지 않는다.
> - **Repository/files:** Approved iOS archive/manifest, App Store Connect/TestFlight, 실제 iPhone.
> - **Dependencies:** T50, T60(I) iOS artifact, T90 PASS, G5 명시 승인.
> - **API/DB/platform impact:** TestFlight release와 식별 가능한 production test rows/push만 생성한다.
> - **Implementation steps:** Checksum/signing/entitlement preflight, upload, TestFlight install, E2E와 cleanup 순으로 수행한다.
> - **Test steps:** Kakao callback, canonical routing, authz, Keychain restart restore, APNs actual receipt/log safety를 검증한다.
> - **Acceptance criteria:** 아래 Final acceptance 전 항목 PASS와 secret/duplicate 0건.
> - **Read before starting:** §0–§5, §10–§11, §14, §16–§19.

### Gate G5

Archive/export checksum, bundle/signing/entitlement/Kakao callback preflight, production config와 test account 목록을 제시하고 TestFlight 업로드·배포 승인을 받는다.

### Final acceptance

실제 iPhone에 TestFlight로 설치하고 T100과 동일한 canonical/authorization 범위를 검증한다. 추가 필수 항목:

- App Store distribution signing.
- APNs production entitlement.
- Keychain secure session 저장과 process termination/relaunch 복원.
- 실제 APNs production push 수신.
- Kakao SDK/runtime log에 provider token·identifier 노출 없음.

Platform-local 실패면 backend를 유지하고 iOS release만 차단·수정한다. Shared 문제면 T90 대응으로 전환한다.

---

<a id="t120"></a>
## 17. T120 — 24시간 push/outbox observation

> **Task contract**
> - **Goal/Scope:** Maintenance 해제 직후부터 push/outbox를 연속 관찰해 rollout 무결성을 최종 판정한다.
> - **Explicit exclusions:** Rollout 비기인 provider 일시 오류를 임의로 실패 처리하거나 실제 회원 row를 정리하지 않는다.
> - **Repository/files:** Production DB/logs/worker metrics와 Android/iOS actual receipt evidence.
> - **Dependencies:** T90 release로 timer 시작; 최종 완료는 T100과 T110도 필요하다.
> - **API/DB/platform impact:** Read-only monitoring과 승인된 test push만 수행한다.
> - **Implementation steps:** Baseline과 동일 지표를 지속 수집하고 이상 발생 시 reset rule을 실행한다.
> - **Test steps:** Count/age/duplicate/liveness/provider convergence와 actual device receipt를 대조한다.
> - **Acceptance criteria:** 연속 24시간 rollout 기인 failure 0건 및 T100/T110 PASS.
> - **Read before starting:** §0–§5, §14–§19.

시작 시점은 별도 승인된 maintenance 해제 직후다. Android/iOS channel E2E와 병행할 수 있지만 둘 중 하나라도 남아 있으면 최종 완료가 아니다.

### 관찰 항목

- iOS/APNs durable outbox의 status counts와 oldest pending age.
- Stuck `PROCESSING`.
- 예기치 않은 `DEAD` 증가.
- Retry 수렴과 무한 retry 여부.
- APNs event/device duplicate row와 duplicate delivery.
- Android direct-FCM send attempts/success/error category/provider liveness와 duplicate delivery.
- Worker liveness/restart/error.
- Android FCM 및 iOS APNs 실제 수신.

### 허용치

Rollout 기인 유실, 중복, stuck, worker 중단, 구조적 mobile push 실패는 모두 0건이다. APNs/FCM 일시 오류는 기존 retry가 제한 시간 내 중복 없이 수렴하고 rollout 비기인임이 확인될 때만 허용한다.

### Reset rule

Rollout 기인 이상이 한 건이라도 확인되면:

```text
→ write 재차단
→ 영향/정합성 확인
→ 수정 또는 backend rollback
→ 갱신된 evidence와 exact smoke/cleanup commands로 새 G3-S 승인
→ full server-side smoke 재실행
→ PASS 후 갱신된 evidence와 exact release command로 새 G3-R 승인
→ maintenance 해제
→ 24시간 timer를 0시간부터 재시작
```

---

## 18. Rollback / forward-fix matrix

| Failure | Immediate action | DB policy | Resume condition |
|---|---|---|---|
| Preflight/backup/restore 실패 | 중단, write block 유지 또는 안전하게 원복 | Mutation 없음 | 원인 수정 후 T70 재실행 |
| Read-only preflight 실패 | Production mutation 없이 중단; consumed G3-P 재사용 금지 | Mutation 없음 | 갱신된 closure/state evidence와 exact command로 새 G3-P 승인 |
| 036–039 단계 실패 | 다음 migration 금지, partial schema 조사 | DDL 자동 rollback 가정 금지, 우선 forward-fix | 해당 단계 postcheck PASS |
| Backfill/data integrity 손상 | write block 유지 | 안전한 forward-fix 불가 시에만 별도 승인 후 full dump restore | Restore/정합성 재검증 |
| Backend upload/restart 실패 | 이전 immutable binary 복구; consumed G3-D 재사용 및 새 G3-D 승인 전 deployment retry 금지 | Additive DB schema 유지 | 갱신된 preflight/deployment evidence와 exact command로 새 G3-D 승인; 이전 health PASS 또는 수정 artifact 승인 |
| Server-side smoke 실패 | release 금지; 갱신된 evidence와 exact commands로 새 G3-S 승인 전 재실행 금지 | 실패 유형별 forward-fix; data damage면 restore 후보 | 새 G3-S 승인으로 Full smoke/cleanup PASS 후 별도 G3-R 승인 |
| Release script 실패 | fail-closed 유지; 갱신된 failure evidence와 exact command로 새 G3-R 승인 전 retry 금지 | DB 변경 없음 | 새 G3-R 승인으로 idempotent release postcheck PASS |
| Android-only 실패 | Android channel만 차단 | Backend/DB 유지 | Android 재검증 PASS |
| iOS-only 실패 | iOS channel만 차단 | Backend/DB 유지 | iOS 재검증 PASS |
| Shared contract/security 문제 | write 재차단 | Backend rollback 또는 forward-fix; DB restore는 별도 승인 | T90 전체 재실행 시 G3-P부터 re-executed stage별 새 approval |
| 24h rollout 기인 outbox 이상 | write 재차단 | 영향별 rollback/forward-fix | Smoke + 새 24h PASS |

### Gate GR — Destructive production restore approval

Full dump restore가 실제로 필요하면 operator는 영향 row/time range, dump SHA-256, backup 이후 write 유실 분석, forward-fix 불가 근거, restore command, restore operator와 post-restore verification을 제시해 별도 사용자 승인을 받는다. GR 승인 없이 production restore를 실행하지 않는다. Restore 후 기존 T80/T90 evidence는 무효화하고 T70 isolated verification 및 G2부터 재진입한다.

Full dump restore는 최후 수단이다. Backup 이후 write-freeze가 깨졌거나 controlled test 외 production write가 존재하면 restore 전에 유실 범위를 다시 산정하고 별도 명시 승인을 받는다.

---

## 19. Evidence와 보안

모든 evidence는 `.hermes/evidence/kakao-rollout/` 아래에 저장하되 secret을 포함하지 않는다.

```text
pre-deploy-baseline.md
migration-fixture-validation.md
independent-review.md
deployment-manifest.md
production-preflight.redacted.md
backup-restore-validation.redacted.md
backup-disposal.redacted.md
migration-execution.redacted.md
server-smoke.redacted.md
android-play-e2e.redacted.md
ios-testflight-e2e.redacted.md
push-outbox-observation-24h.redacted.md
final-acceptance.md
```

금지 대상:

- Kakao native/REST/admin key
- Provider/backend access or refresh token
- Link token/hash input
- Android signing key hash 원문
- APNs/FCM credential와 device token
- DB password/DSN
- OAuth code/state/cookie/provider identifier
- Test account password

---

## 20. Task routing index

Fresh session/subagent는 다음 범위만 읽고 작업한다.

| Assignment | Required sections | Exact repository |
|---|---|---|
| Backend auth | §3, §6 | Backend worktree |
| Credential source remediation | §0, §2.4, §5.1, §11, §19 | Backend deploy files + read-only production preflight |
| Migrations | §7, §12, §13, §18 | Backend worktree + MariaDB 10.1.38 fixture |
| Maintenance/deploy | §8, §12, §14, §18 | Backend worktree + production runtime |
| Android | §3, §9, §15 | Android worktree |
| iOS | §3, §10, §16 | iOS repository |
| Push/outbox | §5, §8, §14, §17 | Backend + Android + iOS |
| Independent review | §1–§4, 담당 section, §18–§19 | Read-only across relevant repositories |
| Production operator | §4, §12–§19 | Approved immutable artifacts only |

### 20.1 Seed traceability

| Approved requirement | Work items | Release evidence |
|---|---|---|
| Canonical `authenticated`/`linkRequired`와 strict mobile decoding | T10, T40, T50 | T90 smoke, T100/T110 E2E |
| Provider email auto-link 금지와 one-time social-link | T10, T20 | T90 social-link smoke |
| Tracked credential current-source 제거와 비노출 경로 | T05 | Secret scan/deploy-contract evidence; SMTP 대응은 §2.4 사용자 소유 예외 |
| Pending/rejected/approved와 legacy eligibility fail-closed | T10, T20 | T90 authz smoke, T100/T110 routing |
| MariaDB 10.1.38 migration 036–039 | T20, T70, T80 | Fixture, production postcheck |
| Full dump, off-host copy, isolated restore | T30, T70 | Backup/restore validation artifact |
| DB/backend/Play/TestFlight/restore 별도 승인 | G2, G3-P, G3-D, G3-S, G3-R, G4, G5, GR | Approval evidence + manifests |
| T90 four-gate exact-command approval amendment | G3-P, G3-D, G3-S, G3-R, T90 | Four approval records + ordered PASS evidence; prior PASS is not next-stage approval |
| Dirty production provenance 보완 | T00, T60 | Baseline and deployment manifest |
| Maintenance all-writer freeze와 smoke-only exception | T30, T70, T90 | Writer-zero and release postcheck |
| Android Play/iOS TestFlight actual-device acceptance | T40, T50, T100, T110 | Channel E2E artifacts |
| APNs outbox/Android direct-FCM 보존과 24시간 reset rule | T00, T30, T90, T120 | 24h observation artifact |

### 20.2 Machine-verifiable Whole-Seed coverage

`C01–C19`는 Seed `constraints`의 선언 순서와 1:1로 대응한다. AC는 Seed의 `semantic_ac_key`를 그대로 사용한다.

| Seed item | Work items / gates | Required evidence |
|---|---|---|
| C01 planning-only | §0, §22 | Planning-only status/git scope |
| C02 dirty production provenance | T00, T60 | Baseline + manifest limitation |
| C03 canonical request/outcomes/no fallback | T10, T40, T50 | Contract tests + channel E2E |
| C04 provider email prefill only | T10 | Backend social-link regression |
| C05 nested verification/approved-only access | T10, T90 | Authorization matrix smoke |
| C06 hash-backed transactional social-link | T10, T20 | Transaction/replay smoke |
| C07 Android/iOS independent callback/session/routing/restore | T40, T50, T100, T110 | Platform regression + E2E |
| C08 mobile credential non-disclosure | T05, T40, T50 | Secret/log/artifact scan |
| C09 strict vertical TDD | §0, T10, T20, T30, T40, T50 | RED/GREEN/regression evidence |
| C10 pre-mutation backup/restore/preflight/manifest/review | T20, T60(B), T70 | G2 evidence bundle |
| C11 independent DB/backend/Play/TestFlight approvals | G2, G3-P, G3-D, G3-S, G3-R, G4, G5 | Exact-command approval records |
| C12 no unapproved git mutation/history operation | §0, T60 | Git status + approval record |
| C13 clean immutable production artifact | G1B, G1A, G1I, T60 | Commit/artifact checksums |
| C14 maintenance write block/smoke-only/four separate approvals | T30, T70, G3-P, G3-D, G3-S, G3-R, T90 | Writer-zero + four approval records + ordered PASS/release evidence |
| C15 ordered 036→039 stop-on-failure | T20, T80 | Per-step postchecks |
| C16 destructive restore only after separate approval | GR, §18 | Restore approval + re-entry evidence |
| C17 Play/TestFlight production-signed acceptance | T100, T110 | Channel-installed device E2E |
| C18 test-row-only cleanup | T90, T100, T110 | Run-ID cleanup postcheck |
| C19 task-scoped Markdown handoff | §4, T00–T120, §20 | Anchor/contract validator |
| ac_464090fd6f88ba3f | T10–T50, §4 | Exact files/dependency/TDD validation |
| ac_ad53a1460bdd0a91 | T00, T60 | Provenance/manifest validation |
| ac_1cd78ccab8d7178b | T20, T30, T70–T90, GR | Migration/maintenance/recovery gates |
| ac_3829a3d775edfa71 | T10, T30, T90 | Canonical/authz/social-link/push smoke |
| ac_7b4c29cd5181337e | T40, T50, T100, T110 | Play/TestFlight E2E evidence |
| ac_a44f5ea902a8e134 | T90, T120, §18 | 24h/reset/failure-domain evidence |

---

## 21. Plan artifact verification

```bash
python3 - <<'PY'
from pathlib import Path
import hashlib, re

plan = Path('docs/plans/kakao-canonical-auth-production-rollout.md')
seed = Path('/Users/jerryhwang/.ouroboros/seeds/seed_65da4507f693.yaml')
s = plan.read_text()
seed_text = seed.read_text()
seed_hash = hashlib.sha256(seed.read_bytes()).hexdigest()
assert plan.stat().st_size > 0
assert f'Seed SHA-256: `{seed_hash}`' in s
assert 'Seed generation authorization:' in s
assert '2026-08-04T06:11:17.309937+09:00' in s
assert 'Post-generation Whole-Seed approval response: **완성된 Whole-Seed를 승인한다**' in s
assert '2026-08-04T07:36:50+09:00' in s

ids = ['T00','T05','T10','T20','T30','T40','T50','T60','T70','T80','T90','T100','T110','T120']
id_set = set(ids)
labels = [
    'Goal/Scope:', 'Explicit exclusions:', 'Repository/files:',
    'Dependencies:', 'API/DB/platform impact:', 'Implementation steps:',
    'Test steps:', 'Acceptance criteria:', 'Read before starting:',
]
positions = {}
for index, task in enumerate(ids):
    anchor = task.lower()
    assert s.count(f'<a id="{anchor}"></a>') == 1
    assert s.count(f'](#{anchor})') == 1
    start = s.index(f'<a id="{anchor}"></a>')
    positions[task] = start
    next_anchor = f'<a id="{ids[index + 1].lower()}"></a>' if index + 1 < len(ids) else '## 18. Rollback'
    section = s[start:s.index(next_anchor)]
    assert all(label in section for label in labels)
    dependency_line = re.search(r'> - \*\*Dependencies:\*\*(.*)', section).group(1)
    for dependency in set(re.findall(r'\bT\d+\b', dependency_line)) - {task}:
        assert dependency in id_set
        assert positions.get(dependency, s.index(f'<a id="{dependency.lower()}"></a>')) < start

assert set(re.findall(r'\bT\d+\b', s)) <= id_set
def validate_t90_gate_contract(document):
    policy = document.split('## 21. Plan artifact verification', 1)[0]
    allowed_gates = {'G0','G1B','G1A','G1I','G2','G3','G3-P','G3-D','G3-S','G3-R','G4','G5','GR'}
    observed_gates = set(re.findall(r'(?<![A-Za-z0-9-])(?:G\d+[A-Za-z0-9-]*|GR)(?![A-Za-z0-9-])', policy))
    assert observed_gates <= allowed_gates
    required_t90_gates = {'G3-P', 'G3-D', 'G3-S', 'G3-R'}
    assert required_t90_gates <= observed_gates
    bare_g3_lines = [line for line in policy.splitlines() if re.search(r'(?<![A-Za-z0-9-])G3(?![A-Za-z0-9-])', line)]
    assert bare_g3_lines == ['### Gate G3 — Four separate production approvals']

    expected_graph = '''T80 PASS + T60(B) immutable artifact 재검증
 → G3-P 승인
 → T90 read-only preflight 실행/PASS
 → G3-D 승인
 → T90 backend deployment 실행/PASS
 → G3-S 승인
 → T90 controlled server-side smoke/cleanup 실행/PASS
 → G3-R 승인
 → T90 maintenance release 실행/PASS'''
    graph = policy.split('T80 PASS + T60(B) immutable artifact 재검증', 1)[1].split('T60(A) + T90', 1)[0]
    graph = 'T80 PASS + T60(B) immutable artifact 재검증' + graph
    assert graph.strip() == expected_graph

    expected_gate = '''### Gate G3 — Four separate production approvals

- **G3-P read-only preflight:** local authority/artifact checksums, reviewed closure SHA-256, 64-hex attempt nonce와 production read-only state query의 exact command를 승인받는다. PASS evidence에는 approved closure, maintenance generation, current backend SHA-256/size와 consumed attempt ID를 기록한다. Production mutation은 금지한다.
- **G3-D backend deployment:** 같은 closure에 결속된 G3-P PASS evidence SHA-256, 동일 maintenance generation, immutable binary checksum, preflight-observed backend SHA-256/size, 64-hex attempt nonce, rollback command와 exact deployment command를 제시하고 별도 승인을 받는다. Build window 이후 rollback copy의 hash/size와 maintenance generation을 final replacement 전에 재검증하며 drift면 replacement/restart 없이 fail closed한다.
- **G3-S controlled smoke:** G3-D PASS 뒤 test run ID, 허용된 writer/proof 범위, exact smoke/cleanup commands와 실패 시 maintenance 유지 조건을 제시하고 별도 승인을 받는다.
- **G3-R maintenance release:** G3-D와 G3-S가 모두 PASS한 뒤 release command, generation binding, worker/job 재개 postcheck를 제시하고 별도 승인을 받는다.

어느 단계의 PASS도 다음 단계 승인이 아니다. 각 approval bit는 해당 exact command와 reviewed evidence에만 유효하다.

각 G3-P/G3-D/G3-S/G3-R approval bit는 single-use이며 reviewed evidence SHA-256과 exact command bytes에 결속된다. Approval은 첫 execution attempt 시작과 동시에 성공/실패와 무관하게 consumed된다. 같은 stage retry 또는 whole-T90 replay에서는 re-executed stage마다 갱신된 evidence를 검토하고 새 applicable approval을 받아야 한다. 각 stage launcher는 approved command에 포함된 64-hex attempt nonce로 canonical payload를 derive하고 trusted local `0600` no-overwrite consumed record를 첫 attempt 전에 생성하며, 동일 record가 이미 있으면 fail closed한다. T90-D1 launcher는 G3-P/G3-D에 이를 강제하고, G3-S/G3-R launcher도 해당 stage 승인 요청 전에 같은 contract를 구현·검증해야 한다.'''
    gate = policy.split('### Gate G3 — Four separate production approvals', 1)[1].split('### Deployment', 1)[0]
    gate = '### Gate G3 — Four separate production approvals' + gate
    assert gate.strip() == expected_gate

    expected_release = '''### Separately approved maintenance release

다음 AND와 별도 G3-R exact-command 승인이 모두 참일 때만 `release-maintenance.sh`를 실행한다.

```text
backend deployment PASS
AND full server-side smoke PASS
```

Release script는 sentinel을 제거하고 일반 background jobs/worker를 재개하며 idempotent postcheck를 수행한다. Release 실패 시 write block을 유지하고 성공으로 보고하지 않는다.

Maintenance 해제 즉시 T120의 24시간 timer를 시작한다.'''
    release = policy.split('### Separately approved maintenance release', 1)[1].split('\n---', 1)[0]
    release = '### Separately approved maintenance release' + release
    assert release.strip() == expected_release

    expected_recovery = '''## 18. Rollback / forward-fix matrix

| Failure | Immediate action | DB policy | Resume condition |
|---|---|---|---|
| Preflight/backup/restore 실패 | 중단, write block 유지 또는 안전하게 원복 | Mutation 없음 | 원인 수정 후 T70 재실행 |
| Read-only preflight 실패 | Production mutation 없이 중단; consumed G3-P 재사용 금지 | Mutation 없음 | 갱신된 closure/state evidence와 exact command로 새 G3-P 승인 |
| 036–039 단계 실패 | 다음 migration 금지, partial schema 조사 | DDL 자동 rollback 가정 금지, 우선 forward-fix | 해당 단계 postcheck PASS |
| Backfill/data integrity 손상 | write block 유지 | 안전한 forward-fix 불가 시에만 별도 승인 후 full dump restore | Restore/정합성 재검증 |
| Backend upload/restart 실패 | 이전 immutable binary 복구; consumed G3-D 재사용 및 새 G3-D 승인 전 deployment retry 금지 | Additive DB schema 유지 | 갱신된 preflight/deployment evidence와 exact command로 새 G3-D 승인; 이전 health PASS 또는 수정 artifact 승인 |
| Server-side smoke 실패 | release 금지; 갱신된 evidence와 exact commands로 새 G3-S 승인 전 재실행 금지 | 실패 유형별 forward-fix; data damage면 restore 후보 | 새 G3-S 승인으로 Full smoke/cleanup PASS 후 별도 G3-R 승인 |
| Release script 실패 | fail-closed 유지; 갱신된 failure evidence와 exact command로 새 G3-R 승인 전 retry 금지 | DB 변경 없음 | 새 G3-R 승인으로 idempotent release postcheck PASS |
| Android-only 실패 | Android channel만 차단 | Backend/DB 유지 | Android 재검증 PASS |
| iOS-only 실패 | iOS channel만 차단 | Backend/DB 유지 | iOS 재검증 PASS |
| Shared contract/security 문제 | write 재차단 | Backend rollback 또는 forward-fix; DB restore는 별도 승인 | T90 전체 재실행 시 G3-P부터 re-executed stage별 새 approval |
| 24h rollout 기인 outbox 이상 | write 재차단 | 영향별 rollback/forward-fix | Smoke + 새 24h PASS |'''
    recovery = policy.split('## 18. Rollback / forward-fix matrix', 1)[1].split('### Gate GR', 1)[0]
    recovery = '## 18. Rollback / forward-fix matrix' + recovery
    assert recovery.strip() == expected_recovery

    assert '> - **Dependencies:** T80 PASS, T60(B) checksum revalidation, G3-P/G3-D/G3-S/G3-R 각각의 명시 승인과 선행 단계 PASS.' in policy
    assert '> - **Acceptance criteria:** G3-P/D/S/R 네 approval record, read-only preflight PASS evidence, deployment+smoke+cleanup+release postcheck PASS, jobs 재개, T120 timer 시작 evidence.' in policy

validate_t90_gate_contract(s)
policy = s.split('## 21. Plan artifact verification', 1)[0]
def replace_once(document, old, new):
    assert document.count(old) == 1
    return document.replace(old, new, 1)

invalid_t90_gate_fixtures = [
    replace_once(policy, '- **G3-P read-only preflight:**', '- **G3-X1 read-only preflight:**'),
    replace_once(policy, '- **G3-P read-only preflight:**', '- **G3-x read-only preflight:**'),
    replace_once(policy, 'Production mutation은 금지한다.', 'Production mutation을 허용한다.'),
    replace_once(policy, 'backend deployment PASS\nAND full server-side smoke PASS',
                 'backend deployment PASS\nOR full server-side smoke PASS'),
    replace_once(policy, '새 applicable approval을 받아야 한다.', '새 applicable approval을 받아야 한다. 단 G3-R approval은 재사용할 수 있다.'),
    replace_once(policy, ' → G3-D 승인\n → T90 backend deployment 실행/PASS\n → G3-S 승인\n → T90 controlled server-side smoke/cleanup 실행/PASS',
                 ' → G3-S 승인\n → T90 controlled server-side smoke/cleanup 실행/PASS\n → G3-D 승인\n → T90 backend deployment 실행/PASS'),
    replace_once(policy, '새 G3-D 승인 전 deployment retry 금지', 'deployment retry 허용'),
]
for invalid_document in invalid_t90_gate_fixtures:
    try:
        validate_t90_gate_contract(invalid_document)
    except (AssertionError, ValueError, IndexError):
        pass
    else:
        raise AssertionError('invalid T90 gate fixture unexpectedly passed')

# Every Seed constraint and semantic acceptance key must have a stable traceability row.
constraint_block = seed_text.split('constraints:', 1)[1].split('acceptance_criteria:', 1)[0]
constraints = re.findall(r'^-', constraint_block, flags=re.MULTILINE)
assert len(constraints) == 19
for number in range(1, len(constraints) + 1):
    assert len(re.findall(rf'^\| C{number:02d} ', s, flags=re.MULTILINE)) == 1
seed_ac_keys = set(re.findall(r'semantic_ac_key:\s*(ac_[0-9a-f]+)', seed_text))
plan_ac_keys = set(re.findall(r'^\| (ac_[0-9a-f]+) ', s, flags=re.MULTILINE))
assert len(seed_ac_keys) == 6 and seed_ac_keys == plan_ac_keys

# Existing paths must resolve in their repository; future paths must be absent and explicitly marked 새.
roots = {
    'backend': Path('/Users/jerryhwang/Workspace/03_daeil/.worktrees/kakao-canonical-auth-hotfix/web'),
    'android': Path('/Users/jerryhwang/Workspace/03_daeil/.worktrees/dflh-saf-v2-mvp/android'),
    'ios': Path('/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2-swift'),
}
existing_paths = {
    'backend': {
        'deploy/alumni-backend.service', 'deploy/alumni-backend.env.example',
        'deploy/httpd-alumni.conf', 'deploy.sh', 'migrate.sh',
        'scripts/kakao-auth-rollout/secret-scan.sh',
        'scripts/kakao-auth-rollout/secret-scan_test.sh',
        'scripts/kakao-auth-rollout/deploy-env-contract_test.sh',
        'backend/cmd/server/main.go', 'backend/cmd/server/routes.go', 'backend/cmd/server/wire.go',
        'backend/internal/config/config.go',
        'backend/internal/model/mobile_auth.go', 'backend/internal/model/auth_principal.go',
        'backend/internal/model/request.go', 'backend/internal/model/user.go',
        'backend/internal/service/login_eligibility.go', 'backend/internal/service/auth_jwt.go',
        'backend/internal/service/mobile_auth_service.go', 'backend/internal/service/auth_service.go',
        'backend/internal/service/auth_session.go', 'backend/internal/service/push_notifier.go',
        'backend/internal/service/push_notifier_test.go', 'backend/internal/service/push_platform_provider.go',
        'backend/internal/service/push_fcm_provider.go', 'backend/internal/service/push_fcm_provider_test.go',
        'backend/internal/service/push_apns_provider.go', 'backend/internal/service/push_apns_provider_test.go',
        'backend/internal/repository/auth_principal_repo.go', 'backend/internal/repository/refresh_rotation_repo.go',
        'backend/internal/repository/social_link_continuation_repo.go', 'backend/internal/repository/auth_repo.go',
        'backend/internal/repository/session_repo.go', 'backend/internal/repository/password_reset_repo.go',
        'backend/internal/repository/visit_repo.go', 'backend/internal/repository/push_outbox_repo.go',
        'backend/internal/repository/push_outbox_repo_test.go',
        'backend/internal/handler/auth_kakao_mobile_handler.go', 'backend/internal/handler/auth_social_link_handler.go',
        'backend/internal/handler/auth_mobile_response.go',
        'backend/internal/middleware/alumni_approved.go', 'backend/internal/middleware/alumni_approved_test.go',
        'backend/internal/middleware/auth.go', 'backend/internal/middleware/admin_auth.go',
        'backend/internal/middleware/auth_revalidation_test.go',
        'backend/internal/middleware/admin_auth_revalidation_test.go',
        'backend/internal/service/mobile_access_session_test.go',
        'backend/internal/job/session_cleanup.go', 'backend/internal/job/email_worker.go',
        'backend/internal/job/visit_aggregation.go', 'backend/internal/job/push_outbox_worker.go',
        'backend/internal/job/push_outbox_worker_test.go',
        'backend/migrations/030_create_mobile_refresh_token_table.sql',
        'backend/migrations/036_extend_mobile_refresh_token_for_rotation.sql',
        'backend/migrations/037_harden_member_social_links.sql',
        'backend/migrations/038_create_auth_principal_tables.sql',
        'backend/migrations/039_create_social_link_continuation.sql', 'backend/migrations/apply_all.sql',
        'scripts/kakao-auth-rollout/preflight.sql', 'scripts/kakao-auth-rollout/postcheck.sql',
        'scripts/kakao-auth-rollout/apply-migrations.sh', 'scripts/kakao-auth-rollout/test-migrations.sh',
        'backend/migrations/testdata/kakao_auth_028_035_fixture.sql',
        'backend/migrations/testdata/kakao_auth_edge_cases.sql',
        'backend/migrations/testdata/mariadb-10.1.38.image', 'backend/migrations/kakao-auth-036-039.sha256',
        'backend/internal/maintenance/gate.go', 'backend/internal/middleware/maintenance_write.go',
        'backend/internal/middleware/maintenance_write_test.go',
    },
    'android': {
        'app/build.gradle.kts', 'app/src/main/AndroidManifest.xml',
        'app/src/main/kotlin/com/dflh/app/DflhApplication.kt',
        'app/src/main/kotlin/com/dflh/app/core/auth/AuthApi.kt',
        'app/src/main/kotlin/com/dflh/app/core/auth/AuthModels.kt',
        'app/src/main/kotlin/com/dflh/app/core/auth/SessionRepository.kt',
        'app/src/main/kotlin/com/dflh/app/core/auth/TokenSessionManager.kt',
        'app/src/main/kotlin/com/dflh/app/feature/auth/KakaoSdkLoginGateway.kt',
        'app/src/main/kotlin/com/dflh/app/feature/auth/KakaoLoginCoordinator.kt',
        'app/src/main/kotlin/com/dflh/app/feature/auth/SessionViewModel.kt',
        'app/src/main/kotlin/com/dflh/app/feature/push/DflhFirebaseMessagingService.kt',
        'app/src/main/kotlin/com/dflh/app/feature/push/PushApi.kt',
        'app/src/main/kotlin/com/dflh/app/feature/push/PushCoordinator.kt',
        'app/src/main/kotlin/com/dflh/app/feature/push/PushDeduplicator.kt',
        'app/src/main/kotlin/com/dflh/app/feature/push/PushTokenStore.kt',
        'app/src/test/kotlin/com/dflh/app/core/auth/CanonicalAuthContractTest.kt',
        'app/src/test/kotlin/com/dflh/app/core/auth/KakaoAuthContractTest.kt',
        'app/src/test/kotlin/com/dflh/app/core/auth/SocialAccountLinkContractTest.kt',
        'app/src/test/kotlin/com/dflh/app/core/auth/SessionRepositoryTest.kt',
    },
    'ios': {
        'dflh-saf-v2-swift.xcodeproj', 'Sources/App/Feature/Login/LoginModels.swift',
        'Sources/App/Feature/Login/AuthRepository.swift', 'Sources/App/Feature/Login/SocialLoginCoordinator.swift',
        'Sources/App/AppState.swift', 'Sources/App/Network/APIClient.swift',
        'Sources/Auth/TokenSessionManager.swift', 'Packages/KakaoAuthKit/Sources/KakaoAuthKit/KakaoAuthKit.swift',
        'Sources/App/Infrastructure/Push/AppDelegate.swift',
        'Sources/App/Infrastructure/Push/PushNotificationCoordinator.swift',
        'Sources/App/Feature/Push/PushDeviceService.swift', 'Sources/App/Feature/Push/PushPayloadDecoder.swift',
        'Config/DflhSafV2Swift-Release.entitlements', 'Tests/DflhSafV2SwiftTests/AuthSecurityTests.swift',
    },
}
candidate_paths = {
    'backend': {
        'backend/internal/service/login_eligibility_test.go',
        'scripts/kakao-auth-rollout/enter-maintenance.sh', 'scripts/kakao-auth-rollout/server-smoke.sh',
        'scripts/kakao-auth-rollout/release-maintenance.sh', 'scripts/kakao-auth-rollout/writer-zero.sql',
        'scripts/kakao-auth-rollout/dispose-backup.sh',
    },
    'ios': {'Config/ExportOptions-AppStore.plist'},
}
for repository, paths in existing_paths.items():
    for path in paths:
        assert (roots[repository] / path).exists(), f'missing existing path: {repository}:{path}'
        assert f'`{path}`' in s, f'unreferenced existing path: {repository}:{path}'
for repository, paths in candidate_paths.items():
    for path in paths:
        assert not (roots[repository] / path).exists(), f'candidate already exists; reclassify: {repository}:{path}'
        assert re.search(rf'(?:새|필요 시 새) `?{re.escape(path)}`?', s), f'candidate not marked 새: {repository}:{path}'

ios_test_source = (roots['ios'] / 'Tests/DflhSafV2SwiftTests/AuthSecurityTests.swift').read_text()
ios_focused_classes = {
    'SocialAuthResponseTests', 'SocialLinkFormStateTests', 'AuthenticationDestinationTests',
    'TokenSessionManagerTests', 'AuthBootstrapCoordinatorTests',
}
for test_class in ios_focused_classes:
    assert re.search(rf'final\s+class\s+{test_class}\s*:\s*XCTestCase', ios_test_source)
    assert f'-only-testing:DflhSafV2SwiftTests/{test_class}' in s

required = [
    '036', '037', '038', '039', 'MariaDB `10.1.38`',
    'authenticated', 'linkRequired', 'pending', 'rejected', 'approved',
    'Play Console', 'TestFlight', 'PROCESSING', 'DEAD', '0시간',
    'G1B', 'G1A', 'G1I', 'G2', 'G3-P', 'G3-D', 'G3-S', 'G3-R', 'G4', 'G5', 'GR',
]
assert all(item in s for item in required)
assert ('TO' + 'DO') not in s and ('TB' + 'D') not in s
print(
    'PLAN VALIDATION PASS '
    f'tasks={len(ids)} constraints={len(constraints)} acceptance={len(seed_ac_keys)} '
    f'existing_paths={sum(map(len, existing_paths.values()))} '
    f'candidate_paths={sum(map(len, candidate_paths.values()))} seed={seed_hash}'
)
PY
sha256sum docs/plans/kakao-canonical-auth-production-rollout.md
```

> macOS에서는 마지막 command를 `shasum -a 256`으로 실행한다.

---

## 22. Planning-only 생성 시점과 후속 실행 상태

최초 문서 작성 단계에서는 다음을 실행하지 않았다.

- Backend/Android/iOS source 구현
- Migration fixture 또는 production DB mutation
- Production dump
- Commit/push
- Backend deployment
- Play Console/TestFlight upload
- Mobile acceptance

후속 승인에 따라 T00, T05, T10, T20과 T30 source/local remediation은 실행·검증되었다. §2.4 scope amendment에 따라 SMTP/permission 대응은 사용자 소유로 제외되었다. P10 production read-only preflight는 mutation 없이 수행됐고 maintenance pre-stage prerequisites 부재로 NO-GO다. 전체 rollout은 NO-GO이며 EnvironmentFile/Apache/PHP pre-stage, service transition, dump, migration, deployment와 channel upload는 각 gate에서 사용자의 별도 명시 승인을 받는다.
