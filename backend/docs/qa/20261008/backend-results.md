# 인증 QA — backend/API, 2026-10-08

모바일 인증의 서버 경계를 실제 HTTP 라우터와 MariaDB 10.1.38에서 검수했다. 인증 API Golden 23개와 추가 경계 테스트 9개가 모두 통과했다. 소셜 제공자의 실제 화면 및 사용자 인증은 각 기기의 QA 결과에서 별도로 확인해야 한다.

## 검증 결과

| 실행 | 결과 | 범위와 한계 |
| --- | --- | --- |
| 기존 `go test -json ./...` | run 1,578 / pass 1,509 / skip 66 / fail 3 | 하위 테스트 포함 이벤트 수. Docker 비활성화로 생략된 DB 검증은 통과로 계산하지 않았다. |
| `DFLH_DOCKER_TESTS=1 go test -json ./cmd/server -run '^TestGolden(Signup\|MobileSession\|AccountDeletion)$' -count=1` | run 23 / pass 23 / fail 0 / skip 0 | 회원가입, iOS·Android 헤더로 ID/PW 로그인 및 refresh, /me, 강제 업데이트, 회원 탈퇴 접수 및 영수증 조회. |
| `DFLH_DOCKER_TESTS=1 go test -json ./cmd/server -run '^TestAuthQAAPIRecoveryBoundaries$' -count=1` | run 9 / pass 9 / fail 0 / skip 0 | 아래 8개 시나리오와 상위 테스트. 최종 실행 78.381초. |

Apple Silicon에서 amd64 MariaDB를 실행하는 첫 Golden 시도는 schema import의 기본 15초 제한에 걸렸다. 위 실제 DB 실행에서는 테스트 하네스의 schema/query timeout을 일시적으로 120초로 늘렸고, 종료 후 원래 파일을 그대로 복구했다. 애플리케이션의 production timeout, DB, 서버 설정은 이 검증 때문에 변경하지 않았다.

전체 baseline의 세 실패는 `internal/contract`의 후보 migration 승인 목록과 현재 소스의 범위·지문 불일치다. 승인 manifest를 자동으로 고쳐 통과시키지 않았다.

- `TestCanonicalIdentityCandidateRangeCheck`
- `TestCanonicalIdentityCandidatePacketMatchesCurrentSource`
- `TestMigrationRunnerApprovesEveryFutureMigrationIncluding063`

## 추가 경계 시나리오

검증 코드: `cmd/server/auth_qa_edge_cases_test.go`. 전용 DB와 합성 사용자·전화·제공자 subject만 사용한다. DB 장애는 해당 테스트 DB에 임시 trigger를 넣어 재현한다.

| 시나리오 | 관찰한 결과 |
| --- | --- |
| 소비된 refresh 재사용 | `401 REFRESH_REPLAY_DETECTED`. 회전된 새 refresh와 access도 같은 SID 전체가 거부됨. 다른 디바이스의 별도 SID는 유지됨. |
| 현재 기기 로그아웃 / 전체 로그아웃 | 현재 SID 로그아웃은 다른 SID를 유지함. 전체 로그아웃은 다른 SID의 access와 refresh까지 거부함. 모두 `204`, 응답 본문 없음. |
| 인증 미제출 회원의 로그인 / refresh | 미제출 principal의 정상 서비스 세션 발급 가능. `/me`는 접근 가능하고 승인된 동문 목록은 `403 ALUMNI_APPROVAL_REQUIRED`. refresh로 승인 권한이 생기지 않음. |
| 로그인 후 정지 처리 | 기존 access로 `/me` 접근 불가. refresh는 `403 ACCOUNT_SUSPENDED`, 해당 세션 family가 폐기됨. |
| ID 중복으로 회원가입 실패 | `409 ID_TAKEN`. 신규 회원 및 로그인 세션 생성 없음. 인증된 전화 grant는 소비되지 않아 재시도 가능. |
| 소셜 회원 연결 transaction 실패 | `500 LINK_FAILED`. 회원 insert 및 관련 연결을 rollback. 전화 grant와 소셜 continuation을 재사용 가능. |
| 회원 생성 commit 후 세션 저장 실패 | `500 LOGIN_FAILED`. 회원은 존재하지만 서비스 refresh row는 0개. 전화 grant·소셜 continuation은 이미 소비됨. 같은 제출 재시도는 `400 PHONE_NOT_VERIFIED`. 새 제공자 로그인은 기존 회원을 찾아 정상 세션 발급 가능. |
| 회원 탈퇴 취소 | 요청 즉시 양 디바이스 세션 폐기 및 신규 로그인 거절. 잘못된 취소 비밀은 `404`. 정상 취소 후에도 이전 토큰은 거절하며 새 로그인만 허용. |

위 소셜 경계 테스트의 제공자 검증 결과는 fixture로 주입한다. 실제 Kakao·Apple 토큰 교환/동의 UI, biometric, 제공자 세션의 유효성까지 검증한 결과로 해석하면 안 된다.

## 인증 구조와 실패 시 세션

