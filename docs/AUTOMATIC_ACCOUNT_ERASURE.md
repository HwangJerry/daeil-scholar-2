# 자동 계정 삭제 구현 및 운영 연결

2026-09-07. 로컬 구현 문서이며 운영 배포·실제 회원 삭제 기록이 아니다.

## 처리 방식

- migration 057 이전의 수동 요청은 수동으로 유지한다. 이후 새 요청은 자동 처리 대상이다.
- root 관리자는 기존 계정 삭제 화면에서 수동 전환·자동 재개를 할 수 있다. 기존 수동 접수·검증·완료 기능을 유지한다. 현재 작업 단계가 끝난 뒤 전환이 반영될 수 있다.
- 분당 최대 10개 요청을 순차 처리하며 DB 연결에 연결된 named lock으로 서버 간 중복 실행을 막는다. 실패는 안전한 오류 코드만 남기고 5분 후 재시도한다.
- 기존 Apple/Kakao worker에 실제 철회를 요청한다. 이번 요청에 대한 성공 증거가 없으면 삭제를 완료하지 않는다.
- 기부 자료 분류·소셜 철회 확인 → 외부 작업용 식별정보 암호화 확보 → DB 삭제·암호화 보존·파일 작업 등록 → 실제 파일 삭제 → 외부 삭제 증거 → 재검증 → 비공개 접수증 완료 게시 순서다. 외부 처리 지연만으로 운영 DB 삭제를 막지 않는다. DB 단계는 같은 트랜잭션으로 묶이며, 파일 실패 후에는 DB 삭제를 반복하지 않고 남은 파일부터 재개한다.
- 자동 완료 확인은 로그인 화면에서 연결되는 비공개 접수증에 게시한다. 자동 이메일 발송은 추가하지 않았다. 수동 완료는 기존 개별 통지 증빙을 요구한다.

## 한국 기부 자료 보존 검토

| 자료와 적용 조건 | 기간·기산점 | 구현 |
| --- | --- | --- |
| 적용 대상 발급자의 기부자별 영수증 발급명세 | 발급일부터 5년 | `receipt_5y`. 실제 발급일을 입력해야 하며 탈퇴일로 다시 계산하지 않는다. |
| 해당 공익법인등의 장부·중요 증빙 | 해당 사업연도 종료일부터 10년 | `ledger_10y`. 실제 회계연도와 해당 의무 확인이 전제다. |
| 홈택스 전자기부금영수증 | 발급명세 작성·보관 의무에 법정 예외 | 해피나눔에서 파일을 받는다는 사실만으로 홈택스 전자 발급이라고 판단하지 않는다. |
| 법정 원본을 다른 보관소에서 유지하는 앱 사본 | 원본의 의무는 별도 이행 | `external_original`. 원본 보존 위치·증빙을 확인한 뒤 앱 사본을 삭제한다. |
| 별도 보관 의무 없는 자료 | 삭제 | `erase`. 검토 근거를 남긴다. |

