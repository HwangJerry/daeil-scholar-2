# 가입 동의 체크 UI 및 전화번호 수집 목적 고지 작업 계획 (2026-09-23)

- 출처: 2026-09-15 App Store 리젹(5.1.1(v) 전화번호 필수) 대응. 회신 자료는 `dflh-saf-v2-swift/docs/app-review/2026-09-15/`.
- 결정(2026-09-23): 전화번호는 **필수 유지**. 이유는 동문 명부 대조 시 동명이인 구분과 SMS 본인 확인. 대응은 기능 축소가 아니라 **목적 고지**.
- 고지 방식: 필드 옆 장문 설명이 아니라 **[필수] 개인정보 수집·이용 동의 체크 1개 + 동의 표(항목·목적·보유기간)**. 국내 서비스 관행과 일치(`PHONE_FIELD_COPY_RESEARCH_KO.md`).
- 이 문서는 착수 전 계획이다. 백엔드 심사관 우회 번호(`de13b04`)는 이미 커밋됐다.

## 0. 요약

| # | 작업 | 저장소 | 의존 | 상태 |
|---|---|---|---|---|
| T1 | 동의 표 문구·버전 확정 | docs | 없음 | **문구 확정(2026-09-23)**, 버전 값은 T2 게시일로 결정 |
| T2 | 처리방침 수정안 반영 (`policyContent.ts`, 갱신일) | dflh-saf-v2 frontend | T1 | 착수 전 |
| T3 | 백엔드: 가입·소셜 연동 요청에 동의 필드 수신, `AUTH_CONSENT` 기록 | dflh-saf-v2 backend | T1 | **1단계 구현 완료(2026-09-23)**. 2단계는 `PRIVACY_CONSENT_ENFORCE=true` 전환만 남음 |
| T4 | iOS: 가입 2화면에 동의 체크·동의 표 시트, 전화번호 hint | dflh-saf-v2-swift | T1, T3 | 착수 전 |
| T5 | 웹: 가입·소셜 연동 폼에 동의 체크·모달, 인증 안내 문구 | dflh-saf-v2 frontend | T1, T3 | 착수 전 |
| T6 | Android: 가입 화면에 **휴대폰 인증 자체가 없음** → OTP + 동의 체크 | dflh-saf-v2-kotlin | T1, T3 | 착수 전, 별도 규모 산정 |
| T7 | 심사 재제출: 회신문 문장 수정, 스크린샷, 빌드 교체, 우회 번호 env 배포, App Privacy 라벨 | swift docs, 운영 | T2, T4 | 착수 전 |

권장 순서: **T1 → T3 → T4 ∥ T5 → T2 → T7**. T6은 iOS 재제출을 막지 않으므로 병행하되, Android 출시 전에는 반드시 끝나야 한다.

## 1. 현황 (코드로 확인)

- iOS 일반 가입(`NativeSignUpView.swift`)과 소셜 연동 가입(`LoginView.swift` 소셜 폼) 모두 전화번호·인증번호 단계는 있으나 **약관·개인정보 동의 체크가 없다**. 웹 `RegisterForm.tsx`, `AccountLinkNewForm.tsx`도 같다.
- 가입 폼은 `DSReverseStepForm`(필드 배열 + submit) 구조라 체크박스 필드 타입이 없다. 컴포넌트 확장이 필요하다.
- 웹에는 `/privacy`, `/support`만 있고 **이용약관 페이지가 없다**. iOS는 내정보 화면에서 `/privacy`로 링크한다.
- 백엔드 `AUTH_CONSENT` 테이블(마이그레이션 041, TERMS/PRIVACY/MARKETING, 버전·수락 시각)이 이미 있으나 **계정 삭제 단계에서만 참조**되고 가입 시 기록하지 않는다.
- `RegisterRequest`, 소셜 연동 요청 모델에 동의 필드가 없다.
- Android `NativeSignUpViewModel.kt`는 전화번호를 받지만 인증 요청·토큰 전송 코드가 없다. 백엔드가 인증 토큰을 강제하므로 **현재 Android 네이티브 가입은 서버에서 거부된다.**

## 2. T1. 동의 표 문구·버전

동의 항목은 **개인정보 수집·이용 동의 1개(필수)** 로 한다. 이용약관 문서가 없으므로 이번 범위에서 약관 동의는 만들지 않는다(추가 시 별도 작업).

동의 표 초안(처리방침 수정안과 동일 논리, `PRIVACY_POLICY_PHONE_REVIEW_KO.md` §3):

