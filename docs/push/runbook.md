# Push Runbook

Version: 1.1
Last Updated: 2026-07-17
Owner: SRE / On-call

본 Runbook은 push 이슈 발생 시 초동 대응을 통일한다.

## 공통 응답 체계

- 심각도 기준
  - **P1**: 전체 알림 실패율 급증, 서비스 영향 크고 복구 지연 가능
  - **P2**: 특정 유형/특정 단말군에 국한된 이상
  - **P3**: 경미한 정합성/문구 이상
- 목표: P1 15분, P2 60분, P3 24시간 내 1차 조치
- 기록 형식: `incident` 티켓 + 채널 링크 + 증상 로그 링크 + 조치 결과

## 시나리오 1: invalid token 급증

### 증상
- `invalid_token` 급증, `delivery_success` 하락, 사용자 불만 증가

### 즉시 조치(1단계)
1. 토픽/이벤트별 invalid율 급증 상위 10개 추출
2. APNs/FCM 장애 여부 확인(공급사 상태)
3. 최근 배포 변경사항(푸시 provider credential, provider key, 토큰 스키마) 확인

### 1차 대응
1. 무효 토큰 자동전환 실패 건수 확인
2. purge job 강제 실행(검토 후)
3. 필요 시 임시적으로 특정 이벤트 타입 발송 중단

### 종료 조건
- invalid율이 임계치(예: 1% p95) 이내로 회복

### 증빙
- 지표 캡처, 이벤트별 임계치 로그, 조치 커맨드/실행 시간

## 시나리오 2: 중복 알림 폭증

### 증상
- 동일 이벤트 반복 알림, 사용자 혼란, 고객지원 티켓 증가

### 즉시 조치
1. event_id 중복률 및 dedupe hit율 확인
2. `event_id` 생성 흐름(메시지 큐/배치) 점검
3. `collapse_key` 또는 outbox 유니크 제약 상태 확인

### 1차 대응
1. 중복 발생 이벤트 타입 즉시 라우팅 제한
2. 클라 캐시 TTL 임시 상향 조정 검토
3. 서버 고급 로그 기준으로 중복 원인 분기

### 종료 조건
- 사용자 노출 중복률 0.1% 미만 재확인

### 증빙
- 중복률 추이, 조치 이전/이후 비교 스크린샷

## 시나리오 3: 오프라인 정합성 이탈

### 증상
- 푸시 알림은 받았는데 목록/상태가 다름

### 즉시 조치
1. pull sync API `cursor` 동작 확인
2. 앱 진입/복귀 경로에서 sync 강제 여부 점검
3. SSE/WebSocket fallback 동작 상태 확인

### 1차 대응
1. 강제 sync 재실행 기능 점검
2. 동기화 장애 구간의 이벤트 백로그 수동 재처리

### 종료 조건
- 5분 내 정합도 지표가 99% 이상 회복

### 증빙
- sync_lag_ms 그래프 + 이벤트 재처리 로그

## 시나리오 4: 보안 이슈(위조/필터 우회)

### 증상
- 위조 action_token 탐지, allowlist 위반 deep link 트래픽 증가

### 즉시 조치
1. 보안 로그에서 공격 source, IP, 토큰 타입 추적
2. 유효하지 않은 토큰/서명 실패 수치 확인
3. API/딥링크 허용 목록 점검

### 1차 대응
1. `action_token` 키 회전 고려(필요 시)
2. 관련 템플릿 발송 중단(심각도에 따라)
3. SRE/보안팀 동시 대응

### 종료 조건
- 위조/오류율이 기준치 이하

### 증빙
- 보안 로그, 조치 내역

## 시나리오 5: iOS 백그라운드 정책 위반

### 증상
- BGTask 실패율 급증/앱 푸시 반응 지연

### 즉시 조치
1. 백그라운드 실행시간 top N 추출
2. silent push에서 실제 수행한 작업 유형 검사

### 1차 대응
1. 앱 업데이트가 필요한 경우 hotfix 준비
2. Background 처리를 foreground로 이전할 임시 정책 결정

### 종료 조건
- background fail율이 임계치 이하로 회귀

### 증빙
- 기기군별 BGTask 실패 로그

## 시나리오 6: APNs/FCM credential 또는 systemd 배포 실패

### 증상

- `APNs provider configured` startup 로그가 없음
- `.p8` read/parse 오류로 backend가 시작하지 못함
- FCM service account JSON `permission denied`로 backend가 재시작을 반복함
- `InvalidProviderToken`, `BadDeviceToken` 급증
- `deploy.sh`가 필수 환경변수 또는 APNs key 권한 검사에서 중단

### 즉시 조치

1. `SKIP_ENV_CHECK=1`로 우회하지 않는다.
2. 새 binary 업로드 또는 daemon 재시작이 실행됐는지 확인한다.
3. `systemctl status`와 `journalctl -u alumni-backend`의 첫 startup error를 확인한다.
4. `/etc/sysconfig/alumni-backend`는 값이 아니라 key name만 출력해 누락 여부를 확인한다.
5. daemon 사용자로 해당 환경 `.p8`과 FCM JSON 읽기 가능 여부를 검사한다.
6. FCM credential, 환경파일, Android 앱의 Firebase project id가 일치하는지 확인한다.
7. token의 `APNS_ENVIRONMENT`와 configured credential environment가 일치하는지 확인한다.

### 복구

- EnvironmentFile 전환 중 운영값이 누락됐다면 기존 unit/env 백업에서 복구한다.
- FCM JSON은 `root:alumni-backend`, `0640`으로 수정하고 daemon 사용자 기준 `test -r`을 통과시킨다.
- 새 binary가 시작하지 못하면 binary, unit, EnvironmentFile 세 파일을 함께 이전 백업으로 되돌린다.
- Production credential이 없는 동안 Production job을 발송하거나 replay하지 않는다.
- 상세 명령과 장애 이력은 [centos7-apns-operations.md](centos7-apns-operations.md)를 따른다.

### 종료 조건

- backend가 `alumni-backend` 사용자로 active
- APNs provider configured environment가 예상값과 일치
- FCM credential이 설정된 경우 backend dependency wiring이 성공
- `/api/health`가 HTTP 200
- outbox smoke job이 `SENT`
- journal에 private key, raw device token, 환경파일 내용이 노출되지 않음

### 증빙

- 배포 preflight 로그
- `systemctl status` 및 startup log
- outbox status query 결과
- 적용한 unit/env/key 파일의 owner/group/mode, 값 제외

## 사전 체크리스트(재발방지)

- contract/policy/qa 갱신 반영
- 토큰 상태 FSM과 purge 정책 확인
- 배포 노트에 iOS 동작 변경 사항 명시
- 모니터링 패널 임계치 재조정