근거: [법인세법 제112조의2](https://www.law.go.kr/LSW/lsSideInfoP.do?docCls=jo&joBrNo=02&joNo=0112&lsiSeq=280349&urlMode=lsScJoRltInfoR), [소득세법 제160조의3](https://law.go.kr/LSW/lsInfoP.do?lsId=001565), [법인세법 시행령 제155조의2](https://www.law.go.kr/LSW/lsSideInfoP.do?docCls=jo&joBrNo=02&joNo=0155&lsiSeq=283635&urlMode=lsScJoRltInfoR), [상속세 및 증여세법 제51조](https://www.law.go.kr/LSW/lsSideInfoP.do?docCls=jo&joBrNo=00&joNo=0051&lsiSeq=276123&urlMode=lsScJoRltInfoR).

영수증 법정 발급명세에는 앱이 수집하지 않는 주소·고유식별정보 등이 포함될 수 있다. 이번 앱 보관소는 수집 중인 최소 거래 증빙만 보관하며 **완전한 법정 영수증 원본을 대체하지 않는다**. 없는 주민등록번호 등을 새로 수집하지 않는다. 영수증 원본은 현재 발급·회계 시스템에서 필요한 범위를 유지해야 한다. 여러 의무가 겹치면 해당 자료에 실제로 적용되는 가장 늦은 종료일을 가진 근거를 선택하고 나머지 근거는 검토 증빙에 기록한다.

운영자의 단체명과 공개 공익단체 지정만으로 모든 세법상 지위를 확정하지 않았다. 2026-09-07 운영자가 회계연도 종료일 12월 31일, 해피나눔 디지털 영수증 발급·원본 시스템 보관을 확인했다. 홈택스 전자 발급이나 특정 장부 보존 의무의 적용을 이 답변만으로 확정하지 않는다. 법적 근거 확인 플래그는 임의 활성화하지 않는다.

## 최소 보관과 삭제

보관소에는 기부자명·기부일·기부/환불/순수령 금액·출처·거래번호·기간 근거와 검토 증빙만 저장한다. 회원 번호·전화·이메일·기수·학과·프로필은 복사하지 않는다. AES-256-GCM과 독립 키를 사용하고 일반 회원/관리자 조회 API를 제공하지 않는다. 보존 종료일의 한국 시간 자정 이후 삭제한다. 키는 DB와 분리해 관리하고 필요한 자료의 만료 전까지 복구 가능하게 보관한다. 암호화는 익명화를 의미하지 않는다.

운영 DB의 `WEO_ORDER`, 관련 PG 사본과 회원 연결은 삭제한다. 개인별 거래를 남기는 대신 모금 합계·인원 합계 차이를 식별자 없는 단일 집계에 이관한다. 다음 집계에서도 전체 모금액이 줄지 않게 반영한다. 해피나눔 원본의 보존/삭제 책임과 앱으로의 재수입 제외 절차는 실제 계약·운영에 맞춰 연결해야 한다. 삭제한 개인정보가 이후 엑셀 재수입으로 복원되지 않도록 운영 검증에 포함한다.

## 기부 분류 설정

이미 검토된 거래별 결정은 `backend/cmd/donation-retention`으로 입력한다. 기본은 파일 검증만 하고 `-apply`를 지정해야 DB에 기록한다. JSON에는 개인정보 원문을 넣지 않는다.

```json
[{"orderId":123,"basis":"ledger_10y","basisDate":"2025-12-31T00:00:00Z","retainUntil":"2035-12-31T00:00:00Z","evidenceReference":"회계담당자-검토문서-번호"}]
```

```sh
go run ./cmd/donation-retention -file /secure/reviewed-retention.json
go run ./cmd/donation-retention -file /secure/reviewed-retention.json -apply
```

장부 보존 의무와 회계연도, 완전한 영수증 원본 별도 보존이 확인되면 환경 설정으로 **향후 요청도 자동 분류**할 수 있다. `DONATION_LEDGER_RETENTION_CONFIRMED=true`, `DONATION_RECEIPT_ORIGINALS_SEPARATE=true`, `DONATION_LEDGER_YEAR_END_MONTH=확인한 월`, `DONATION_LEDGER_RETENTION_EVIDENCE=검토 증빙 번호`를 설정한다. 기존 거래별 결정을 우선한다. 미결제 등 분류가 불명확한 거래는 자동 추정하지 않는다. 윤년과 회계연도 경계를 달력 기준으로 계산한다.

`DONATION_ARCHIVE_KEY`는 전용 32바이트 키의 64자리 hex 값이다. 채팅·Git·명령 인수·운영 로그에 노출하지 않고 서버의 비밀 설정으로 주입한다.

## 외부 삭제 연결 — 아직 운영 연결되지 않음

`ACCOUNT_ERASURE_EXTERNAL_URL`은 신뢰하는 HTTPS 처리기, `ACCOUNT_ERASURE_EXTERNAL_TOKEN`은 그 처리기의 전용 인증이다. 이번 변경은 호출 계약과 실패 차단을 구현했다. **실제 Sentry API·백업 저장소·레거시 파일 정리 처리기는 아직 연결되지 않았으며 이 코드만 배포한다고 외부 자료가 자동 삭제되는 것은 아니다.**

처리기는 POST JSON으로 요청 번호·회원 번호·로그인 식별자·이메일·전화·소셜 subject·기부 보존 결정 목록을 받는다. 이 요청 번호로 재시도를 멱등 처리해야 한다. 전달된 법정 보존 결정을 존중하며 Sentry/로그/백업의 해당 개인정보, 간접 식별자, 과거 업로드 사본을 실제로 정리해야 한다. 일반 앱 DB를 임의 삭제하거나 보존 원본·보관소를 함께 지우면 안 된다. 개인정보를 응답·일반 로그에 넣지 않는다.

작업이 완료된 경우에만 아래 형태의 200 응답을 보낸다. 접수만 했다면 202 등으로 응답하고 완료를 가장하지 않는다. 시간 초과·인증 실패·부분 완료·잘못된 요청 번호·누락 증거는 모두 대기로 남는다. 리디렉션을 따라 인증정보를 다른 호스트로 보내지 않는다.

```json
{"requestId":17,"complete":true,"retentionRespected":true,"backupsErased":true,"externalDataErased":true,"historicalFilesErased":true,"otherIdentifiersChecked":true,"evidenceReference":"비공개-작업증빙-번호"}
```

Sentry 데이터의 실제 식별 가능성, 백업 선택 삭제/만료·복원 시 삭제 재적용, 예전 프로필 교체로 소유 이력이 사라진 파일은 설정과 실사 없이는 확인할 수 없다. 근거 없이 이 응답을 반환하는 더미 처리기를 운영에 두지 않는다. 실제 연동이 없으면 관리자가 기존 수동 절차로 처리할 수 있다.

## 배포와 검증

- migration 057·058·059를 056 다음에 적용하고 backend·frontend·admin을 함께 배포한다. 이번 작업에서는 운영 배포와 데이터 변경을 하지 않는다.
- migration 058은 존재하는 알려진 삭제 대상 MyISAM 테이블만 InnoDB로 전환한다. ALTER TABLE은 암묵적 커밋·테이블 재구축을 수반하므로 운영 적용 전 저장 공간과 작업 시간을 확인한다. 개별 전환은 재실행 가능하다. 삭제 대상 테이블과 관련 트리거가 InnoDB인지 확인한다. 알려지지 않은 참조나 비트랜잭션 저장소는 자동 처리를 차단한다. 운영 스키마에는 과거 레거시 테이블이 있을 수 있으므로 가상 계정으로 실제 배포 스키마를 검증해야 한다.
- 업로드 소유 기록은 새 프로필·명함 업로드부터 저장한다. 예전 파일 전체의 소유 관계를 소급해 알아냈다고 주장하지 않는다. 관리 경로 밖 파일·symlink·경로 이동은 자동 삭제하지 않고 확인 대상으로 남긴다.
- 로컬 자동 삭제·수동 유지·트랜잭션 롤백·간접 참조·기부 최소 보존·합계 유지·보존 만료·파일 실패 재개·암호화 무결성·안전한 파일 경로를 테스트한다. 공급자 실제 철회와 외부 저장소는 연결 뒤 별도 릴리스 후보 검증 대상이다.
- 계정 삭제의 실제 범위·기간·예외를 방침 및 앱에 일치시킨다. 정책 작성만으로 삭제 동작을 대신하지 않는다. [Apple 계정 삭제 안내](https://developer.apple.com/support/offering-account-deletion-in-your-app/).

## 로컬 검증 결과

Go 전체 테스트·vet, 실제 MariaDB 10.1.38 자동/수동 삭제 시나리오, frontend 139개 테스트, 두 SPA 빌드와 변경 파일 ESLint, 공개 화면 Playwright 5개, 관리자 모바일/데스크톱 수동 전환·자동 재개 테스트가 통과했다. iOS Debug 시뮬레이터 빌드가 통과했다. 이 결과는 운영 환경에서의 외부 자료 삭제나 App Review 승인을 의미하지 않는다.

## 2026-09-07 Sentry 실사 후속

- 조직 `metanoia-lab`, iOS 프로젝트 `4512014185725952`, Android `4512014191820805`. 로그인된 조직 설정에서 미국 저장과 신규 Business 체험판을 확인했다. 운영자는 체험 후 무료 요금제를 선택했다. 요금제 자체는 변경하지 않았다.
- 양쪽 프로젝트의 IP 저장 방지를 켜고 체크 상태를 확인했다. 기본 scrubber는 이미 켜져 있었다. 이 설정은 새 이벤트에 적용되며 과거 기록을 소급 삭제하지 않는다.
- 실제 iOS 네트워크 추적 표본에서 설치 식별자(`user.id`), 도시·국가, 요청 경로와 시각이 확인되었다. 원문 식별자는 이 문서에 복사하지 않았다. 전체 데이터에 개인정보가 없다는 증거로 사용할 수 없다.
- iOS 수집은 오류·충돌 진단의 허용 필드만 남기는 방식으로 보완했다. 성능/네트워크/세션/Replay/로그/지표 수집을 끄고 사용자·요청·임의 메시지·로컬 경로를 보내지 않는다. 앱 업데이트 전에는 적용되지 않는다. Android SDK 코드는 이번 iOS 작업에서 변경하지 않았다.
- Sentry에는 사용자별 DELETE API가 없다. 오류는 이슈 단위 삭제, 성능 기록은 개별 삭제 불가다. 기존 성능 데이터에는 설치 식별자가 있으므로 회원 ID 검색 결과가 없다는 이유만으로 삭제 완료를 증명할 수 없다. 프로젝트 전체 삭제는 다른 기록까지 제거하므로 자동 수행하지 않았다.
- 외부 처리기의 완료 증빙 검증은 유지한다. Sentry·백업 처리 완료를 반환하는 가짜 연동이나 404를 성공으로 처리하는 경로는 추가하지 않았다. 기존 데이터 삭제 또는 실제 만료 확인, 구버전 재수집 차단, Sentry 백업 만료 확인은 남아 있다.
- 신규 계정 체험은 기본 Team 보존: 오류 90일·성능 30일. 무료 전환 이후 새 데이터는 30일. 기존 데이터 보관 기간은 수집 당시 기준을 유지한다. Sentry 백업은 유형별 생성 후 30/90일이며 장학회 서버 백업과 별개다.

근거: [Sentry 보관 기준](https://docs.sentry.io/security-legal-pii/security/data-retention-periods/), [Sentry 삭제 기능과 백업](https://www.sentry.help/en/articles/16187006-how-do-i-complete-a-gdpr-erasure-request-for-a-user-in-sentry).

검증: iOS Debug 빌드와 개인정보 경계 테스트 4개 통과(직렬화 결과의 식별정보 제거, 미지원 이벤트 차단, 수집 경로 제한, 구버전 캐시 일회 삭제). 웹 production build·변경 파일 ESLint와 localhost 개인정보처리방침의 내용·가로 넘침·콘솔 오류 확인 완료. 새 서명 Release의 실제 전송 검증은 남아 있다.

설정 예제에는 확인된 `DONATION_RECEIPT_ORIGINALS_SEPARATE=true`, `DONATION_LEDGER_YEAR_END_MONTH=12`를 반영했다. 법적 적용 확인 플래그는 false를 유지하므로 이 두 운영 사실만으로 자동 10년 보존을 활성화하지 않는다. 운영 환경은 변경하지 않았다.

## Release project replacement — 2026-09-07

The operator confirmed both old projects contained pre-release test records and explicitly authorized deletion and replacement. The Sentry UI removed `daeil-ios` (4512014185725952) and `daeil-android` (4512014191820805); the project list now contains only the replacements below. This supersedes the earlier pending project-deletion decision. Sentry's backend/backup purge is not independently verified by this UI result.

| Platform | Project | ID |
| --- | --- | --- |
| iOS | daeil-ios-release | 4512042275897344 |
| Android | daeil-android-release | 4512042279632896 |

Both belong to `metanoia-lab` (US) and team `metanoia-lab`. Default scrubbers and IP-address storage prevention are enabled. No new email alert subscriptions or test events were created. The account remains on its existing trial; the operator intends to use the free plan afterward.

App DSNs are updated in iOS `Config/Info.plist` and Android `app/src/main/AndroidManifest.xml`. Existing installed builds still contain the retired project DSNs and must be replaced by newly built apps. No App Store/TestFlight/Google Play upload was performed.

Backend deployment configuration must use `SENTRY_ORG=metanoia-lab`, `SENTRY_IOS_PROJECT=daeil-ios-release`, `SENTRY_ANDROID_PROJECT=daeil-android-release` with a read token that can access both new projects. The committed backend env example is updated; the production service has not been restarted or deployed. Android mapping-upload jobs must use `SENTRY_PROJECT=daeil-android-release`; iOS symbol-upload jobs must use `SENTRY_PROJECT=daeil-ios-release`. No authentication token was generated or exposed.

Validation: iOS Debug simulator build and Android `:app:assembleDebug` succeeded. Actual release telemetry/symbolication and the backend monitoring proxy remain deployment-time checks. Android telemetry minimization is a separate follow-up; the DSN replacement does not implement the iOS crash-field allowlist on Android.

## 2026-09-07 삭제 단계 분리

migration 059는 외부 작업용 암호화 정보만 별도 테이블에 보관한다. `ACCOUNT_ERASURE_CONTEXT_KEY`에 JWT·기부 보관소와 다른 32바이트 hex 비밀키를 설정해야 한다. 키가 없거나 기존 정보의 인증 복호화가 실패하면 DB를 삭제하지 않는다. AEAD는 요청 번호에 결합되고 복호화 후 회원 번호도 대조한다. 일반 API에는 암호문·식별정보를 제공하지 않는다.

외부 증거를 받으면 작업 정보를 같은 트랜잭션에서 파기한다. 재시도로 생성 시점이나 만료를 연장하지 않는다. 생성 후 10일은 **작업 인계 한도**이며 법정 기간이나 서버 백업 만료일이 아니다. 만료 후 접근을 차단하고 기존 분당 정리 작업으로 파기한다. 만료된 미완료 요청은 `ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED`로 남으며 완료하지 않는다. 운영자는 만료 전 외부 작업을 처리·인계해야 한다. 실제 백업 운영에 맞는 별도 복원 방지 이력은 아직 구현되지 않았으므로 이 테이블을 복원 방지 대장으로 사용하지 않는다.

DB 삭제 여부는 비공개 접수증 및 관리자 목록에 표시한다. 최종 완료에는 파일 삭제와 외부 증거가 여전히 필요하다. 기존에 외부 증거를 확보한 요청은 새로운 식별정보를 만들지 않고 재개한다. 기존 수동 완료 시에도 임시 작업 정보를 정리한다.

이번 변경은 저장소별 외부 작업 처리기, 백업 복원 후 재삭제, 기부 재수입 방지까지 완성한 변경이 아니다. 각각의 구현과 운영 검증을 완료한 뒤 배포해야 한다.

## 저장소별 처리 계약과 운영 결정

migration 060을 059 다음에 적용한다. 기존 전체 범위의 검증 증거는 항목별 완료로 이전하며, 증거가 없는 기존 요청은 대기로 시작한다. 완료 후 30일에 접수증을 정리하면 항목별 근거도 FK cascade로 함께 파기한다.

HTTPS 처리기는 `requiredTargets`에 지정된 미완료 대상만 처리해야 한다. 새 응답 예시는 다음과 같다(식별정보·개인정보 원문을 근거에 넣지 않는다).

```json
{"requestId":17,"retentionRespected":true,"targets":[
 {"target":"backups","status":"pending","evidenceReference":""},
 {"target":"historical_files","status":"complete","evidenceReference":"file-audit-17"},
 {"target":"external_data","status":"not_applicable","evidenceReference":"verified-release-payload-audit-17"},
 {"target":"other_identifiers","status":"complete","evidenceReference":"reference-audit-17"}
]}
```

각 응답에는 요청한 대상이 정확히 한 번씩 있어야 한다. `not_applicable`은 실제 조사 근거가 있어야 하며, 단순 검색 결과 없음이나 설정 미확인은 근거가 아니다. 미완료 응답에는 오류 원문을 저장하지 않고 서버 소유 코드만 저장한다. 다음 시도는 미완료 대상만 요청한다. 구형 응답은 기존 전체 완료 조건을 만족할 때에만 전체 증거로 인정한다. 처리기 자체는 여전히 실제 저장소에 연결해야 한다.

가비아 백업·해피나눔 원본 보존 설정 확인 담당자는 황제철이다. 엑셀에는 이름·기수·과·연락처·금액만 있고 거래 고유번호는 없다. 일괄 날짜는 관리자의 등록일이다. 업로드 전 중복 거래·신규 기부·탈퇴자의 과거 자료 재반영 여부와 모금 합계 중복을 황제철이 판단한다. 추가 자동 중복 차단과 입금일 입력 강제는 운영자 결정으로 제외했다.

`happy_nanum` 엑셀 자료는 등록일로 자동 장부 보존 기산일을 산출하지 않는다. 기존 거래별 원본 검토에 근거한 보존 결정은 계속 사용한다. 보관소에도 해당 날짜가 등록일임을 표시한다. 실명은 검토된 최소 증빙에 포함하지만 연락처를 일괄 장기 보존하도록 변경하지 않았다. 운영자는 계좌이체 기부 원본을 별도 엑셀로 보관하고, 탈퇴 후 영수증 발급 등을 위해 연락처가 필요할 수 있음을 확인했다. 엑셀의 구체적인 저장 매체·접근 권한·보존 종료 기준은 아직 확인되지 않았다. 이 운영상 필요성을 연락처의 일괄 장기 보존 의무로 간주하지 않는다. 연락처 보존 범위와 종료 기준을 확정한 뒤 실제 삭제 동작과 방침에 반영한다.

## 서버 백업 실사와 연락처 기준안

가비아 자동 백업·스냅샷 및 서버 이미지 백업은 아직 사용하지 않는 것으로 운영자가 확인했다. 유지보수 디렉터리는 목록상 설정 파일 사본이며 회원 DB·업로드 백업은 발견되지 않았다. 조사 범위와 한계, 기부 연락처 보존 기준 제안은 [저장소 실사 기록](ACCOUNT_ERASURE_STORAGE_AUDIT.md)에 정리했다. 운영 완료 상태나 공개 개인정보처리방침은 이번 조사만으로 변경하지 않았다.

## 채택된 영수증 연락 업무 종료 기준 구현

운영자는 **진행 중인 영수증 업무와 결과 전달 완료 후 해당 업무용 연락처 삭제**를 채택했다. 이를 migration 061과 기존 root 관리자 삭제 화면에 연결했다. 새 별도 수동 삭제 시스템이나 기부자 연락처 조회 API를 추가하지 않았다.

- 업무 상태는 `unreviewed`, `not_required`, `active`, `completed`다. 회원·전화번호·이메일 원문을 업무 테이블에 복사하지 않는다. 원본 위치와 개인정보 없는 증빙 번호, 확인 담당자·시각만 기록한다.
- 기부 내역이 연결된 자동 요청은 진행 업무를 확인해야 한다. 업무가 있으면 해피나눔·별도 엑셀·양쪽 원본 중 어디에 필요한 연락처와 보존 근거가 확보됐는지 담당자가 확인한다. 확인 후 앱 DB·현재 파일 삭제는 진행할 수 있다.
- 진행 중인 업무가 있으면 외부 삭제 호출과 최종 완료를 보류한다. 접수증에서 영수증 업무가 진행 중임을 설명한다. 연락처를 법정 증빙 암호화 보관소에 일괄 복사하거나 5년·10년 보존으로 설정하지 않는다.
- 담당자가 실제 업무 완료·결과 전달 및 원본/사본의 불필요한 연락처 정리를 각각 확인하고 증빙을 기록하면 자동 작업을 다시 예약한다. `active`를 단순히 '업무 없음'으로 바꿔 완료 확인을 우회할 수 없다.
- 기존 수동 완료도 `active` 업무가 남으면 차단한다. 미확인 상태의 기존 수동 요청은 기존 전체 삭제 검증 증빙으로 '업무 없음'을 확인할 수 있다. 이미 종료한 업무를 재개하여 보관 기간을 늘릴 수 없다.
- 이 기록은 외부 원본 삭제를 실행하는 API가 아니다. 황제철 담당자가 해피나눔·별도 회계 엑셀의 실제 정리를 수행해야 한다. 연락처 보존이 필요하다는 운영 사실만으로 법정 보존 예외라고 자동 판정하지 않는다.
- 업무가 장기 지연되어 외부 작업용 암호화 정보의 기존 10일 인계 한도가 지나면 기존 수동 확인 경로로 처리한다. 일괄 무기한 연장하지 않는다. 운영 지연 사유·예정일 안내 의무는 유지한다.

적용 순서: migration 061 → backend·admin·frontend 배포. 현재는 로컬 구현이며 운영 마이그레이션이나 데이터 삭제를 수행하지 않았다. 서버 백업 미사용 확인과 계좌이체 원본의 별도 엑셀 보관을 방침 문구에 반영했다.

## 확인된 과거 파일을 자동 삭제에 연결 — 2026-09-07

`cmd/erasure-files`로 검토한 과거 파일을 기존 큐에 넣을 수 있다. 기본은 dry-run이며, DB 삭제 후의 자동 처리 요청에만 적용한다. 다른 회원/게시글의 현재 참조, 잘못된 요청 대상, 경로 우회를 차단하고 기존 수동 전환을 존중한다. 전체 과거 파일 범위나 외부 저장소의 삭제 완료를 대신 증명하지 않는다. 상세 입력·권한·검증 방법은 [과거 파일 운영 절차](HISTORICAL_FILE_ERASURE_RUNBOOK.md)를 참조한다.

`ACCOUNT_ERASURE_LEGACY_ROOT`는 공개 파일 제공 경로와 별도로 검증한 실제 디렉터리를 지정한다. 루트 부재는 삭제 성공이 아니며, 심볼릭 링크 검사를 생략하지 않는다. 새 요청/panic 로그는 라우트 템플릿만 기록하고 panic 값 원문 전송을 제거했다. 과거 로그 및 PG·Debug Agent·Sentry의 실제 데이터 정리는 별도 확인 대상으로 남아 있다.

## 현재 참조 보존과 출시 검증 — 2026-09-07

Migration 062는 현재 관리 사진/명함/썸네일 경로를 확보하고, 사진/명함 교체 시 이전 경로를 같은 트랜잭션에 기록한다. 자동 탈퇴는 그 이력을 파일 큐로 옮긴 후 제거한다. `/upload/`, `/old/upload/`도 실제 레거시 루트로 연결한다. 이미 사라진 이전 연결을 추측해서 복원하지 않는다.

운영 키 준비·PG 정리본·오류 로그 최소화·실제 App Store IPA와 Sentry 수신 검증 및 남은 외부 연동은 [최신 인계 문서](ACCOUNT_ERASURE_RELEASE_HANDOFF.md)를 참조한다.