| 구분 | 항목 | 목적 | 보유기간 |
|---|---|---|---|
| 필수 | 아이디·비밀번호 또는 소셜 계정 식별자, 이름, 전화번호, 이메일, 기수, 학과, 졸업연도 | 본인 확인, 회원 식별, 중복 가입 방지, 동문 자격 확인·가입 승인, 공지 안내 | 회원 탈퇴 시 삭제(법령 보존 항목 제외) |
| 필수 | 문자 인증번호, 인증 요청·확인 시각 | 입력한 전화번호의 본인 소유 확인 | 인증 후 24시간 |
| 선택 | 사진, 소속, 직책, 업종, 소개, 명함, 연락처 공개 설정 | 동문 검색·프로필 표시 | 회원 탈퇴 또는 삭제 시 |

- 거부권 문구: "동의를 거부할 수 있으나, 필수 항목에 동의하지 않으면 동문 확인이 불가하여 가입할 수 없습니다."
- 안심 문구: "전화번호는 본인이 공개로 설정하지 않으면 다른 동문에게 보이지 않습니다. 광고에 사용하지 않습니다."
- 체크박스 라벨: "[필수] 개인정보 수집·이용에 동의합니다" + "내용 보기".
- 전화번호 필드 hint(짧게): "인증번호가 문자로 발송됩니다."
- 동의 버전: `CONSENT_VERSION = 처리방침 갱신일`(예: `2026-09-XX`). T2 게시일과 같은 값을 클라이언트·서버가 공유한다.
- 소스 오브 트루스: 문구는 처리방침(`policyContent.ts`)에서 파생하고, 동의 표 전용 상수를 `frontend/src/domains/privacy/consentContent.ts`로 분리한다. iOS·Android는 같은 문구를 각자 리소스로 복제한다(웹뷰 대신 네이티브 시트, 오프라인·심사 스크린샷 고려).

2026-09-23 사용자 결정: 첫 행 목적은 관용어로 짧게, 보유기간 초안 유지, 소셜 가입을 위해 항목에 "또는 소셜 계정 식별자" 포함. 위 표가 확정본이다. 남은 것은 버전 값(T2 게시일)만이다.

## 3. T2. 처리방침 반영

- `PRIVACY_POLICY_PHONE_REVIEW_KO.md` §3-1~3-4 수정안을 `policyContent.ts`, `PrivacyPolicyPage.tsx`에 적용.
- `PRIVACY_UPDATED_AT`을 게시일로 변경. 이 값이 T1 동의 버전이 된다.
- 문자 발송 업체 표기(네이버클라우드)와 운영 `SMS_PROVIDER` 값 일치 확인.
- 검증: `frontend/tests/privacy-policy.spec.ts` 갱신, 공개 URL 확인.

## 4. T3. 백엔드 동의 기록

- `RegisterRequest`, 소셜 연동 요청에 `privacyConsent: { version: string, accepted: bool }` 추가.
- 가입·연동 성공 후 `AUTH_CONSENT`에 `PRIVACY / version / IS_REQUIRED=1 / IS_ACCEPTED=1 / ACCEPTED_AT` 1행 기록. 계정 ID 매핑은 041 스키마의 canonical account 기준을 따른다(기존 `account_erasure_steps.go` 참조).
- **호환성 단계**: 기존 TestFlight·Android 빌드는 동의 필드를 보내지 않는다. 1단계는 필드가 없으면 경고 로그만 남기고 통과, 2단계(강제 업데이트 후, `APP_FORCE_UPDATE_PLAN.md` 참고)에 `accepted != true` 또는 버전 불일치를 400으로 거부.
- 서버가 허용하는 현재 버전은 환경변수 또는 상수 `PRIVACY_CONSENT_VERSION`으로 두고, 클라이언트가 보낸 버전이 다르면 2단계에서 거부.
- 검증: 핸들러 테스트(필드 없음/거짓/버전 불일치/정상), 기록 행 확인, 계정 삭제 단계가 새 행을 지우는지 확인.

구현 결과(2026-09-23):
- 요청 필드: `privacyConsent: { "version": "<처리방침 게시일>", "accepted": true }`. `POST /api/auth/register`, `POST /api/auth/social/link` 공통. 응답 코드 `CONSENT_REQUIRED`(400, 미동의 또는 강제 단계에서 누락), `CONSENT_VERSION_OUTDATED`(400, 강제 단계에서 버전 불일치).
- 환경변수: `PRIVACY_CONSENT_VERSION`(비우면 버전 검사 생략), `PRIVACY_CONSENT_ENFORCE`(기본 false).
- 정책: `accepted:false`는 단계와 무관하게 거부. 필드 누락·버전 불일치는 1단계에서 경고 로그 후 통과, 2단계에서 거부. 동의 검사는 휴대폰 인증 검사보다 먼저 실행된다.
- 기록: 계정 생성 후 `AUTH_CONSENT`에 `PRIVACY / version / IS_REQUIRED=1 / IS_ACCEPTED=1 / ACCEPTED_AT` upsert(재시도 시 갱신). 소셜 연동은 신규 계정일 때만. 기록 실패는 로그만 남기고 가입은 유지. `AUTH_CONSENT`가 없는 DB(041 미적용)는 기동 시 경고 후 기록 생략.
- 파일: `internal/model/consent.go`, `internal/service/consent_service.go`, `internal/repository/consent_repo.go`, `internal/handler/auth_handler.go`, `auth_social_link_handler.go`, `cmd/server/wire.go`, `.env.example`. 테스트 3파일.
- 운영 반영 순서: 배포 → `PRIVACY_CONSENT_VERSION`을 T2 게시일로 설정 → 클라이언트 T4·T5 배포 → 강제 업데이트 이후 `PRIVACY_CONSENT_ENFORCE=true`.

