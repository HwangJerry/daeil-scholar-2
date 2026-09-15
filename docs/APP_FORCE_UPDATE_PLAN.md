# 앱 강제 업데이트 작업 계획

- 작성일: 2026-09-15 (개정: 관리자 활성화/버전 기준 관리, 플랫폼별 개별 관리 반영)
- 범위: `dflh-saf-v2/backend`, `dflh-saf-v2/admin`, `dflh-saf-v2-kotlin`, `dflh-saf-v2-swift`
- 목표: 관리자가 iOS와 Android의 업데이트 정책(활성화 여부, 버전 기준)을 각각 관리하고, 앱(1차)과 API 서버(2차) 양쪽에서 구버전을 차단한다.

## 0. 인계 요약 (다음 세션은 여기부터 읽기)

### 현재 상태 (2026-09-15)

- **단계: Phase 1(백엔드) 커밋 완료(`875de7f`), Phase 2(관리자 화면) 구현 완료·커밋 전.** 작업 브랜치 `feat/app-force-update-backend` (`dflh-saf-v2` 저장소)
- 백엔드: `go build`/`go vet`/`go test ./...` 통과. 관리자: `tsc -b`, `npm run lint`, `npm run test`(17파일 66테스트), `npm run build` 통과
- 다음은 Phase 3~5(앱). Android와 iOS는 서로 병렬 가능
- 앱은 아직 출시 전이고 배포된 구버전이 없다. **심사를 통과한 현재 빌드는 출시하지 않고 보류**하며, 게이트가 들어간 빌드를 첫 공개 릴리스로 낸다 (D15)
- 따라서 게이트 없는 사용자는 처음부터 존재하지 않는다. 8절의 구버전 대응 선택지는 모두 불필요해졌다

### 결정된 사항

| # | 결정 | 근거 |
|---|---|---|
| D1 | 외부 라이브러리 대신 **자체 백엔드가 정책을 내려주는 방식** | 두 플랫폼을 한 번에 다루는 네이티브 라이브러리가 없음. 강제 여부를 우리가 정해야 함. iOS에 Firebase가 없어 Remote Config 도입 비용이 큼 |
| D2 | 정책 저장소는 **기존 `app_settings` 테이블 재사용**, 플랫폼별 JSON 키(`app_update_policy_ios`, `app_update_policy_android`) | 앱이 이미 `GET /api/settings/public`을 호출하므로 추가 요청이 없음. 한 행 저장이라 원자적이고 플랫폼끼리 분리됨 |
| D3 | 버전 비교는 **빌드 번호**(iOS `CFBundleVersion`, Android `versionCode`) | 버전 이름이 사실상 고정(Android `1.0.0`, iOS `1.0`)이라 비교에 못 씀. 빌드 번호는 릴리스마다 반드시 커짐 |
| D4 | **이중 차단**: 앱 게이트(1차) + 서버 426 미들웨어(2차). 둘 다 **fail-open** | 켜둔 세션이나 게이트 버그도 막기 위해. 서버 장애가 전체 차단으로 번지지 않게 |
| D5 | 관리자 **전용 페이지 `/app-update`**. iOS/Android 카드가 따로 있고, 각각 강제/권장 토글, 기준 빌드, 저장 버튼, 변경 이력을 가짐 | 요구사항 R1~R3 |
| D6 | 기준 빌드는 **앱이 보고한 빌드 목록(`app_client_builds`)에서 선택**. 목록에 없으면 경고와 함께 직접 입력 | 관리자가 빌드 번호를 외울 필요 없음 |
| D7 | 사고 방지: 서버 검증(3절), 강제 켜기/기준 상향 시 "스토어 100% 배포 완료" 체크 필수, 동시 수정 `409`, 변경 이력 테이블 | 값 하나로 전체 사용자가 차단될 수 있음 |
| D8 | 기존 "앱 설정" 화면과 `PUT /api/admin/settings/{key}`에서는 정책 키를 **숨기고 거부** | 검증을 우회하는 수정 경로 제거 |
| D9 | 판정 순서: OS 미지원 → 강제 → 권장. **OS 미지원 사용자는 차단하지 않음** | 앱 안에서 해결할 방법이 없는 사용자를 가두지 않기 위해 |
| D10 | Android: 강제는 Play In-App Update `IMMEDIATE`, 권장은 `FLEXIBLE`. iOS: 자체 강제 화면 + 권장 바텀시트 | 각 플랫폼의 표준 방식 |
| D11 | 권장 노출: 메인 화면에서만, 3일 간격, 권장 빌드당 최대 3회, 이후 마이페이지 표시만 (6.1절) | 반복 팝업은 무시하는 습관만 만듦 |
| D12 | OS 미지원 안내: 닫을 수 있는 1회성 다이얼로그, 기준 OS 값마다 1회, 마이페이지 상시 표시. iOS 설정 딥링크 금지 (6.2절) | 차단하지 않고 알리기만. 딥링크는 비공개 API라 심사 거절 사유 |
| D13 | 디버그 빌드는 게이트 비활성. **구현 방식: 디버그 빌드는 버전 헤더 자체를 보내지 않는다** | 로컬 빌드 번호 기본값이 `1`이라 항상 걸림. 서버 게이트는 헤더가 없으면 무조건 통과시키므로, 헤더를 빼는 것이 앱과 서버 양쪽에서 동시에 성립하는 유일한 방법 |
| D14 | 운영: 최소 빌드는 해당 플랫폼 스토어 100% 배포 후에만 올림. 플랫폼별로 따로 운영 (10절) | 단계적 출시 중에 올리면 대상이 아닌 사용자가 갇힘 |
| D15 | **심사를 통과한 현재 빌드는 출시하지 않고 보류.** 게이트가 들어간 빌드를 첫 공개 릴리스로 낸다 | 한 번도 출시된 적이 없어, 보류하면 게이트 없는 구버전이 0이 된다. iOS는 "수동 출시", Android는 "관리형 게시" 설정 완료 (2026-09-15) |
| D16 | 출시 범위: **Phase 1~5를 모두 끝내고** 첫 출시 | 관리자 화면까지 완성된 상태로 시작. 사용자 결정 (2026-09-15) |
| D17 | **단계적 출시(phased release / staged rollout)를 쓰지 않는다.** 100% 배포로 릴리스 | 단계적 출시 중에는 최소 빌드를 올릴 수 없다. 사용자 결정 (2026-09-15) |
| D18 | iOS `storeUrl`은 마이그레이션 기본값을 빈 문자열로 두고 활성화 전에 채운다. 검증 규칙으로 빈 값 상태의 강제 활성화를 막는다 | 숫자 App ID가 없어도 Phase 1 착수가 가능 |

