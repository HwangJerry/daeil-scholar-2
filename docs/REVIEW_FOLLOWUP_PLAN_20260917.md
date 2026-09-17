# 리뷰 미해결 지적 대응 계획 (2026-09-17)

- 출처: 강제 업데이트 작업 중 실행한 `/code-review high` 2회 (Android 저장소, iOS 저장소)
- 대상: 강제 업데이트 기능이 아닌 **다른 작업에서 나온 지적**. 강제 업데이트 관련 지적은 모두 처리 완료(`APP_FORCE_UPDATE_PLAN.md` 참고)
- 이 문서는 지적별 원인·수정 방향·검증 방법을 담는다. 아직 착수하지 않았다

## 0. 요약

| # | 지적 | 심각도 | 저장소 | 상태 |
|---|---|---|---|---|
| 1 | 인증 문자 발송 후 번호를 바꿔도 "인증 완료"로 표시 | 높음 | swift | 착수 전 |
| 2 | 인증된 번호 수정 시 제출이 조용히 막힘 | 중간 | swift | 착수 전 |
| 3 | 동시 API 요청이 호스트당 5개로 제한 | 중간 | kotlin | 착수 전 |
| 4 | **토큰 회전 원자성이 깨져 조용한 강제 로그아웃** (리뷰의 "대기자 동반 취소"는 이 문제의 증상 하나) | **높음** | kotlin + backend | 착수 전 |
| 5 | 계정 삭제 화면 `imePadding` 이중 적용 | 중간 | kotlin | 착수 전 |
| 6 | `windowLightNavigationBar` v27 대응 누락 | 낮음 | kotlin | **오탐 — 대응 불필요** |

권장 순서: **4-4의 1~3단계**(비용이 거의 없고 위험이 큼) → **1 → 2**(가입 흐름을 막으므로 출시 전 필수) → **5**(작고 눈에 띔) → **3** → **4-4의 4~6단계**(설계 변경·서버 변경).

1과 2는 같은 화면에서 이어지는 문제라 **한 사람이 묶어서** 하는 편이 낫다. 3~5는 서로 독립적이라 병렬 가능하다.

## 1. 인증 문자 발송 후 번호를 바꿔도 "인증 완료" (높음)

**증상**: A번호로 인증번호를 받고, 입력 전에 전화번호 필드를 B로 고친 뒤 인증번호를 입력하면 앱이 "휴대폰 인증이 완료되었습니다"를 표시하고 제출을 허용한다. 서버는 그 토큰이 A의 것이라 `PHONE_NOT_VERIFIED`로 거절한다. 사용자는 인증을 마친 화면에서 원인 모를 실패를 겪는다.

**원인** (`Sources/App/Feature/Login/PhoneVerificationController.swift`)

- `:37` `invalidateIfPhoneChanged`가 `guard !verifiedPhone.isEmpty`로 시작한다. `verifiedPhone`은 **인증 확인 성공 후에만** 채워지므로, 발송~확인 사이에는 번호를 바꿔도 아무 일도 일어나지 않는다. `verificationId`는 여전히 A의 것이다
- `:74` `confirmCode(phone:)`이 성공 시 `verifiedPhone = Self.digits(phone)`으로 **폼의 현재 값(B)**을 기록한다. 서버 토큰은 A를 증명하는데 앱은 B가 인증됐다고 믿는다

**보안 영향은 없음**: 서버가 토큰에 묶인 번호와 제출 번호를 비교한다(`backend/internal/service/phone_verification_service.go:165-175`, `AssertPhoneVerified` → `grantSubject`). 잘못된 번호로 가입이 뚫리지는 않는다. 실제 피해는 **막다른 길**이다.

**수정 방향**

1. 발송 시점의 번호를 기억한다: `requestCode`(`:51`) 성공 시 `requestedPhone = Self.digits(phone)`을 저장
2. `invalidateIfPhoneChanged`의 가드를 `verifiedPhone` 대신 **"추적 중인 번호"**(`verifiedPhone`이 있으면 그것, 없으면 `requestedPhone`) 기준으로 바꾼다. 어느 쪽이든 현재 폼 값과 다르면 `reset()`
3. `confirmCode`는 폼 값이 아니라 **문자를 보낸 번호**를 기준으로 동작한다. 폼 값이 달라졌다면 확인을 시도하지 않고 재발송을 요구한다
4. `reset()`에 `requestedPhone` 초기화를 추가한다