## 5. T4. iOS

- `DSReverseStepForm`에 submit 버튼 위 `footer` 슬롯(또는 `consent: DSConsentRow?`)을 추가. 체크박스 + 라벨 + "내용 보기" 버튼. `canSubmit`은 체크 여부와 AND.
- "내용 보기"는 네이티브 시트로 동의 표(T1)를 보여주고 하단에 `/privacy` 링크(`AppConfig.foundationInfoURL(path: "privacy")`).
- 적용 화면: `NativeSignUpView`(일반), `LoginView` 소셜 가입 폼. 두 곳의 전화번호 필드 hint에 "인증번호가 문자로 발송됩니다." 추가(일반 가입은 중복 확인 메시지와 공존해야 하므로 hint 합성 규칙 필요).
- 요청 페이로드에 `privacyConsent` 추가(`AuthRepository`, `NativeSignUpRequest`, 소셜 연동 요청).
- 디자인: Remember 스타일 토큰 사용, 디자인 시스템 저장소에 `consentRow` 컴포넌트 계약 등록(기존 T14 등록 방식 참고).
- 검증: 체크 전 submit 비활성, 시트 열림, 페이로드 포함 여부 단위 테스트, 심사용 스크린샷(iPhone·iPad) 확보.

## 6. T5. 웹

- `RegisterForm.tsx`, `AccountLinkNewForm.tsx` submit 버튼 위에 동의 체크 + "내용 보기" 모달(동의 표 컴포넌트 공유, `consentContent.ts` 사용).
- `PhoneVerificationField.tsx` 83행 안내문을 "전화번호를 입력하면 인증번호가 문자로 발송됩니다."로 정리.
- `useRegisterFormValidation`에 동의 미체크 오류 추가. API 클라이언트(`api/auth.ts`, `useAccountLinkSubmit.ts`)에 `privacyConsent` 전달.
- 검증: 단위 테스트 갱신, Playwright 가입 폼 시나리오 1개 추가.

## 7. T6. Android (범위 경고)

- 현재 네이티브 가입은 인증 토큰을 보내지 않아 서버에서 거부된다. 이는 이번 리젹과 무관하게 **Android 가입이 깨져 있는 상태**다.
- 필요 작업: 인증번호 요청·확인 API 연동, `RegisterRequestDto`에 `phoneVerificationToken`·`privacyConsent` 추가, 화면에 인증 단계와 동의 체크 추가. iOS `PhoneVerificationController` 상태 기계를 참고해 이식.
- 규모는 iOS 인증 구현(`73063a3` + 후속 수정 2건)과 유사. 별도 계획 항목으로 산정하고 Android 출시 전 필수.

## 8. T7. 심사 재제출

- `APPLE_REPLY_EN.md`의 "registration screen states next to the phone field…" 문장을 "registration requires explicit consent to a data-collection notice that lists the phone number, its purposes (alumni roster matching, member identification, SMS ownership verification) and retention"으로 교체. Review Notes도 동일.
- 첨부: 동의 체크 화면, 동의 표 시트, 인증번호 단계 스크린샷.
- 운영 서버에 `SMS_REVIEW_TEST_PHONES`, `SMS_REVIEW_TEST_CODE` 설정 후 회신문 대괄호 기입. 심사 종료 후 제거.
- 표시 이름 반영 + 동의 UI 포함 빌드를 업로드하고 App Store Connect에서 명시적으로 선택.
- App Privacy: Phone Number = App Functionality, Linked, Not tracking.

## 9. 리스크

- 동의 UI 없이 회신하면 심사관은 "필수 항목이 늘었다"고 볼 수 있다. T4를 재제출 빌드에 반드시 포함한다.
- T3 2단계 강제 시점을 강제 업데이트와 맞추지 않으면 구버전 가입이 막힌다.
- 동의 문구를 세 플랫폼에 복제하므로 버전 값 불일치가 생기기 쉽다. 서버가 버전을 검증하고, 릴리스 체크리스트에 "동의 버전 = 처리방침 갱신일" 확인 항목을 넣는다.
- Android 가입이 현재 동작하지 않는 상태가 운영자에게 공유되어 있는지 확인 필요.