**범위 밖 (이번에 하지 않음):** 특정 빌드 차단 목록, 구빌드 "지원 종료" 정책, 서버 문구 관리, 노출 빈도의 관리자 설정화

### 아직 정해야 하는 사항

| # | 항목 | 선택지 / 추천 | 필요한 시점 |
|---|---|---|---|
| O1 | **iOS App Store 숫자 App ID** | App Store Connect 앱 화면 URL(`/apps/<숫자>/`) 또는 앱 정보의 "Apple ID"에서 확인. 저장소에 없고(`fastlane/Appfile`은 bundle ID만) 발급자 ID도 없어 조회 불가. **착수를 막지는 않음** (D18) | 첫 출시 빌드 제출 전 |
| O3 | 스토어 릴리스마다 **버전 이름도 올리는 규칙** 채택 여부 | 채택 추천. 비교에는 안 쓰지만 관리자 빌드 목록에서 구분이 쉬워짐. 채택 시 `project.yml` `MARKETING_VERSION`, GitHub 변수 `APP_VERSION_NAME` 관리 방식 변경 필요 | Phase 3 전 |
| O4 | 강제/권장/OS 미지원 화면을 **design-system 계약**(`design-system/contracts/`)에 등록할지 | 새 화면이라 등록 여부 결정 필요. 등록하지 않으면 기존 DS 컴포넌트만으로 구성 | Phase 4/5 전 |
| O5 | 활성화 전 검증 환경 | 완료 기준의 "스테이징 검증"을 어디서 할지. 스테이징 서버 유무 확인 필요 | Phase 1 배포 전 |

### Phase 1에서 실제로 만든 것 (2026-09-16)

| 파일 | 역할 |
|---|---|
| `migrations/071_seed_app_update_policies.sql` | 플랫폼별 정책 행 2개 시드 (둘 다 꺼짐) |
| `migrations/072_create_app_update_policy_history.sql` | 정책 변경 이력 테이블 |
| `migrations/073_create_app_client_builds.sql` | 관측된 빌드 테이블 (사용자 식별자 없음) |
| `internal/model/app_update_policy.go` 외 2개 | 정책·이력·관측 빌드 모델 |
| `internal/service/app_update_evaluator.go` | 판정 규칙 (순수 함수) |
| `internal/service/app_update_policy_validation.go` | 필드 단위 검증 |
| `internal/service/app_update_policy_service.go` | 정책 조회/저장/이력, 저장 후 공개 캐시 무효화 |
| `internal/service/app_client_build_service.go` | 빌드 관측 (10분 스로틀, 버전명 64자 절단) |
| `internal/repository/app_update_policy_repo.go` | `FOR UPDATE` 잠금 + 설정 갱신 + 이력 기록을 한 트랜잭션으로 |
| `internal/repository/app_client_build_repo.go` | 빌드 upsert/목록/최댓값 |
| `internal/handler/admin_app_update_handler.go` | 관리자 API 4개 |
| `internal/middleware/client_app_headers.go` | 버전 헤더 파싱 (게이트와 기록기가 공유) |
| `internal/middleware/app_version_gate.go` | 426 차단, 예외 경로, fail-open |
| `internal/middleware/app_client_build_recorder.go` | **인증된 요청만** 빌드를 기록 |

기존 파일 변경: `app_setting_service.go`(정책 키 쓰기 거부 + 캐시 무효화 메서드), `app_setting_handler.go`(거부를 400으로 매핑), `routes.go`/`wire.go`/`main.go`(배선), `apply_all.sql`(검증 쿼리).

구현하며 확정한 세부 사항:

- `minOsVersion` 판정은 iOS `17.0` 형태와 Android API 레벨 `26`을 같은 코드로 비교한다
- iOS `storeUrl`은 강제/권장 중 하나라도 켤 때만 필수다. 꺼진 기본 상태에서는 빈 값이 유효하다
- 관측된 적 없는 빌드를 기준으로 삼으려면 `allowUnobservedBuild`를 명시해야 한다
- 낙관적 동시성은 `updatedAt` 비교로 하고, 불일치면 아무것도 쓰지 않고 409를 돌려준다
- **빌드 관측은 인증된 요청에서만 한다.** 관측된 최댓값이 관리자 입력의 상한이므로, 익명 요청이 빌드 번호를 만들어낼 수 있으면 안전장치가 무력화된다. 로그인 전 화면의 빌드는 기록되지 않는다
- Android `minOsVersion`은 정수(API 레벨)만 허용한다. `8.0`처럼 입력하면 어떤 기기와도 맞지 않아 안내가 조용히 사라진다
- 정책 키 거부는 대소문자를 구분하지 않는다. `AS_KEY` 콜레이션이 대소문자를 구분하지 않아 다른 표기로 같은 행에 도달할 수 있다
- `updatedAt`이 비어 있으면(제로값) 저장을 거부한다. 동시 수정 검사가 통째로 건너뛰어지기 때문이다
- 한쪽 플랫폼 행이 깨져도 다른 쪽은 조회된다(`unavailable` 표시). 강제 잠금을 끄는 경로가 막히면 안 된다

