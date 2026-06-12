---
name: Push 10가지 고민 이슈
about: Push 안정성 이슈를 10가지 고민 기준으로 등록
title: "[Push] "
labels: [push, operations]
---

## 1) 제목
[Push] 팀명 - 항목명

## 2) 영향 범위
- Backend / Mobile / iOS / QA / SRE

## 3) 가정
해당 변경이 기존 흐름(푸시 발송, 동기화, 동의, 알림 렌더링)에 미치는 영향:

## 4) 적용 고민 항목 (1개 이상)
- [ ] 1) 토큰 관리
- [ ] 2) 중복 알림
- [ ] 3) 보안/PII
- [ ] 4) iOS 백그라운드
- [ ] 5) 오프라인 정합성
- [ ] 6) 로컬라이제이션/시간
- [ ] 7) 테스트 게이트
- [ ] 8) 모니터링 대응
- [ ] 9) 동의/철회
- [ ] 10) App Store 정책

## 5) DoD (수치)
- Example: invalid 반영 60초 이내, purge 95% 이상

## 6) Concern DoD (수치) (적용 항목만 작성)
- 1) 
- 2) 
- 3) 
- 4) 
- 5) 
- 6) 
- 7) 
- 8) 
- 9) 
- 10) 

## 7) Acceptance Criteria
- 테스트/로그/지표로 증빙 가능한 조건:

## 8) Evidence (운영 포함)
- 링크:
  - Contract Test:
  - Security Scan:
  - QA:
  - Monitoring:
  - Runbook/incident:

## 9) Rollback
- 실패 시 복구 방안:

## 10) 참고 문서
- [Push Contract](../docs/push/contract.md)
- [Push Policy](../docs/push/policy.md)
- [Push PR Gate](../docs/push/pr-gate.md)
- [QA Matrix](../docs/push/qa-matrix.md)
- [Runbook](../docs/push/runbook.md)