**검증**

- `Tests/DflhSafV2SwiftTests/`에 `PhoneVerificationControllerTests` 신설. 최소 4개: 발송 후 번호 변경 시 상태 초기화 / 변경 없이 확인하면 인증 완료 / 인증 후 번호 변경 시 토큰 무효화 / 번호가 바뀐 상태에서는 `isVerified`가 false
- 기존 `AuthSecurityTests`가 이 컨트롤러를 건드리는지 확인
- `xcodebuild build` + `-only-testing:DflhSafV2SwiftTests`

**주의**: `LoginView`(소셜 가입)와 `NativeSignUpView`(아이디/비밀번호 가입)가 같은 컨트롤러를 쓴다. 두 화면 모두에서 확인한다.

## 2. 인증된 번호 수정 시 제출이 조용히 막힘 (중간)

**증상**: 인증을 마친 뒤 전화번호를 고치면 컨트롤러는 초기화되는데 단계형 폼의 "휴대폰 인증번호" 단계는 `completed`로 남는다. 검토 화면에서 제출 버튼이 비활성인데 오류 메시지도 없고 인증 단계로 돌아갈 경로도 없다.

**원인**: `Sources/App/Feature/Login/NativeSignUpView.swift:84`의 "전화번호" `DSStepField`에 `invalidates`가 없다. 같은 파일 `:81`의 비밀번호 필드는 `invalidates: [2]`로 "비밀번호 확인" 단계를 되돌린다 — 이미 있는 패턴을 전화번호에만 적용하지 않았다. `LoginView.swift:362`도 동일.

**수정 방향**

1. 전화번호 필드에 `invalidates: [<휴대폰 인증번호 단계 인덱스>]`를 추가한다. `NativeSignUpView`의 현재 배열 순서는 0 아이디 / 1 비밀번호 / 2 비밀번호 확인 / 3 이름 / 4 전화번호 / 5 `phoneVerificationField`이므로 `[5]`. **인덱스는 하드코딩 전에 배열을 다시 세어 확인한다**(1번 작업으로 순서가 바뀔 수 있음)
2. `LoginView.swift:362`의 단계 배열도 같은 방식으로 확인해 적용한다
3. 인덱스를 숫자로 흩뿌리지 말고 상수로 뽑는 편이 안전한지 검토한다

**검증**: 1번과 같은 테스트 타깃에서, 인증 완료 후 번호를 고치면 인증 단계가 미완료로 돌아가고 제출이 막히되 **이유가 표시되는지** 확인한다.

## 3. 동시 API 요청이 호스트당 5개로 제한 (중간)

**증상**: 로그인 직후 첫 화면에서 피드·뱃지·프로필·동문 필터·기부 현황·배너가 동시에 요청되는데, 6번째부터는 앞선 요청이 끝나야 시작된다. 첫 렌더가 직렬화되어 느려진다.

**원인**: `app/src/main/kotlin/com/dflh/app/core/network/DflhApiClient.kt:409` `executeOnce`가 `call.enqueue(...)`를 쓴다(`23cbcaa`에서 동기 `execute()`에서 전환, 로그아웃 시 진행 중 요청 취소 목적). OkHttp `Dispatcher`는 **비동기 호출에만** `maxRequests`(64)/`maxRequestsPerHost`(5)를 적용한다. 동기 호출에는 제한이 없었다. 이 앱은 모든 요청이 한 호스트로 가고, `AppContainer.kt:69`가 `newBuilder()`로 파생시켜 **Dispatcher를 공유**한다.

**수정 방향**

1. `enqueue` 자체는 유지한다. 취소 전파(`invokeOnCancellation { call.cancel() }`)가 이 전환의 목적이고, 되돌리면 로그아웃 시 요청 취소가 깨진다
2. `AppContainer`에서 공유 클라이언트를 만들 때 `Dispatcher`의 `maxRequestsPerHost`를 올린다. 첫 화면 동시 요청 수(측정 후 결정, 대략 10~16)를 근거로 값을 정하고 **왜 그 값인지 주석**을 남긴다
3. 서버가 1코어/1GB 인스턴스임을 감안한다(`dflh-saf-v2/CLAUDE.md`). 무제한으로 열지 말고 상한을 둔다