### Phase 2에서 실제로 만든 것 (2026-09-16)

| 파일 | 역할 |
|---|---|
| `admin/src/types/appUpdate.ts` | 정책·이력·관측 빌드 타입 |
| `admin/src/api/appUpdate.ts` | 관리자 API 4개 호출 |
| `admin/src/lib/appUpdatePolicyErrors.ts` | 필드 오류 추출과 상황별 안내 문구 |
| `admin/src/hooks/useAppUpdatePolicies.ts` 외 3개 | 정책·빌드·이력 조회, 저장 뮤테이션 |
| `admin/src/components/appUpdate/PlatformPolicyCard.tsx` | 플랫폼 하나의 독립 폼 |
| `admin/src/components/appUpdate/BuildSelect.tsx` | 관측 빌드 선택 + 직접 입력(경고 표시) |
| `admin/src/components/appUpdate/PolicyChangeConfirmDialog.tsx` | 배포 완료 체크 전 저장 불가 |
| `admin/src/components/appUpdate/PolicyHistoryList.tsx` | 펼칠 때 이력 조회 |
| `admin/src/pages/AppUpdatePolicyPage.tsx` | `/app-update` 화면 |

기존 파일 변경: `api/client.ts`(오류 본문 `payload` 보존 — 백엔드의 `details.fields`를 읽기 위해), `routes.tsx`, `navItems.ts`, `AdminSidebar.tsx`, `AppSettingsPage.tsx`(정책 키 숨김).

구현하며 확정한 세부 사항:

- 카드 `key`에 `updatedAt`을 넣어, 저장 후 최신 값으로 폼이 자연스럽게 리셋되게 했다
- 빌드를 직접 입력하면 `allowUnobservedBuild`가 자동으로 켜진다. 경고 문구도 함께 표시한다
- 확인 다이얼로그는 **강제를 새로 켤 때와 최소 빌드를 올릴 때만** 뜬다. 끄거나 낮추는 건 즉시 저장된다
- 한쪽 정책 행이 깨져도 다른 쪽 카드는 정상 동작한다
- 저장 요청에 `expectedPolicy`(관리자가 보고 있던 정책)를 함께 보낸다. `UPDATED_AT`이 초 단위라 같은 초에 들어온 동시 수정은 타임스탬프만으로 구분할 수 없다
- 빌드 선택은 "직접 입력했는지"가 아니라 "실제로 관측된 적 없는 값인지"로 `allowUnobservedBuild`를 정한다. 그래야 서버의 미관측 빌드 검증이 실제로 동작한다

### 다음 작업

1. Phase 3~5(앱). Android(Phase 4)와 iOS(Phase 5)는 서로 병렬 가능하다
2. 배포 시 8.1절의 마이그레이션 승인 절차를 반드시 함께 처리한다
3. O1(iOS 숫자 App ID)은 첫 출시 빌드를 제출하기 전까지 확보하면 된다. O3~O5는 해당 Phase 착수 전에 정한다
4. 출시는 Phase 1~5가 모두 끝난 뒤다 (D16). 그때까지 심사 통과 빌드는 출시하지 않고 보류한다 (D15)

### 구현 시 확인할 것

- 마이그레이션 번호: 작성 시점 기준 마지막이 `070_create_comment_reports.sql`이라 `071`~`073`을 계획했다. 착수할 때 다시 확인한다
- Android: `DflhApiClient.defaultOkHttpClient()`를 거치지 않는 OkHttp 인스턴스(실시간 메시지 소켓 등)가 있는지 확인한다
- iOS: `APIClient`의 요청 생성 경로가 여러 개(일반, 업로드)라 헤더 적용이 빠지지 않게 한다
- 백엔드 규칙: `dflh-saf-v2/CLAUDE.md`(파일 단일 책임, MariaDB 10.1 제약)
- Android 규칙: `dflh-saf-v2-kotlin/AGENTS.md`(feature 단위 구성, design-system 토큰만 사용)
- iOS 규칙: `dflh-saf-v2-swift/AGENTS.md`

### 문서 이력

| 날짜 | 내용 |
|---|---|
| 2026-09-15 | 최초 작성. 관리자 활성화/버전 기준 관리, 플랫폼별 개별 관리, 권장·OS 미지원 안내 UX, 심사 중 빌드 반영 |
| 2026-09-16 | Phase 2 코드 리뷰 반영: 같은 초 동시 수정 감지(`expectedPolicy`), 미관측 빌드 판정 정확화, 직접 입력 필드 유지, 버전명 덮어쓰기 방지, 관측 실패 시 스로틀 미적용 |
| 2026-09-16 | Phase 2(관리자 화면) 구현 완료 |
| 2026-09-16 | Phase 1 코드 리뷰 반영: 빌드 관측을 인증 요청으로 제한, 정책 키 거부 대소문자 무시, Android API 레벨 검증, 제로 `updatedAt` 거부, `ListPolicies` 부분 실패 허용 |
| 2026-09-16 | Phase 1(백엔드) 구현 완료. 마이그레이션 승인 다이제스트 갱신 |
| 2026-09-15 | 첫 출시 전략 확정(D15~D17): 심사 통과 빌드 보류, Phase 1~5 완료 후 게이트 포함 빌드로 첫 출시, 단계적 출시 미사용. iOS `storeUrl` 처리 방식 확정(D18)으로 O1/O2 해소 |

## 1. 요구사항

| # | 요구사항 | 반영 |
|---|---|---|
| R1 | 관리자 화면에서 업데이트 기능 활성화 여부를 관리 | 강제/권장 각각 켜고 끄는 토글 |
| R2 | 관리자 화면에서 버전 기준을 관리 | 최소 빌드/권장 빌드를 앱이 보고한 빌드 목록에서 선택 (직접 입력 가능) |
| R3 | Android와 iOS를 개별 관리 | 플랫폼별 독립 정책. 저장, 이력, 적용이 서로 영향을 주지 않음 |

