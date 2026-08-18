# Push 운영 Playbook (10가지 고민 종료 제안)

이 문서는 `dflh-saf-v2` 푸시 과제에서 "10가지 고민"을 실제로 **종료(Closed)** 상태로 만들기 위한 실무 실행안이다.
우선순위는 구현 완성도가 아니라 `수치 DoD + 증빙 + 게이트 + 대응 기록`이다.

## 1) 종료 원칙 (Closed Rule)

항목이 `Closed`가 되려면 다음 4조건이 모두 충족되어야 한다.

- 수치 DoD 충족 (예: 60초, 5분, 1분 등)
- 증빙 링크 최소 1개 이상(테스트/로그/모니터링)
- PR Gate 체크리스트 완료
- Runbook 대응 기록 등록(경보/인시던트가 있었을 경우)

## 2) 10가지 고민별 RACI와 클로저 기준

| # | 고민 | DoD (최소 기준) | Evidence(증빙) | R | A | C | I |
|---|---|---|---|---|---|---|---|
| 1 | 토큰 무효화 | invalid 반영 60초 이내, purge 95% 이상 | 토큰 상태 전이 로그, purge job 로그 | Backend | Tech Lead | QA, SRE | iOS, PM |
| 2 | 중복 알림 | 사용자-이벤트 노출 1회 | 멱등성 테스트, dedupe 지표 | Backend, iOS | Tech Lead | QA | SRE |
| 3 | PII/보안 | 금지 키 탐지 0건, 위조 토큰 처리 100% | security lint, 보안 테스트 | Backend, Security | Tech Lead | QA | SRE |
| 4 | iOS 백그라운드 | silent에서 heavy 작업 0건, overrun 경보 미초과 | BGTask 로그, background 실패율 | iOS | Tech Lead | SRE | Backend |
| 5 | 오프라인 정합성 | 앱 복귀 5분 내 sync 정합화 | offline/재접속 E2E, sync_lag 지표 | Mobile, Backend | Tech Lead | QA | SRE |
| 6 | 로컬라이제이션/시간 | locale 누락 0건, UTC→local 변환 pass | i18n/timezone 테스트 | Frontend, iOS | Tech Lead | QA | Backend |
| 7 | 테스트 게이트 | Contract/보안/Unit/Integration/E2E 통과 | CI 링크 | QA | Tech Lead | Backend, iOS | SRE |
| 8 | 모니터링 대응 | 핵심 경보 발생 시 15분 내 대응 기록 | 알람 룰, incident 링크 | SRE | Tech Lead | QA | PM |
| 9 | 동의/철회 | 철회 후 1분 내 발송 차단 | 동의 감사로그 + 발송 판정 로그 | Backend | Tech Lead | QA, SRE | iOS |
|10| 앱스토어 정책 | 릴리즈 전 체크리스트 100% 완료 | 승인 체크리스트 링크 | PM, Tech Lead | Tech Lead | Legal, QA | SRE |

## 3) 일일/주간 운영 루프

### 일일 스탠드업 (15분)
- 진행 중인 이슈 10개 항목 중 매핑 상태 점검
- 각 항목에 대해 `Blocked`, `DoD`, `증빙` 상태 업데이트
- 임계치 경보 발생 시 runbook 트리거 및 티켓 생성 확인

### 주간 게이트 리뷰(1회)
- Open 항목 중 PR Gate 미충족 건이 있으면 블록
- 증빙 누락 항목은 다음 스프린트로 이월 금지
- 재발 원인 하나당 정책/문서/테스트 한 축으로 되감기

## 4) 이슈 템플릿 사용 기준(최소 입력)

- 관련 고민 항목: `1~10` 중 1개 이상 선택
- DoD: 수치 형태(예: `60초`, `5분`, `100%`)로 선기입
- 증빙 계획: 테스트/로그/알람 링크 항목(체크박스)
- 롤백 조건: 1차 실패 시 복구 기준(구현 이전 상태 or feature flag) 명시

## 5) PR 템플릿 자동 차단 규칙

아래 중 하나라도 미기재면 PR은 merge 금지.

- Contract Test 링크 미등록
- Security Scan 링크 미등록
- QA 항목 링크 미등록
- Runbook 링크 미등록
- 문제되는 10가지 항목 체크리스트 미기입

## 6) 2주 실행 계획(바로 시행)

1일차: Playbook 배포 + 템플릿/라벨 고정  
2일차: CI에 contract lint + security lint + push e2e를 필수 게이트로 추가  
3일차: 핵심 지표 패널/알람 임계치 등록  
4~7일차: 10개 항목별 이슈 생성 및 소유자 지정  
8~10일차: PR 제출, 증빙 완비, 게이트 검증 반복  
11~14일차: runbook 실제 대응 드릴 + 미종결 항목 교차 점검

## 7) 승인 기준(요약)

PR은 아래 4개를 충족한 항목만 `Merged` 허용한다.

- 변경 영향이 있는 항목 10개 중 DoD 충족
- 테스트/보안/운영 증빙 링크 존재
- PR Gate 체크 100% 통과
- runbook 또는 incident 대응 기록 정합