모바일 access는 15분, refresh는 30일이다. refresh는 SID를 유지하며 JTI를 회전시키고, 이전 JTI가 소비되었는지 transaction으로 확인한다. 소비된 토큰의 재사용은 해당 SID의 family를 폐기한다. Bearer middleware는 JWT 서명·유효기간에 더해 현재 회원의 로그인 허용 상태와 SID의 유효 refresh row를 매 요청 확인한다. 회원의 인증 상태와 관리자 역할도 DB principal에서 다시 읽는다.

ID/PW는 canonical `AUTH_IDENTITY` 및 `AUTH_PASSWORD_CREDENTIAL`을 우선 사용한다. legacy 비밀번호를 지원하는 경로가 별도로 있다. 소셜 계정 식별은 제공자가 검증한 provider subject 기준이며 이메일을 같다는 이유로 자동 연결하지 않는다. 소셜 회원가입 continuation은 인증 완료 전의 자료를 5분 동안 보관하며 ready/processing/consumed lease로 중복 처리를 방지한다. 실패 전 account transaction이 rollback되면 lease를 반환하고, account가 생성되면 continuation을 소비한 뒤 모바일 세션을 발급한다.

확인된 서버 쪽 세션 잔류 문제는 없다. 다만 account commit 이후 세션 발급 실패는 회원 생성과 사용자에게 보이는 실패가 함께 발생하는 경계다. Android `signUpAsKakaoMember`와 iOS `completeSocialSignUp`의 일반 오류 처리에서는 가입 context를 그대로 남기므로, 이용자가 같은 폼에서 재시도하면 소비된 grant에 막힐 수 있다. 이 경계의 서버 응답·복구는 위 DB 장애로 재현했고, 실제 모바일 화면에서의 복구 UX는 별도 확인 대상으로 남긴다.

## 운영 계정 분리 확인

운영 DB에서 이메일·전화로 프로필을 대조하되 연락처, 비밀번호, 제공자 subject/token을 보고서에 남기지 않았다. Android 계정의 mapping은 실제 Kakao 인증 후 새로 캡처한 프로필에 기반한다.

| 용도 | usrSeq | usrId | 확인 당시 상태·권한 | 로그인 수단 |
| --- | --- | --- | --- | --- |
| 새 ID/PW root | 12813 | jerryhwang018 | ZZZ / root | LOCAL_USERNAME만 존재. Kakao 연결 없음. |
| iPhone에서 사용 중인 기존 root | 12787 | ghkdwp018 | ZZZ / root | ID/PW + Kakao + Apple |
| Android의 실제 Kakao 계정 | 12811 | K4895702339 | BBB / 관리자 권한 없음 | Kakao만 존재, password credential 없음. |

새 root의 legacy 및 canonical 비밀번호 credential은 사용자의 지시에 따라 기존 root의 저장된 값을 DB transaction 안에서 복사했다. 원래 계정은 변경하지 않았다. 새 root의 세션 refresh `200` 및 root 전용 관리자 `/api/admin/operators` 접근 `200`을 복사 이후 재확인했다. 새 root에는 제거할 Kakao 연결이 없다는 점도 legacy social 및 canonical identity 양쪽에서 확인했다.

관리자 API는 `adms.daeilfoundation.or.kr` host를 사용해야 한다. 회원 사이트 host의 `/api/admin/*` 거절을 인증 오류로 오판하지 않았다.

## 운영 탈퇴 QA의 처리 조건

확인한 활성 설정은 request/worker enabled, 대기 168시간, 전용 test-user 제한 없음, external mode `internal`이다. Kakao admin key, Apple private-key 경로, 소셜 credential encryption key가 설정되어 있으며 두 디바이스 계정의 legacy 암호화 credential도 존재한다.

반복 탈퇴·재가입은 운영의 일반 탈퇴 절차로 접수한 후 계정별 요청만 검토한다. 전역 7일 설정을 바꾸거나 SQL로 회원 테이블을 직접 지우지 않는다. root가 `/api/admin/account-deletions/{id}/preview`의 실제 대상·blocker를 검토하고 현재 `planDigest`로 `action=expedite`를 승인하면 그 요청에만 `EXPEDITED_AT`이 기록되어 대기 날짜 이전에도 worker 대상이 된다. 기존 scheduled/target 날짜는 그대로 남는다. worker는 결제/보존 결정, 제공자 철회, 파일, 외부 식별값 및 백업 증거를 계속 검사하며 증거를 임의로 성공 처리하지 않는다.

실기기의 운영 탈퇴 요청별 preview, 최종 처리 및 재가입 결과는 통합 QA 보고서에 후속 기록한다.

## 재현 로그

비공개 workspace의 `build/auth-qa-20261008/`에 baseline JSONL, `backend-auth-golden.jsonl`, 최종 `backend-auth-edge-api-final.jsonl`, 계정 mapping 및 root 권한 검증의 비밀값 없는 증거를 저장했다. production 세션 및 preview 원본은 별도 0600 파일로만 보관하며 커밋 대상이 아니다.