## 2. 설계 요약

```
관리자 "앱 업데이트 관리" 화면 (iOS 카드 | Android 카드, 각각 저장)
        │ PUT /api/admin/app-update-policies/{platform}   ← 검증 + 이력 기록 + 캐시 무효화
        ▼
app_settings: app_update_policy_ios / app_update_policy_android  (JSON, 공개)
        │
        ├─ GET /api/settings/public (기존) ──▶ 앱 AppUpdateGate: force / recommend / unsupportedOS / none
        │
        └─ 서버 AppVersionGate 미들웨어 ◀── 모든 API 요청 헤더: X-App-Platform, X-App-Build, X-App-Version, X-App-OS-Version
                 │  force 대상이면 426 APP_UPDATE_REQUIRED
                 └─ 관측된 빌드 기록 ──▶ app_client_builds ──▶ 관리자 빌드 선택 목록
```

### 기존 자산 재사용

| 자산 | 위치 | 활용 |
|---|---|---|
| `app_settings` 테이블 + 공개 설정 API | `backend/internal/service/app_setting_service.go`, `GET /api/settings/public` | 정책 저장소와 앱 전달 경로. 앱 시작 시 추가 요청이 없음 |
| 관리자 설정 메뉴 | `admin/src/components/layout/navItems.ts`, `AdminSidebar.tsx` | 같은 그룹에 "앱 업데이트" 메뉴 추가 |
| 관리자 UI 컴포넌트 | `admin/src/components/ui/` (`ConfirmDialog`, `Select`, `Badge` 등) | 새 화면 구성 |
| Android 공개 설정 조회 | `core/settings/PublicSettingsRepository.kt` | 같은 응답에서 정책 파생 |
| iOS 공개 설정 조회 | `Feature/Login/AppSettingsRepository.swift` | 같은 응답에서 정책 파생 |

### 버전 기준: 빌드 번호

두 앱 모두 버전 이름(마케팅 버전)을 릴리스마다 올리지 않는다. Android는 `APP_VERSION_NAME` 기본값이 `1.0.0`(`.github/workflows/play-deploy.yml:99`)이고, iOS는 `MARKETING_VERSION: "1.0"`(`project.yml:64`)으로 고정이다. 반면 빌드 번호는 릴리스마다 반드시 커진다.

| 플랫폼 | 비교 값 | 생성 규칙 |
|---|---|---|
| iOS | `CFBundleVersion` | `fastlane/Fastfile:73`, `yyyyMMddHHmm` (예: `202609151230`) |
| Android | `versionCode` | `fastlane/lib/play_release.rb`, 날짜 오프셋 + 일일 카운터 |

- 비교는 빌드 번호로 한다.
- 관리자는 숫자를 외울 필요가 없다. 앱이 보고한 빌드를 **"버전명 (빌드) · 최초/최근 확인일"** 목록에서 고른다.
- 운영 규칙 제안: 스토어 릴리스마다 버전 이름도 올린다. 비교에는 쓰지 않지만 관리자 화면에서 빌드를 구분하기 쉬워진다.
- 로컬/디버그 빌드는 빌드 번호 기본값이 `1`이라 항상 기준 미만이 된다. **디버그 빌드는 게이트를 끈다.**

## 3. 정책 모델

`app_settings`에 플랫폼별 키 하나씩, JSON으로 저장한다(`AS_PUBLIC = 'Y'`). 한 행에 한 번에 쓰므로 필드 간 검증과 원자적 저장이 쉽고, 플랫폼끼리 완전히 분리된다.

```jsonc
// app_update_policy_ios
{
  "forceEnabled": false,        // 강제 업데이트 사용 여부
  "minBuild": 0,                // 이 빌드 미만이면 강제
  "recommendEnabled": false,    // 권장 업데이트 사용 여부
  "recommendedBuild": 0,        // 이 빌드 미만이면 권장
  "minOsVersion": "17.0",       // 현재 스토어 최신 빌드의 최소 OS
  "storeUrl": "https://apps.apple.com/app/id<APP_ID>"
}

// app_update_policy_android
{
  "forceEnabled": false,
  "minBuild": 0,
  "recommendEnabled": false,
  "recommendedBuild": 0,
  "minOsVersion": "26"          // API 레벨. 스토어 경로는 applicationId(com.dflh.saf.v2)로 고정
}
```

### 검증 규칙 (서버에서만 강제)

| 규칙 | 이유 |
|---|---|
| `platform`은 `ios` / `android`만 허용 | 개별 관리 |
| 빌드 값은 음이 아닌 정수 | 형식 오류 방지 |
| `forceEnabled`이면 `minBuild > 0` | 켰는데 기준이 없는 상태 방지 |
| `recommendEnabled`이면 `recommendedBuild > 0` | 같은 이유 |
| 둘 다 켜져 있으면 `recommendedBuild >= minBuild` | 정책 모순 방지 |
| `minBuild`, `recommendedBuild`는 관측된 최신 빌드 이하 | 존재하지 않는 미래 빌드를 입력해 전체 차단되는 사고 방지. 관측 이력이 없는 빌드는 관리자가 명시적으로 확인해야 저장 |
| iOS `minOsVersion`은 `^\d+(\.\d+){0,2}$`, Android는 정수 | 형식 오류 방지 |
| iOS 정책은 `forceEnabled`일 때 `storeUrl`이 비어 있으면 안 됨 | 이동할 곳 없는 강제 화면 방지 (D18) |
| iOS `storeUrl`은 `https://apps.apple.com/`로 시작 | 잘못된 링크 방지 |
| 저장 요청에 마지막으로 읽은 `updatedAt`을 포함하고, 다르면 `409` | 관리자 두 명의 동시 수정 덮어쓰기 방지 |

