# 회원 탈퇴 검증 결과 — 2026-09-07

## 범위와 판정

사용자 요청에 따라 과거 운영 데이터 정리를 보류하고 탈퇴 검증만 수행했다.
격리된 로컬 MariaDB 10.1.38의 합성 계정과 임시 파일만 생성·삭제했다.
운영 DB·파일·기부 기록·결제 로그는 변경하지 않았고, 삭제 큐 등록·운영 배포도 하지 않았다.
기존 데이터 정리 결재안은 계속 미승인이다.

**로컬 구성요소 검증은 통과, 운영 출시 후보의 종단 검증은 미완료다.**

## 실행 결과

| 검증 | 결과 | 실제 확인한 범위 |
|---|---|---|
| MariaDB 자동/수동 탈퇴 | 2개 통과 | 합성 회원의 실제 DB 삭제, 상태만 바꾼 요청의 완료 차단, 다른 회원 유지, 실패 시 롤백, 기부 합계 유지 |
| 백엔드 서비스·HTTP handler | 28개 통과 | 인증 필수, 접수증 처리, 안전한 파일 경로, 재시도, 미완료 외부 작업 차단, 영수증 업무, 기부 보존 기산일·암호화 |
| 실제 임시 파일 삭제 | 위 28개 중 포함 | 자동 서비스가 테스트 파일을 삭제하고 외부 확인 없이 완료하지 않는지 확인, 레거시 URL 별칭 검사 |
| iOS Release 설정 | 8개 통과 | 취소/확인 상태, 접수 성공 후 세션 정리, 실패 시 세션 유지, 접수증 선저장, 통신 끊김 후 접수 확인, 구형 204 응답 거부 |

상위 테스트 함수 기준 총 38개이며, 하위 사례를 중복 집계하지 않았다.
테스트용 MariaDB 컨테이너는 실행 후 제거되었다.

재현 명령은 backend에서 다음과 같다.

```sh
AUTOMATIC_ERASURE_DOCKER_INTEGRATION=1 MANUAL_DELETION_DOCKER_INTEGRATION=1 \
go test ./internal/repository -run 'Test(AutomaticErasureOnMariaDB101|ManualAccountDeletionLifecycleOnMariaDB101)$' -count=1 -v

go test ./internal/service ./internal/handler ./internal/middleware \
-run '(Erasure|Deletion|Receipt|LoginEligibility)' -count=1 -v

go test ./internal/service \
-run 'Test(DonationRetentionUsesRecordedClockNotWithdrawalDate|DonationArchiveAuthenticatedEncryption|LedgerRetentionTemplateRequiresConfirmedPolicy|ManualTakeoverAndUnverifiedRetentionPreventAutomaticMutation|ExternalFailureResumesWithoutMemberRows|MissingContextKeyPreventsLosingExternalIdentifiers|Historical|Legacy)' -count=1 -v

go test ./internal/service -run '^TestWorkerDeletesRealHistoricalFileWithoutFalseExternalCompletion$' -count=1 -v
```

iOS는 `xcodebuild test`, `Release`, `ENABLE_TESTABILITY=YES`로
`AccountDeletionUIStateTests` 2개, `AuthSessionTerminationCoordinatorTests`의 탈퇴 2개,
`AuthNoContentResponseTests`의 탈퇴 4개를 지정했다.
시뮬레이터는 `92D331D5-D3EE-410D-B7FC-CB99D29ADEA4`다.

실행 로그는 로컬 `/tmp/dflh-withdrawal-verification-{db,backend,retention,files,ios}.log`에 있다.

## 운영 준비 상태 — 읽기 전용 확인

운영 프로세스 설정은 값 원문 없이 존재 여부만 검사했다.
운영 DB는 읽기 전용 세션에서 스키마만 조회했다.

- 다음 테이블 6개는 아직 없음:
  `ALUMNI_ACCOUNT_DELETION_REQUEST`, `ALUMNI_ACCOUNT_ERASURE`,
  `ALUMNI_ERASURE_CONTEXT`, `ALUMNI_ERASURE_TARGET`,
  `ALUMNI_ERASURE_RECEIPT_WORK`, `ALUMNI_PROFILE_FILE_HISTORY`.
- `ACCOUNT_ERASURE_CONTEXT_KEY`, `DONATION_ARCHIVE_KEY` 미설정.
- `ACCOUNT_ERASURE_EXTERNAL_URL`, `ACCOUNT_ERASURE_EXTERNAL_TOKEN` 미설정.

따라서 현재 운영 상태에서 새 탈퇴 접수부터 최종 완료까지 동작한다고 판정할 수 없다.
최신 백엔드·마이그레이션·필요 설정 반영과 실제 외부 처리 범위 연결이 선행되어야 한다.
설정 미확인 항목을 임의 완료시키거나 더미 외부 완료 응답을 운영에 넣지 않았다.

## 검증의 한계와 다음 검증

- MariaDB 테스트는 운영과 같은 DB 버전의 합성 스키마이며 운영 전체 데이터 사본이 아니다.
- DB 보관소 삽입과 암호화는 각각 검사했다. DB 통합 테스트의 암호화 콜백은 합성 응답을 쓰고,
  실제 AES-GCM 무결성은 별도 서비스 테스트에서 확인했다.
- Apple/Kakao 철회와 외부 삭제 증거는 테스트 대역을 사용했다. 실제 공급자의 토큰 철회나
  해피나눔·별도 엑셀 삭제를 수행·검증한 결과가 아니다.
- iOS는 네트워크 대역을 사용한 테스트이며, 실제 기기 화면 조작부터 운영 DB 삭제까지 이어지는 검증이 아니다.
- 운영 준비 후 검증 전용 계정으로 탈퇴 접수, 재로그인 차단, 관련 DB/파일 삭제,
  해당되는 소셜 토큰 철회, 접수증 완료를 확인해야 한다.
  운영 삭제가 필요한 경우 대상 테스트 계정을 특정하고 사용자의 사전 결재를 받은 범위만 처리한다.
