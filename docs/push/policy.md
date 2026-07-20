# Push Policy (dflh-saf-v2)

Version: 1.1  
Last Updated: 2026-07-02  
Owner: Engineering Team (Backend + Mobile + QA + SRE)

본 문서는 푸시 기능의 운영 정책을 정의한다.  
모든 항목은 `do-or-die` 성격으로 정책 위반 시 PR를 차단한다.

## A. 토큰 정책

- 토큰 상태: `ACTIVE`, `STALE`, `UNVERIFIED`, `REVOKED` (필수)
- 필수 메타: `platform`, `app_version`, `last_seen_at`, `invalid_count`, `updated_at`, `locale`
- 상태 전이:
  - `ACTIVE` → `STALE` : `last_seen_at`이 7일 이상 경과
  - `STALE` → `REVOKED` : invalid_count 증가 임계치 초과 또는 APNs/FCM invalid 응답
  - `UNVERIFIED` → `ACTIVE` : 정상 토큰 갱신 이벤트 수신
- 삭제:
  - `REVOKED` 토큰은 즉시 발송 제외
  - 30일 이상 `REVOKED`/`STALE` 미사용 시 purge

KPI:
- invalid 반영 지연 60초 이내
- purge 적용률 95% 이상

## B. 중복 제어 정책

- 각 푸시 이벤트는 `event_id`를 필수로 사용한다.
- `dedupe_ttl`(기본 24h) 내 동일 사용자/동일 `event_id`는 1회만 표시한다.
- 클라 표시 정책: 동일한 `event_id`가 이미 표시된 경우 추가 알림은 user-facing suppression.

KPI:
- 동일 이벤트 사용자당 노출 횟수 1회
- 중복 drop률은 과도(>99%)가 아닌 범위로 추적 후 이상치 경보

## C. 보안 정책

- 금지: custom data/`args`에 `이름/DM 본문/전화번호/주소/학번/개인식별 토큰` 저장
- 허용: `message.new`의 사용자 표시용 alert에 발신자 이름과 최대 80 Unicode 글자의 쪽지 미리보기 표시
- 허용: `admin.notice`의 사용자 표시용 alert에 공지 제목 표시
- iOS 재시도를 위한 outbox의 alert 제목/본문 저장은 허용하되 애플리케이션 로그에는 출력하지 않는다.
- 그 외 이벤트 데이터는 template key 기반 렌더링 + 인앱 fetch를 기본으로 한다.
- `action_token`:
  - 서명 필요(HMAC/JWT)
  - 만료(예: 10분) 및 1회성 우선 적용
  - 서명 실패 시 처리: 이벤트 무시 + 보안 경보
- 딥링크:
  - allowlist에 등록된 경로만 허용
  - 미허용 경로는 무시 및 경고 로그

KPI:
- payload PII 스캔 0건
- invalid signature 비율은 0.2% 미만이면 경보 없음

## D. iOS 정책

- silent push는 “동기화 트리거”로만 사용한다.
- 실제 데이터 동기화는:
  - 앱 실행 전환/포그라운드 전환 시
  - BackgroundTasks 사용 시점에만 수행
- 백그라운드에서 대량 네트워크/저장소 변경 금지

KPI:
- background overrun(임계치 초과) 0건
- background fail/retry 5분 구간 모니터링

## E. 동의/철회 정책

- 동의 상태는 서버 단일 진실원으로 관리.
- 앱은 동의 상태를 캐시할 수 있으나 송신 판단은 서버 조회 기준.
- 동의 철회 즉시: 송신 제외 + 관련 토큰 revoke 큐 적재 + 감사 로그 저장

KPI:
- 철회 요청 후 1분 이내 발송 금지
- 동의/철회 감사 로그 보존(내부 정책 기준)

## F. 리텐션 및 감사

- 푸시 이벤트 90일 보존(운영 정책에 따라 조정 가능)
- dead-letter 및 invalid 토큰 로그 최소 180일 보존
- 감사 로그 변경 불가 형태로 저장

## G. 운영 정책

- APNs/FCM provider credentials are runtime configuration only.
  - Firebase service-account JSON must never be committed.
  - Backend FCM requires `FCM_CREDENTIALS_JSON` or `FCM_CREDENTIALS_FILE` and, when needed, `FCM_PROJECT_ID`.
  - Missing local credentials may use backend no-op push behavior; configured but invalid production credentials must fail clearly before smoke/launch.
- Android registration/provider support unblocks backend smoke, but physical-device validation still requires Firebase project config and a real device/emulator FCM registration token.

- PR 승인 이전에 다음이 충족되어야 함:
  1. `docs/push/contract.md`와 일치
  2. 보안/테스트 실패 0
  3. 모니터링 룰 존재
  4. 런북 대응 시나리오 존재
- 위반 시 PR 차단