### 판정 규칙 (앱과 서버가 동일하게 구현)

```
1. 디버그 빌드, 정책 없음, 정책 파싱 실패       → none
2. 기기 OS < minOsVersion                        → unsupportedOS  (업데이트를 받을 수 없으므로 차단하지 않음)
3. forceEnabled && build < minBuild              → force
4. recommendEnabled && build < recommendedBuild  → recommend
5. 그 외                                          → none
```

## 4. 관리자 화면 설계

새 페이지 `admin/src/pages/AppUpdatePolicyPage.tsx`, 경로 `/app-update`, 메뉴 "앱 업데이트"(앱 설정 그룹).

```
┌─ iOS ───────────────────────────┐  ┌─ Android ───────────────────────┐
│ 상태: ● 강제 업데이트 사용 중      │  │ 상태: ○ 사용 안 함               │
│                                 │  │                                 │
│ 강제 업데이트        [● 켜짐]     │  │ 강제 업데이트        [○ 꺼짐]     │
│  최소 빌드 [1.2.0 (202609101530)▾]│  │  최소 빌드 [선택 ▾]              │
│ 권장 업데이트        [● 켜짐]     │  │ 권장 업데이트        [○ 꺼짐]     │
│  권장 빌드 [1.3.0 (202609141100)▾]│  │  권장 빌드 [선택 ▾]              │
│ 최소 지원 OS [17.0]              │  │ 최소 지원 API 레벨 [26]          │
│ 스토어 URL  [https://apps...]    │  │                                 │
│                    [iOS 저장]    │  │                  [Android 저장]  │
│ 변경 이력 ▸                      │  │ 변경 이력 ▸                      │
└─────────────────────────────────┘  └─────────────────────────────────┘
```

- **플랫폼별로 독립된 폼과 저장 버튼.** 한쪽 저장이 다른 쪽 값을 건드리지 않는다
- **빌드 선택**: `app_client_builds`에서 해당 플랫폼 빌드를 최신순으로 보여준다. 목록에 없는 빌드는 "직접 입력"으로 넣을 수 있지만 경고가 뜬다
- **확인 다이얼로그**(`ConfirmDialog`): 강제 업데이트를 켜거나 최소 빌드를 올릴 때 띄운다
  - 체크 항목: "해당 빌드가 스토어에 100% 배포 완료됨" (체크해야 저장 가능)
  - 문구: "빌드 N 미만의 {플랫폼} 앱은 즉시 사용이 차단됩니다"
- **변경 이력**: 누가, 언제, 어떤 값을 어떤 값으로 바꿨는지 플랫폼별로 표시
- **기존 "앱 설정" 화면**(`AppSettingsPage`)에서는 `app_update_policy_*` 키를 숨긴다. 수정 경로를 하나로 유지해 검증을 우회하지 못하게 한다

## 5. 작업 단계

### Phase 0. 착수 전 결정 사항

- [ ] iOS App Store 숫자 App ID 확인. 저장소에 없음(`fastlane/Appfile`에는 bundle ID만 있음)
- [ ] 심사 중인 빌드 처리 방침. 8절 참고
- [x] 이미 배포된 구버전: 없음 (2026-09-15 기준 첫 출시 심사 중)
- [x] 권장 업데이트 노출 빈도: 6.1절
- [x] OS 미지원 사용자 안내 방식: 6.2절

### Phase 1. 백엔드 (먼저 배포, 모든 정책이 꺼진 상태라 동작 변화 없음)

1. **마이그레이션** (054와 같은 `INSERT ... WHERE NOT EXISTS` 패턴, MariaDB 10.1 호환, `apply_all.sql`과 `mvp_migration_contract_test.go` 갱신)
   - `071_seed_app_update_policies.sql`: `app_update_policy_ios`, `app_update_policy_android` 기본값(전부 꺼짐) 삽입
   - `072_create_app_update_policy_history.sql`: `PLATFORM`, `BEFORE_JSON`, `AFTER_JSON`, `CHANGED_BY`, `CHANGED_AT`
   - `073_create_app_client_builds.sql`: `PLATFORM`, `BUILD`, `VERSION_NAME`, `FIRST_SEEN_AT`, `LAST_SEEN_AT`. PK `(PLATFORM, BUILD)`. **사용자 식별자는 저장하지 않는다**
2. **도메인 모델** `internal/model/app_update_policy.go`: 정책 구조체, 플랫폼 상수, 판정 결과 타입
3. **검증** `internal/service/app_update_policy_validation.go`: 3절 검증 규칙
4. **판정** `internal/service/app_update_evaluator.go`: 3절 판정 규칙. 순수 함수
5. **정책 서비스** `internal/service/app_update_policy_service.go`
   - 플랫폼별 조회(공개 설정 캐시 재사용), 저장(검증 → `app_settings` 갱신 + 이력 기록을 한 트랜잭션으로 → 공개 설정 캐시 무효화), 이력 조회
   - 저장 후 캐시를 바로 무효화하므로 서버 차단은 즉시 반영된다
6. **빌드 관측** `internal/service/app_client_build_service.go`
   - 미들웨어가 넘긴 `(platform, build, versionName)`을 upsert. 같은 조합은 인메모리 캐시로 10분에 한 번만 DB에 쓴다
7. **관리자 API** `internal/handler/admin_app_update_handler.go`
   - `GET /api/admin/app-update-policies`: 두 플랫폼 정책
   - `PUT /api/admin/app-update-policies/{platform}`: 한 플랫폼만 저장. `409`(동시 수정), `400`(검증 실패, 필드별 사유 포함)
   - `GET /api/admin/app-update-policies/{platform}/history`
   - `GET /api/admin/app-client-builds?platform=`
