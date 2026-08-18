# Pull Request Template (Push 변경용)

## Summary
- 변경 항목:
- 영향 이벤트 타입:
- 변경 범위: (Backend / iOS / QA / SRE)

## Related Docs
- [Push Contract](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/contract.md)
- [Push Policy](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/policy.md)
- [Push PR Gate](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/pr-gate.md)
- [QA Matrix](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/qa-matrix.md)
- [Runbook](/Users/jerryhwang/Workspace/03_daeil/dflh-saf-v2/docs/push/runbook.md)

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

## 10개 고민 매핑 (적용된 항목만 체크)
- [ ] 1) 토큰 관리
- [ ] 2) 중복 알림
- [ ] 3) 보안/PII
- [ ] 4) iOS 백그라운드
- [ ] 5) 오프라인 정합성
- [ ] 6) 로컬라이제이션
- [ ] 7) 테스트 게이트
- [ ] 8) 모니터링 대응
- [ ] 9) 동의/철회
- [ ] 10) App Store 정책

## Rollback Plan
- 실패 시 롤백 절차:
- 데이터 이전 필요 시:

