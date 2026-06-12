# 14일 실행 플랜 (10가지 고민 종료용)

목표: [운영 Playbook](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/operating-playbook.md)의 기준으로 10가지 고민을 실제로 `Closed` 상태로 전환.

## 0) 실행 원칙
- 이슈/PR/PR Gate만으로도 다음이 가능해야 한다:  
  `이슈 생성 -> PR 게이트 -> 테스트/보안 증빙 -> Runbook 대응 기록 -> Close`.
- 항목당 종료 조건은 `수치 DoD + 증빙 + 대응 기록`.

## 1) Day-by-Day

### Day 1: 기준 정렬
- `[close-checklist-template.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/close-checklist-template.md)` 공유
- [Push PR 템플릿](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/.github/PULL_REQUEST_TEMPLATE/push.md), [Issue 템플릿](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/.github/ISSUE_TEMPLATE/push.md) 공지
- [Push PR Gate 워크플로우](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/.github/workflows/push-pr-gate.yml) Dry-run 한 번 수행

### Day 2: 최소 문턱치 정합
- PR Gate 실패 사유가 무엇인지 기준 재점검
- PR 본문 체크 항목에서 오탐/누락 없는지 수정
- 10개 항목 DoD 수치 최종 확정

### Day 3: 게이트/보드 연결
- PR 체크 항목과 스프린트 보드 필드 매핑
- `요청된 항목`, `DoD`, `증빙 링크`, `Close 상태` 필수 컬럼 추가

### Day 4~7: 10개 항목 파일럿
각 항목 당 최소 1건 이슈 생성(총 4~10건).
- 담당자 배정
- `Concerns` 매핑(1~10)
- `DoD` 선입력
- `증빙 플랜` 작성

### Day 8~10: PR/검증 라운드
- 이슈를 PR로 연결하고, 아래 5개 모두 채우고 병합 시도
  - Contract/Schema 반영
  - 보안/테스트 링크
  - 10개 항목 체크
  - 모니터링/Runbook 링크
  - Close 항목 완료
- Gate 미통과 이슈는 즉시 수정 후 재요청

### Day 11~14: 운영 응답/교정
- 실제 알람/incident 한 건 이상 포함 여부 점검
- 회고 기준 수립(재발 원인마다 정책/테스트/감시 한 축으로 반영)
- 미완 항목은 다음 분기 목표로 이월하지 않고 원인 제거 후 재시작

## 2) 주간 KPI(최소)
- 토큰 무효화: invalid 반영 p95 ≤ 60초
- 중복 알림: 사용자-이벤트 노출 중복률 ≤ 0.1%
- iOS 백그라운드: BG 과부하 경보 회수/시간 = 0 (허용치 초과 시 즉시 핫픽스)
- 오프라인 정합성: 재접속 후 5분 내 sync 정합도 회복률 ≥ 99%
- PII/보안: 금지 키 탐지 0건, 위조 처리율 100%
- 동의 철회: 1분 내 발송 중단

## 3) 의사결정 기준(승인)
- Close 미충족 항목은 병합 금지
- `Runbook` 대응이 필요한 항목에서 운영 로그 누락 시 강제 재개
- 긴급(critical) 경로는 Tech Lead 1인 승인 + 재검증 루프를 거쳐 보완

## 4) 완료 기준
- 14일 내 10개 항목 중 최소 60% 항목이 `Closed`로 기록
- 미클로즈 항목은 원인과 다음 액션이 보드에 남아 있을 것
- PR Gate 실패율 감소 추적(시작 대비 개선)