8. **기존 설정 API 보호**: `PUT /api/admin/settings/{key}`는 `app_update_policy_*` 키를 거부(`400`)한다. 검증을 거치는 경로를 하나로 유지하기 위해서
9. **서버 차단 미들웨어** `internal/middleware/app_version_gate.go`
   - `X-App-Platform`이 `ios`/`android`이고 `X-App-Build`가 정수일 때만 판정한다. 헤더가 없으면(웹 SPA) 통과
   - `force`면 `426` + `{"code":"APP_UPDATE_REQUIRED","message":"..."}` (`model.APIError` 형식)
   - 예외 경로: `/api/health`, `/api/settings/public`, 로그아웃
   - 정책 조회에 실패하면 통과시킨다(fail-open)
   - 판정 후 빌드 관측 서비스로 전달
   - `cmd/server/routes.go`의 `registerAPIRoutes` 그룹에 등록
10. **테스트**
    - 검증: 규칙별 테이블 테스트
    - 판정: 플랫폼 × 토글 × 빌드 × OS 조합
    - 미들웨어: 헤더 없음 통과 / 잘못된 헤더 통과 / 토글 꺼짐 통과 / 기준 미만 426 / OS 미지원 통과 / 예외 경로 통과 / **iOS 정책 변경이 Android 판정에 영향 없음**
    - 서비스: 저장과 이력 기록의 원자성, `409` 동시 수정, 저장 후 캐시 무효화
    - 기존 설정 API가 정책 키를 거부하는지
    - `routes_verification_test.go`: 관리자 라우트 권한, 예외 경로

### Phase 2. 관리자 화면

1. `types/appUpdatePolicy.ts`: 정책, 이력, 빌드 타입
2. `api/`에 정책/이력/빌드 API 함수
3. `hooks/useAppUpdatePolicies.ts`, `hooks/useUpdateAppUpdatePolicy.ts`, `hooks/useAppClientBuilds.ts`, `hooks/useAppUpdatePolicyHistory.ts`
4. `components/appUpdate/`
   - `PlatformPolicyCard.tsx`: 플랫폼 하나의 폼. 로컬 상태와 저장 담당
   - `BuildSelect.tsx`: 관측 빌드 선택 + 직접 입력
   - `PolicyChangeConfirmDialog.tsx`: 강제 활성화/기준 상향 확인
   - `PolicyHistoryList.tsx`
5. `pages/AppUpdatePolicyPage.tsx`: iOS/Android 카드 2개를 나란히 배치. 좁은 화면에서는 세로로 쌓는다
6. `routes.tsx`, `navItems.ts`, `AdminSidebar.tsx`에 `/app-update` 추가
7. `AppSettingsPage`에서 `app_update_policy_*` 키 숨김
8. 에러 처리: `409`면 "다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요", `400`이면 필드별 사유 표시
9. 테스트(`__tests__/`): 플랫폼별 독립 저장, 검증 오류 표시, 확인 다이얼로그 체크 전 저장 불가

### Phase 3. 클라이언트 공통: 요청 헤더

| 헤더 | Android | iOS |
|---|---|---|
| `X-App-Platform` | `android` | `ios` |
| `X-App-Build` | `BuildConfig.VERSION_CODE` | `CFBundleVersion` |
| `X-App-Version` | `BuildConfig.VERSION_NAME` | `CFBundleShortVersionString` |
| `X-App-OS-Version` | `Build.VERSION.SDK_INT` | `UIDevice.current.systemVersion` |

- **Android**: `DflhApiClient.defaultOkHttpClient()`(`DflhApiClient.kt:494`)에 `AppClientHeadersInterceptor`(신규, `core/network/`) 추가. 이 클라이언트를 거치지 않는 OkHttp 인스턴스(실시간 메시지 소켓 등)가 있는지 확인
- **iOS**: `Network/AppClientHeaders.swift`(신규)의 `apply(to: inout URLRequest)`를 `APIClient`의 모든 요청 생성 경로(일반 요청, 업로드)에서 호출

### Phase 4. Android 게이트 (`feature/appupdate/`)

1. 의존성: `com.google.android.play:app-update-ktx` (`libs.versions.toml`)
2. `AppUpdatePolicy.kt`: `app_update_policy_android` JSON 파싱. **자기 플랫폼 키만 읽는다**
3. `AppUpdateEvaluator.kt`: 판정 규칙. 단위 테스트 대상
4. `PublicSettingsRepository`: 한 번의 조회 결과에서 `kakaoOpenChatUrl`과 `appUpdatePolicy`를 함께 파생. 정책은 캐시에 저장하고, 조회 실패 시 캐시 값을 쓴다
5. `AppUpdateGate.kt`: `MainActivity` 콘텐츠 최상위를 감싼다
   - `force`: Play In-App Update `IMMEDIATE`. `onResume`에서 `DEVELOPER_TRIGGERED_UPDATE_IN_PROGRESS`면 흐름 재개. 실패/미지원(사이드로드)이면 닫을 수 없는 자체 화면 + `market://details?id=com.dflh.saf.v2`
   - `recommend`: Play In-App Update `FLEXIBLE`. 노출 규칙은 6.1절. 다운로드가 끝나면 "재시작하여 업데이트" 스낵바
   - `unsupportedOS`: 6.2절
6. 426 처리: `DflhApiClient`에서 `APP_UPDATE_REQUIRED`를 전용 에러로 매핑하고, 앱 전역 `AppUpdateSignal`(SharedFlow)로 게이트를 `force`로 전환
7. 재확인: 현재는 `LaunchedEffect`에서 1회만 `refresh()`한다. `ON_RESUME`에도 재조회(10분 스로틀). **강제 화면에서는 스로틀 없이 재조회**해서 관리자가 끄면 바로 풀리게 한다
8. 디버그 빌드는 게이트 비활성
9. 화면은 design-system 컴포넌트와 토큰만 사용

### Phase 5. iOS 게이트 (`Feature/AppUpdate/`)

