# Push QA Matrix

Version: 1.0  
Last Updated: 2026-06-11  
Owner: QA Team

10가지 고민을 마감하기 위한 테스트 매트릭스다.  
각 항목은 CI에서 재현 가능한 형태로 관리한다.

## 공통 테스트 전제

- 테스트 계정 3종: iOS/Android, iOS silent push 지원 환경, 오프라인 시나리오 장치
- 고정 시간/Locale: UTC, UTC+09:00, 다국어(ko, en, ja)
- 이벤트 데이터는 contract의 `event_id`를 사용

## 1) 토큰 수명/무효화

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| invalid 토큰 처리 | invalid 토큰으로 push 발송 | APNs/FCM invalid 응답 수신 | token 상태가 `REVOKED`, 즉시 삭제 후보 등록 | P1 | 서버 로그 + 상태 조회 |
| 장기 미사용 토큰 | last_seen 35일 경과 | purge job 실행 | stale 토큰 정리 스케줄 처리 | P2 | 배치 로그 |

## 2) 중복 알림

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| 이벤트 멱등성 | 동일 event_id로 재시도 3회 | 서버 수신 및 클라 수신 | 서버는 1건 처리, 클라는 1회만 표시 | P1 | 중복 테스트 로그 |
| collapse 동작 | 동일 collapse_key 이벤트 급증 | 푸시 수신 | 최종 상태 우선 노출 | P2 | 수신 로그 |

## 3) 보안/PII

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| PII 누락 검사 | payload에 금지 키 존재 | Contract lint 실행 | lint 실패 | P1 | lint 리포트 |
| 위조 토큰 | 잘못된 action_token | 앱 수신 처리 | 처리 거부 + 경고 로그 | P1 | 보안 테스트 로그 |

## 4) iOS 백그라운드 정책

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| silent push 동작 | silent payload 수신 | 앱이 background | 알림만 트리거, heavy 작업 미수행 | P1 | BGTask 모니터링 |
| foreground 전환 | 포그라운드 전환 | 동기화 큐 실행 | 데이터 pull sync 수행 | P1 | 동기화 로그 |

## 5) 오프라인 정합성

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| 오프라인 복귀 | 메시지 수신 중 오프라인 | 앱 재기동/복귀 | 5분 내 정합성 복구 | P1 | E2E 결과 |

## 6) 로컬라이제이션/시간

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| locale fallback | 미지원 locale 전달 | 렌더링 | 기본 locale fallback 동작 | P2 | 스냅샷/테스트 |
| timezone 처리 | UTC payload 수신 | 표시 | 로컬 시간으로 변환해 표시 | P1 | 타임존 테스트 |

## 7) 동의/철회

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| 동의 철회 | 사용자가 알림 동의 해제 | 즉시 | 1분 내 발송 중단 | P1 | 감사 로그 + API 응답 |

## 8) 정책/앱스토어

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| 체크리스트 | 릴리즈 전 정책 점검 | PR 제출 | 미확인 항목 0개 | P1 | 체크리스트 첨부 |

## 9) 모니터링/운영 대응

| 항목 | Given | When | Then | 우선순위 | 증빙 |
|---|---|---|---|---:|---|
| invalid spike 대응 | invalid 비율 임계치 초과 | 경보 발동 | 15분 내 티켓/대응 기록 | P1 | Alert + Incident log |

## 10) 회귀 지표

- PR 단위로 다음 KPI 체크:
  - `invalid_token` 감소 추세
  - `dup_drop_rate`
  - `sync_lag_ms` p95
  - `open_rate`, `action_rate`

## 실패 기준

- P1 실패: PR merge 불가
- P2 실패: 재현 후 수정 의무
- 회귀율 허용치 초과 시 Hotfix 배포 필요