**검증**

- 단위 테스트로는 잡기 어렵다. 로그인 직후 요청 타임라인을 로그로 확인하거나, `Dispatcher` 설정값을 검증하는 테스트를 둔다
- 서버 부하가 늘어나는 변경이므로 배포 후 응답 시간을 관찰한다

## 4. 토큰 회전 원자성이 깨져 조용한 강제 로그아웃 (높음)

리뷰는 "선행 코루틴 취소가 대기자까지 취소"(무한 스피너)로 보고했으나, 2026-09-17 추가 조사에서 **본체는 더 넓고 심각한 문제**로 드러났다. 무한 스피너는 증상 하나일 뿐이고, 같은 원인이 **원인 불명의 강제 로그아웃**을 만든다.

### 4-1. 왜 실패 비용이 로그아웃인가

서버 정책(`backend/internal/repository/auth_repo.go:554-616`):

- 갱신 토큰은 **일회용 회전**. `FOR UPDATE` + `CONSUMED_AT IS NULL`로 동시성을 막는다
- 재사용이 감지되면 **그 토큰만이 아니라 `USR_SEQ` + `MRT_SID`의 살아 있는 토큰 전부를 폐기**한다(`:579-583`). 방금 발급한 후속 토큰도 포함된다
- **유예 창이 없다.** 중복 요청 허용치가 0이다

따라서 중복 갱신 한 번이 세션을 죽인다.

1. A가 옛 토큰 소비, 새 토큰 T2 발급 (성공)
2. B가 같은 옛 토큰 제시 → 재사용 판정
3. **T2를 포함해 세션 전체 폐기**
4. A는 T2를 들고 있지만 이미 죽은 토큰 → 다음 요청에서 로그아웃

**합격 기준은 "스피너가 안 뜬다"가 아니다. 지켜야 할 불변식은 "회전 1회당 서버에 도달하는 요청이 정확히 1개"** 이며, 모든 화면 전환·프로세스 상태·진입 경로에 걸쳐 성립해야 한다.

### 4-2. 회전이 끊기는 갈래

**(a) 코루틴 취소 — 지금 가장 잦다**

`TokenSessionManager.refresh()`(`:90-108`)의 꼬리에 보호가 없다.

```kotlin
val refreshed = authApiProvider().refresh(refreshToken)   // suspend = 취소 지점
if (tokenStore.load() != latest) throw SessionEndedException()
tokenParser.decode(...)
tokenStore.save(refreshed)                                 // 여기까지 와야 보존
```

네트워크 호출 중 취소가 오면 **서버는 응답을 보냈는데 코루틴은 재개 지점에서 취소로 끝난다.** `save()`가 실행되지 않아 서버만 회전한 상태가 된다. 다음 갱신 시도는 재사용으로 판정된다.

공유 갱신이 **가장 먼저 도착한 호출자의 코루틴**에서 실행되므로(`:73-76`), 그 화면이 사라지면 이 취소가 일어난다. 단일 Activity + Compose 구조라 화면 전환이 잦아 조건이 잘 갖춰진다.

**(b) 프로세스 사망**

앱이 꺼져 있어도 푸시가 오면 OS가 프로세스를 띄우고 `DflhFirebaseMessagingService`만 실행한다. Activity도 ViewModel도 없는 저우선순위 상태다. 이때 `onNewToken`이 인증 호출을 한다(`PushApi(authenticatedApiClient)`, `AppContainer.kt:149-150`). 갱신 중 프로세스가 죽으면 회전이 끊긴다.

**매니저 스코프 분리로는 못 막는다. 코루틴이 아니라 JVM이 사라진다.** 그리고 푸시만의 문제가 아니다 — 앱 강제 종료, OOM, 네트워크 유실에서 동일하게 발생한다. 푸시는 확률이 높은 한 갈래일 뿐이다.

**(c) 디스크 미기록**

`TokenStore.save()`가 `apply()`를 쓴다(`core/auth/TokenStore.kt:38-42`). 메모리에는 즉시 반영되지만 디스크 쓰기는 미뤄진다. 정상 종료해도 디스크에 닿기 전에 죽으면 같은 결과다.