1. `AppUpdatePolicy.swift`(`app_update_policy_ios`만 파싱), `AppUpdateEvaluator.swift`(순수 함수, XCTest 대상)
2. `AppSettingsRepository`: 공개 설정 응답을 한 번 디코딩해서 카카오 URL과 정책을 함께 반환하도록 확장. 정책은 `UserDefaults`에 캐시
3. `ForceUpdateView.swift`: 닫을 수 없는 전체 화면. 버튼은 정책의 `storeUrl`로 `SKStoreProductViewController` 또는 `itms-apps://` 링크
4. `RecommendUpdateSheet.swift`(6.1절), `UnsupportedOSNotice.swift`(6.2절)
5. `RootView`: 인증 분기(`AuthenticationDestination`)보다 먼저 게이트를 평가. 로그인 화면에서도 차단돼야 하기 때문
6. `AppState`: `bootstrapSession()` 시작과 `appDidEnterForeground()`에서 정책 재조회(10분 스로틀, 강제 화면에서는 스로틀 없음)
7. 426 처리: `NetworkError.updateRequired` 추가 → `AppState`가 강제 상태로 전환
8. `#if DEBUG`에서 게이트 비활성
9. 화면은 `DS*` 컴포넌트만 사용

### Phase 6. 측정 (선택)

- `mobile_app_event`에 이벤트 타입 추가: `update_prompt_shown`, `update_prompt_clicked` (force/recommend 구분)
- `app_client_builds`의 최근 확인일로 "아직 사용 중인 구빌드"를 관리자 화면에 표시

## 6. 사용자 안내 UX

### 6.1 권장 업데이트 노출 규칙

원칙: 권장은 사용자의 흐름을 막지 않는다. 몇 번 거절하면 더 조르지 않고, 눈에 덜 띄는 안내로 바꾼다.

| 항목 | 규칙 |
|---|---|
| 노출 시점 | 앱 실행 후 **메인(피드) 화면에 도착했을 때**. 가입, 로그인, 글/메시지 작성 중에는 띄우지 않는다 |
| 형태 | Android: Play `FLEXIBLE` (백그라운드 다운로드 후 재시작 스낵바). iOS: 닫을 수 있는 바텀시트 ("업데이트" / "나중에") |
| 재노출 간격 | "나중에"를 누르면 **3일** 뒤 다음 실행 때 다시 노출 |
| 최대 횟수 | 같은 권장 빌드 기준 **3회**까지. 그 뒤로는 팝업 대신 마이페이지 앱 정보에 "새 버전 있음" 표시만 남긴다 |
| 초기화 | 관리자가 `recommendedBuild`를 바꾸면 횟수와 간격을 초기화 |
| 세션당 | 한 번 실행하는 동안 최대 1회 |
| 저장 | 기기 로컬: 마지막 노출 시각, 노출 횟수, 기준 `recommendedBuild` |

- 3일/3회는 앱 상수(`RECOMMEND_PROMPT_INTERVAL_DAYS`, `RECOMMEND_PROMPT_MAX_COUNT`)로 두고, 운영 중 조정이 필요해질 때 정책 JSON으로 옮긴다
- 근거: 매 실행마다 뜨는 권장 팝업은 무시하는 습관만 만든다. 3회 안에 업데이트하지 않은 사용자에게 필요한 건 강제 업데이트다

### 6.2 OS 미지원 사용자 안내

**언제 생기나.** 현재 최소 지원 OS(iOS 17, Android API 26)보다 낮은 기기는 스토어에서 앱을 설치할 수 없다. 이 상태는 **나중에 새 빌드에서 최소 OS를 올릴 때** 생긴다. 기존 OS에 남은 사용자는 구빌드를 계속 쓰지만 스토어가 새 빌드를 주지 않는다.

**원칙: 차단하지 않는다.** 사용자가 앱 안에서 해결할 방법이 없는데 막으면 앱을 영영 못 쓰게 된다. 그래서 판정 규칙에서 OS 미지원을 강제보다 먼저 확인하고, 서버 426에서도 제외한다.

| 항목 | 규칙 |
|---|---|
| 형태 | 닫을 수 있는 1회성 다이얼로그 |
| 노출 시점 | 메인 화면 도착 시. 강제/권장 안내와 겹치면 이 안내만 보여준다 |
| 빈도 | `minOsVersion` 값마다 1회. 값이 바뀌면 다시 1회 |
| 상시 표시 | 마이페이지 앱 정보에 "이 기기에서는 최신 버전을 받을 수 없어요" 표시 |
| 버튼 | "확인" 하나 |

문구 예시:

- **iOS**: "이 기기의 iOS 버전에서는 새 업데이트를 받을 수 없어요. 설정 > 일반 > 소프트웨어 업데이트에서 iOS 17 이상으로 업데이트하면 최신 버전을 사용할 수 있어요. 지금 버전은 계속 사용할 수 있어요."
- **Android**: "이 기기의 Android 버전에서는 새 업데이트를 받을 수 없어요. 기기에서 시스템 업데이트가 가능하다면 업데이트 후 최신 버전을 사용할 수 있어요. 지금 버전은 계속 사용할 수 있어요."

주의:

- iOS에서 "소프트웨어 업데이트" 화면으로 바로 보내는 URL(`App-prefs:` 등)은 비공개 API라 심사 거절 사유가 된다. **경로를 문구로만 안내한다**
- Android는 제조사/기기에 따라 OS 업데이트 자체가 불가능한 경우가 많아서 "가능하다면"으로 표현한다. 시스템 업데이트 화면으로 가는 표준 Intent가 없으므로 버튼을 두지 않는다
- 향후 API 호환이 깨져 구빌드를 정말 끊어야 하는 상황이 오면, 이 규칙을 유지한 채 별도의 "지원 종료" 정책을 추가한다. 이번 범위에는 넣지 않는다

## 7. 반영 시간

