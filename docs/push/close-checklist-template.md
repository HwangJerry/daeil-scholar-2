# Push Close 체크리스트 템플릿 (10가지 고민 전용)

이 문서는 PR 승인 전에 매 PR이 10가지 고민에서 실제로 `Closed` 기준을 만족했는지 확인하기 위한 템플릿이다.
아래 항목은 PR 설명 또는 리뷰 코멘트에 그대로 붙여넣어 사용한다.

## 1) 기본 체크

- [ ] 영향 이슈 존재: `[Push]` prefix 이슈 또는 대응 티켓 링크
- [ ] 변경 영향 항목 매핑: `1~10` 중 해당 항목 1개 이상 체크
- [ ] DoD 선기입 확인: 각 항목 수치 목표가 명시되어 있는가
- [ ] Runbook 매핑: 실행/대응 시나리오 번호 또는 "N/A(영향 없음)" 명시

## 2) 10개 항목별 증빙 체크

| 항목 | 상태 | DoD | 증빙 링크 | 비고 |
|---|---|---|---|---|
| 1) 토큰 관리 |  | invalid 반영 ≤ 60초 / purge ≥ 95% |  |  |
| 2) 중복 알림 |  | 사용자/이벤트 노출 1회 |  |  |
| 3) 보안/PII |  | PII 0건 / 위조 토큰 100% 처리 |  |  |
| 4) iOS 백그라운드 |  | silent 과부하 0 / overrun 미초과 |  |  |
| 5) 오프라인 정합성 |  | 복귀 5분 내 sync_lag 정합 |  |  |
| 6) 로컬라이제이션 |  | locale fallback 누락 0 |  |  |
| 7) 테스트 게이트 |  | Contract/보안/단위/통합/E2E pass |  |  |
| 8) 모니터링 대응 |  | 15분 내 대응 이력 존재 |  |  |
| 9) 동의/철회 |  | 1분 내 발송 중단 |  |  |
|10) App Store 정책 |  | 체크리스트 100% 완료 |  |  |

## 3) CSV 업로드용 구조(복붙 가능)

```csv
issue_id,owner,concern_ids,dod,proof_links,status,runbook_scenario,closed_at
PUSH-###,Backend,1,invalid<=60s;purge>=95%,https://ci.example/link;https://log.example/link,closed,runbook-1,2026-06-11T00:00:00Z
PUSH-###,iOS,4;2,background_overrun=0;heavy_task=0,https://bg.example/link,closed,runbook-5,
```

## 4) 주간 리뷰 기준

- `[Open]` 상태라면 `PR Gate`에서 `Ready for merge` 체크를 허용하지 않는다.
- 증빙 링크 1개라도 빈칸이면 `Blocked` 처리.
- `doctemplate` / `qa` / `security` / `runbook` 중 1개라도 누락이면 `Reopen` 처리.

## 5) 운영 알람 메시지 템플릿

```
[Push Close] #PR 번호
- 적용 고민: [1,2,3...]
- DoD: (수치)
- 증빙: contract=<링크> security=<링크> qa=<링크> monitoring=<링크>
- 운영 대응: runbook=#, incident=#(필요 시)
```