**(d) 실패 후 재시도는 언제나 같은 토큰**

갱신이 실패하면 `finally`가 `refreshInFlight`를 비우지만 **저장소에는 옛 토큰이 그대로 남는다.** 다음 호출자는 그 토큰으로 다시 시도한다. 응답 유실 계열에서는 이 재시도가 곧 재사용 판정이다.

### 4-3. 심각도를 올려 잡는 근거

- 영향이 **강제 로그아웃**이다. 로딩 실패가 아니다
- 재현이 거의 불가능하다. 문의를 받아도 원인을 특정하기 어렵다
- 서버 로그에서 **진짜 탈취와 구분되지 않는다.** 재사용 탐지 지표가 오염되어, 실제 공격이 섞여도 알아채기 어렵다
- 지금은 대기자들이 함께 취소되어 아무도 재시도하지 않기 때문에 **증상이 덜 드러났을 가능성**이 있다. (a)를 고쳐 실패 전파가 정상화되면 스피너가 로그아웃으로 바뀌어 보일 수 있다 — 이는 악화가 아니라 은폐되던 문제의 표면화다

### 4-4. 대응 단계

| 순서 | 조치 | 비용 | 덮는 범위 |
|---|---|---|---|
| 1 | **푸시 경로 지연 전송** — `onNewToken`은 로컬 저장만 하고(이미 `pushTokenStore.saveToken` 호출 중), 전송은 앱이 포그라운드일 때 | 아주 작음 | (b)의 가장 잦은 갈래를 제거. FCM 토큰 등록은 몇 시간 늦어도 손해가 없는데 지금은 그 대가로 세션을 걸고 있다 |
| 2 | **응답~저장 구간 `NonCancellable`** — 서버 응답을 받은 뒤 `save()`까지는 취소되지 않게 | 아주 작음 | (a) 전부 |
| 3 | **`TokenStore.save`를 `commit()`으로** | 아주 작음 | (c) |
| 4 | **매니저 스코프 분리** — `AppContainer.kt:48`의 `applicationScope`를 주입해 공유 갱신이 화면 수명과 무관해지도록. 새 스코프를 만들지 말 것 | 중간 | 무한 스피너, (a)의 잔여 |
| 5 | **회전 시도 영속 표식** — 요청 직전에 표식을 남기고 완료 시 제거 | 중간 | 빈도 계측, 조용한 로그아웃 탐지 |
| 6 | **백엔드 유예 창** — `ROTATED_TO_JTI`로 멱등 재발급: 소비된 토큰이 N초 이내 재제시되고 후속 토큰이 아직 미사용·미폐기면, 세션을 끊는 대신 그 후속 토큰을 반환 | 중간 | **(a)(b)(c)(d) 전부. 유일한 근본 해결** |

1~3은 위험이 낮고 즉시 가능하다. 6은 클라이언트가 어떤 이유로 죽든 재시도를 무해하게 만드는 유일한 방법이며, 사슬(`ROTATED_TO_JTI`)이 이미 저장되어 있어 구현 여지가 있다. **탈취 탐지는 창 바깥에서 그대로 유지된다** — 후속 토큰이 이미 사용됐다면 지금처럼 세션을 폐기한다.

### 4-5. 코얼레싱 불변조건 (점검 완료)

코얼레싱(`refreshMutex` + `refreshInFlight`)은 최적화가 아니라 **정확성**이다. 중복 허용치가 0이므로, 여섯 화면이 각자 갱신하면 하나만 성공하고 나머지가 재사용으로 판정되어 세션이 끊긴다.

2026-09-17 점검 결과, 아래 세 조건이 모두 충족된다. 하나라도 깨지면 Mutex는 무의미해진다.