| 동작 | 서버 차단(426) | 앱 화면 |
|---|---|---|
| 관리자가 강제 켜기/기준 상향 | 저장 즉시 (캐시 무효화) | 다음 API 요청(426)에서 즉시, 또는 다음 정책 재조회 |
| 관리자가 강제 끄기 | 저장 즉시 | 강제 화면이 포그라운드 복귀/재조회 시 해제 |

## 8.1 마이그레이션 승인 (배포 전 필수)

이 저장소는 마이그레이션 추가 시 승인 다이제스트를 갱신해야 한다. Phase 1에서 다음을 갱신했다.

- `backend/migrations/testdata/canonical_identity_candidate_lineage.sha256`에 071~073 항목 추가
- 새 매니페스트 다이제스트: `5b5ec0c5c29ab4dd38d5326165303446a2903a0b47f8b937d6e7bb9414642ee7`
- 이 값을 반영한 곳: `backend/.env.example` 주석, `internal/contract/migration_source_approval_test.go`(승인 상수, `future_migrations=34`)

**배포 담당자 확인 사항**: 운영에 적용할 때 `CANONICAL_CANDIDATE_MANIFEST_SHA256`를 위 값으로 설정해야 한다. 이 다이제스트는 "사람이 마이그레이션 내용을 검토했다"는 표시이므로, 적용 전에 071~073 SQL을 직접 확인하기 바란다.

## 8. 첫 출시 전략 (결정됨)

이 앱은 한 번도 출시된 적이 없다. 그래서 **게이트 없는 구버전을 아예 만들지 않는 선택**이 가능하고, 그렇게 하기로 했다 (D15, D16).

- 현재 심사를 통과한 빌드는 **출시하지 않고 보류**한다. iOS "수동 출시", Android "관리형 게시" 설정 완료
- Phase 1~5를 모두 끝낸 뒤, 게이트가 들어간 빌드를 **첫 공개 릴리스**로 낸다
- 결과: 모든 사용자가 처음부터 게이트 탑재 빌드를 쓴다. User-Agent 기반 구버전 차단 같은 우회책은 필요 없다

주의:

- iOS에서 이미 승인된 버전에 다른 빌드를 넣으려면, 승인된 빌드를 **개발자 거부 처리하고 새 빌드로 다시 심사**를 받아야 한다. 지금 받은 승인은 재사용되지 않는다
- Android도 관리형 게시로 멈춰 둔 릴리스를 대체하려면 새 릴리스가 다시 심사를 거친다
- 즉 출시 전 마지막 제출은 "게이트 포함 빌드" 한 번이어야 한다. 그 전에 O1(iOS 숫자 App ID)을 확보해 `storeUrl`을 채워 둔다

## 9. 출시 순서

1. Phase 1 백엔드 + Phase 2 관리자 화면 배포 (모든 정책 꺼짐)
2. Phase 3~5 완료 후 게이트 포함 빌드를 제출한다. **단계적 출시를 쓰지 않고 100% 배포로 공개한다** (D17)
3. 공개된 첫 빌드 번호를 플랫폼별로 기록한다. 이 값이 이후 모든 기준의 출발점이다
4. 출시 시점에는 정책을 꺼둔 채로 둔다. 필요해질 때 관리자 화면에서 플랫폼별로 켠다

## 10. 운영 규칙 (런북에 옮길 내용)

- **단계적 출시를 쓰지 않는다** (D17). 매 릴리스를 100% 배포로 낸다
- **최소 빌드는 해당 플랫폼 스토어에 100% 배포가 끝난 뒤에만 올린다.** iOS 단계적 출시, Android 단계적 출시가 진행 중이면 대상이 아닌 사용자는 받을 업데이트가 없는데 차단된다
- 플랫폼별로 따로 올린다. 한쪽 심사가 늦어져도 다른 쪽에 영향이 없다
- 최소 지원 OS(`deploymentTarget`, `minSdk`)를 올린 빌드를 출시하면 해당 플랫폼 정책의 `minOsVersion`도 같이 올린다
- 긴급 롤백: 관리자 화면에서 해당 플랫폼의 강제 업데이트를 끈다. 서버 차단은 즉시 풀린다
- 활성화 전 검증: iOS는 TestFlight 빌드로, Android는 내부 앱 공유(Internal App Sharing)로 강제/권장 화면을 확인한다

## 11. 완료 기준

- [ ] 관리자 화면에서 iOS와 Android의 강제/권장 활성화 여부와 빌드 기준을 각각 저장할 수 있다
- [ ] iOS 정책을 바꿔도 Android 판정과 화면에 영향이 없다 (반대도 동일)
- [ ] 검증 규칙을 위반한 저장은 필드별 사유와 함께 거부된다
- [ ] 강제 활성화/기준 상향 시 확인 다이얼로그가 뜨고, 변경 이력이 기록된다
- [ ] 기존 "앱 설정" 화면과 API로는 정책을 수정할 수 없다
- [ ] 모든 정책이 꺼져 있을 때 웹과 앱 동작이 기존과 같다
- [ ] 강제를 켜면 두 앱 모두 로그인 전/후 어느 화면에서든 강제 화면으로 전환되고, 끄면 해제된다
- [ ] 앱 게이트를 우회해도(예: 켜둔 세션) 다음 API 요청에서 426으로 차단된다
- [ ] 권장 안내는 메인 화면에서만, 3일 간격으로 최대 3회 뜨고, 이후 마이페이지 표시로 바뀐다. 권장 빌드가 바뀌면 초기화된다
- [ ] OS 미지원 기기는 차단되지 않고, `minOsVersion` 값마다 1회 안내를 받는다
- [ ] 정책 조회 실패와 서버 장애 시 앱이 차단되지 않는다
- [ ] 디버그 빌드는 게이트에 걸리지 않는다
- [ ] 백엔드 `go test ./...`, 관리자 `npm run lint` + 테스트, Android 단위 테스트, iOS XCTest 통과
