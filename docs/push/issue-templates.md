# Push 이슈 템플릿 (10개 고민 전용)

버전: 1.0  
작성일: 2026-06-11  

각 항목은 `docs/push/contract.md`, `policy.md`, `qa-matrix.md`, `runbook.md`와 연계하여 생성한다.  
모든 항목의 종료는 `DoD`와 `증빙`이 제출되어야 한다.

## 공통 템플릿

### 필수 필드
- **제목**: `[Push] # - 내용`
- **영향 범위**: Backend / Mobile / QA / SRE
- **가정**: 해당 이슈가 기존 흐름(푸시 발송, 동기화, 동의, 알림 렌더링)에 미치는 영향
- **DoD**: 정량 목표
- **Acceptance Criteria**: 테스트/로그/지표로 증빙 가능한 조건
- **Evidence**: 링크 1개 이상(테스트/로그/알람 대시보드)
- **Rollback**: 실패 시 되돌릴 정책

---

## 1) 토큰 무효화 처리 강화
- **제목**: `[Push] token lifecycle invalid 처리 강화`
- **담당**: Backend Lead
- **DoD**:
  - APNs/FCM invalid 처리 지연 60초 이내
  - 30일 미사용 토큰 purge 완료율 95% 이상
- **증빙**:
  - invalid_count 전환 로그
  - purge job 실행 로그/건수

## 2) 이벤트 멱등성 구축
- **제목**: `[Push] event_id 멱등성 + 중복 표시 방어`
- **담당**: Backend + Mobile
- **DoD**:
  - `event_id` unique 처리로 중복 insert 차단
  - 사용자 노출 중복률 0.1% 미만
- **증빙**:
  - 중복 이벤트 통합 테스트 결과
  - 클라 dedupe cache 히트율 로그

## 3) PII/보안 강화
- **제목**: `[Push] payload 최소화 및 액션 토큰 보안`
- **담당**: Backend Security
- **DoD**:
  - 보안 스캔에서 PII 금지 키 탐지 0건
  - 위조 token 처리율 100%
- **증빙**:
  - 보안 lint 리포트
  - 보안 테스트 결과

## 4) iOS 백그라운드 정책 정합
- **제목**: `[iOS] silent push 처리 분리 및 BGTask 정책 적용`
- **담당**: iOS Lead
- **DoD**:
  - silent push 수신 시 네트워크 heavy 처리 없음
  - background overrun 경보 미발생
- **증빙**:
  - iOS 런타임 로그
  - BGTask 통계

## 5) 오프라인 정합성 개선
- **제목**: `[Sync] 앱 복귀 시 pull sync 강제화`
- **담당**: Mobile + Backend
- **DoD**:
  - 앱 복귀 5분 내 동기화 오차율 < 1%
- **증빙**:
  - 오프라인/복귀 E2E 테스트 결과
  - sync_lag_ms 추적

## 6) 다국어/시간 처리 정합
- **제목**: `[i18n] key/args 렌더링 및 시간대 처리 표준화`
- **담당**: Frontend/iOS 팀
- **DoD**:
  - locale 미매칭 fallback 동작 100%
  - timezone 변환 회귀 테스트 pass
- **증빙**:
  - i18n 테스트 리포트
  - 시간대 시나리오 로그

## 7) 게이트 강화
- **제목**: `[CI] Push PR Gate 4단계 파이프라인 도입`
- **담당**: QA Lead
- **DoD**:
  - Contract/보안/단위통합/E2E 실패 시 merge block
- **증빙**:
  - workflow 설정 diff
  - 차단 시나리오 테스트 로그

## 8) 모니터링/알람 고도화
- **제목**: `[SRE] push 운영 지표 및 incident 응답 체계`
- **담당**: SRE
- **DoD**:
  - invalid/sync_lag/dup_drop/bad_signature 지표 보드 구성
  - P1 15분 대응 티켓 기록
- **증빙**:
  - 알람 룰 링크
  - incident 처리 로그

## 9) 동의/철회 동기화
- **제목**: `[Consent] 철회 즉시 발송 중단 + 감사로그`
- **담당**: Backend
- **DoD**:
  - 철회 후 1분 이내 발송 중단
- **증빙**:
  - consent audit 로그
  - 발송 금지 재검증 로그

## 10) App Store/정책 체크
- **제목**: `[Policy] 권한/알림 카테고리 정합성 점검`
- **담당**: PM + Tech Lead
- **DoD**:
  - 출시 전 체크리스트 100% 완료
- **증빙**:
  - 체크리스트 첨부 및 승인 이력