| 조건 | 상태 | 근거 |
|---|---|---|
| `TokenSessionManager`가 애플리케이션 수명 단일 인스턴스 | 충족 | 프로덕션 생성 지점은 `AppContainer.kt:82` 한 곳, `AppContainer`는 `DflhApplication.kt:15`에서 한 번만 생성. DI 프레임워크 없는 수동 컨테이너라 스코프 오설정 여지가 없다 |
| 단일 프로세스 | 충족 | `AndroidManifest.xml`에 `android:process` 없음. 컴포넌트는 단일 Activity + `DflhFirebaseMessagingService` 하나이며, 서비스도 같은 컨테이너를 사용한다. `WorkManager`/`CoroutineWorker` 사용처 없음 |
| 갱신 진입점이 모두 같은 매니저를 경유 | 충족 | 선제적 `authorizationHeader()`와 401 경로 `refreshAuthorizationHeader()` 모두 `refreshIfPossible()`로 합류(`AppContainer.kt:92,94`). 실시간 스트림도 같은 refresher를 탄다(`MessageRealtimeClient.kt:76`). OkHttp `Authenticator` 미사용이라 OkHttp 내부 스레드에서 도는 우회 경로가 없다 |

**작업 시 지킬 것**

1. 매니저는 계속 컨테이너 소유 단일 인스턴스로 둔다. 스코프를 만들려고 매니저를 밖에서 새로 생성하지 않는다
2. 두 진입점이 같은 `refreshInFlight`를 공유하는지 테스트로 고정한다
3. 회귀 테스트에 "동시 401 다발 → 서버 도달 갱신 요청 1회"를 넣는다(`sharedRefreshWaitersReceiveRefreshedTokenWhenStoreStillMatches` 확장)
4. 저장이 끝나기 전에는 어떤 경로로도 취소되지 않음을 테스트한다

**앞으로의 설계 제약**

- 위젯, `android:process`로 분리된 서비스, 별도 프로세스 `WorkManager`를 추가하면 **다른 JVM이 되어 Mutex가 통하지 않는다.** 두 프로세스가 각자 갱신하면 서버가 재사용으로 판정해 세션을 끊는다
- 그런 컴포넌트가 필요하면 갱신을 메인 프로세스에 위임(IPC)하거나, 백엔드 유예 창(4-4의 6)을 먼저 도입해야 한다
- 이 연결고리는 코드만 봐서는 드러나지 않는다. 별도 프로세스 컴포넌트를 검토할 때 이 문서를 확인할 것

## 5. 계정 삭제 화면 `imePadding` 이중 적용 (중간)

**증상**: `DeletionPage.Receipts`에서 확인번호 입력란을 탭하면 본문이 키보드 높이의 **두 배**만큼 줄어든다. 버튼 위에 키보드만 한 빈 띠가 생기고 입력란이 의도보다 더 밀려난다.

**원인**: `app/src/main/kotlin/com/dflh/app/feature/accountdeletion/AccountDeletionScreen.kt`

- `:76` `bottomBar`가 `.navigationBarsPadding().imePadding()`을 적용한다. 키보드가 올라오면 bottomBar 높이가 그만큼 커지고, `Scaffold`는 그 높이를 본문 `padding`으로 돌려준다
- `:85` 본문이 `.padding(padding).imePadding()`으로 **한 번 더** 적용한다

**수정 방향**: 본문의 `.imePadding()`을 제거한다(`bottomBar` 쪽을 남긴다). `layout.isShortHeight`로 bottomBar가 없는 경로가 있으므로(`:75`), 그 경우 본문이 키보드에 가리지 않는지 함께 확인한다.

**검증**: 단위 테스트로 잡기 어렵다. 기기에서 `Receipts` 페이지 입력란 포커스 시 레이아웃을 확인한다. `isShortHeight` 경로도 함께 본다.

## 6. `windowLightNavigationBar` v27 대응 누락 — 오탐

리뷰는 "`values/styles.xml:7`에서 `android:windowLightNavigationBar`가 제거됐는데 `values-v27/styles.xml`이 없다"고 지적했으나, **해당 파일은 이미 존재한다**.

- `app/src/main/res/values-v27/styles.xml`에 `AppTheme`이 정의되어 있고 `android:windowLightNavigationBar`가 `true`다
- 커밋 `cfb9823`("add dark mode color scheme infrastructure")에서 추가되어 추적 중이다
- 이 속성은 API 27에서 추가되었으므로 `values/`(27 미만)에서는 no-op이다. 기본 리소스에서 빼고 `values-v27`에서 지정하는 현재 형태가 올바르다

**대응 불필요.** 리뷰 결과를 그대로 티켓으로 옮기지 말 것.
