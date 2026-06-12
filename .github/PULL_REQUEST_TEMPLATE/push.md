# Push 변경 PR 템플릿 (Close 규칙 적용)

## Summary
- 변경 항목:
- 영향 이벤트 타입:
- 변경 범위: Backend / iOS / QA / SRE
- 이슈 링크: (예: #123)

## Push 관련 문서
- [Push Contract](../../docs/push/contract.md)
- [Push Policy](../../docs/push/policy.md)
- [Push PR Gate](../../docs/push/pr-gate.md)
- [QA Matrix](../../docs/push/qa-matrix.md)
- [Runbook](../../docs/push/runbook.md)
- [Close Checklist](../../docs/push/close-checklist-template.md)

## Push Gate Checklist (필수)
- [ ] Contract/Schema 변경이 `docs/push/contract.md`에 반영됨
- [ ] Policy 영향점이 `docs/push/policy.md`에 반영됨
- [ ] Contract Test 업데이트됨
- [ ] 보안/PII 검사 통과
- [ ] QA Matrix 해당 항목 실행 및 통과
- [ ] Monitoring 항목/알람 영향 분석 첨부
- [ ] Runbook 대응 항목 첨부

## Evidence (각 항목에 링크 1개 이상)
- Contract Test:
- Security Scan:
- Unit/Integration:
- E2E:
- Monitoring/Dashboard:
- Incident 대응(해당 시):
- Runbook 대응 항목:

## 10개 고민 매핑 (적용된 항목만 체크)
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

## Concern DoD (수치) (적용 항목만 작성)
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

## Close Checklist (붙여넣기)
- [ ] 영향 이슈 존재
- [ ] 수치 DoD 선기입
- [ ] 증빙 링크 1개 이상
- [ ] PR Gate 완료
- [ ] Runbook/incident 대응 링크

## Rollback Plan
- 실패 시 롤백 절차:
- 데이터 이전 필요 시:
