# Push 운영 가이드 (10가지 고민 정리)

이 폴더의 문서는 `dflh-saf-v2`의 푸시 시스템을 실무 기준으로 운영하기 위한 표준입니다.

목표는 10가지 고민을 기능 단위가 아니라, **종료 가능한 DoD + 증빙 + 게이트 체계**로 닫는 것입니다.

## 0) 먼저 읽을 순서

- [contract.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/contract.md)
- [policy.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/policy.md)
- [pr-gate.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/pr-gate.md)
- [qa-matrix.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/qa-matrix.md)
- [runbook.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/runbook.md)
- [operating-playbook.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/operating-playbook.md)
- [close-checklist-template.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/close-checklist-template.md)
- [implementation-plan-14days.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/implementation-plan-14days.md)
- [branch-protection-sample.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/branch-protection-sample.md)
- [fail-recovery-playbook.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/fail-recovery-playbook.md)
- [pr-gate-workflow-sample.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/pr-gate-workflow-sample.md)
- [issue-templates.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/issue-templates.md)
- [pull-request-template.md](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/pull-request-template.md)

## 1) 목표

- 10가지 고민(토큰무효화, 중복알림, PII, iOS백그라운드, 오프라인정합성, 로컬라이제이션, 테스트게이트, 모니터링, 동의·철회, 정책리스크)을
  계약-정책-테스트-운영의 네 축으로 관리한다.
- 수정은 기능 구현이 아니라, **증빙이 붙은 DoD 충족**으로만 종료한다.

## 2) 운영 흐름

1. 고민별 이슈 생성
1. 이슈 템플릿(`issue-templates.md`)에 DoD와 증빙 계획 기록
1. 구현 후 PR 작성
1. PR 템플릿(`pull-request-template.md`)에 관련 항목과 링크 채움
1. PR Gate 조건 충족 확인
1. merge
1. 운영 대응/모니터링 점검

## 3) 10가지 고민 종료 기준(요약)

1. 토큰 관리
   - DoD: invalid 반영 60초 이내, purge 95% 이상
1. 중복 알림
   - DoD: 이벤트당 사용자 노출 1회
1. 보안/PII
   - DoD: payload PII 스캔 0건, 위조 토큰 100% drop
1. iOS 백그라운드
   - DoD: 백그라운드 과부하 경보 임계치 미초과
1. 오프라인 정합성
   - DoD: 복귀 후 5분 내 정합도 회복
1. 로컬라이제이션/시간
   - DoD: locale 누락 0건, UTC→로컬 변환 검증
1. 테스트 게이트
   - DoD: Contract/보안/Unit+Integration/E2E 통과
1. 모니터링
   - DoD: 주요 지표 임계치 경보 대응 기록
1. 동의/철회
   - DoD: 철회 후 1분 내 발송 중단
1. App Store 정책
   - DoD: 릴리즈 전 정책 체크리스트 100% 통과

## 4) 증빙 룰

- PR은 아래 링크/로그가 없는 경우 미완료로 간주한다.
- 모든 항목은 정량 로그 또는 테스트 리포트 링크를 반드시 첨부한다.
- 스크린샷만 첨부할 경우, 테스트 단계/쿼리/임계치 판단식도 함께 기재한다.

## 5) 빠른 점검 리스트

- PR Gate 체크박스가 모두 체크되었는가?
- 10가지 고민 중 적용 항목에 DoD가 기입되었는가?
- QA Matrix의 관련 시나리오가 실행되었는가?
- Runbook 시나리오 링크가 첨부되었는가?
- 운영 알람 임계치 변화가 사전 협의되었는가?

## 6) 책임자(Role) 배치

- Backend Lead: 토큰/멱등성/보안 페이로드/동의 처리
- iOS Lead: 백그라운드/딥링크/클라이언트 중복 표시
- QA Lead: QA Matrix와 PR 게이트 검증
- SRE: 모니터링/알람/Runbook 응답 체계

## 7) 권장 다음 액션

1. GitHub PR 템플릿을 [pull-request-template.md](pull-request-template.md) 형식으로 등록한다.
1. 이슈 템플릿(`issue-templates.md`)을 스프린트 보드 생성 프로세스에 반영한다.
1. 위 문서에 있는 KPI 임계치와 경보 룰을 운영 모니터링에 등록한다.
