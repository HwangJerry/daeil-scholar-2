# PR Gate Workflow 샘플 (참고용)

`docs/push` 기준으로 `GitHub Actions`에서 사용할 수 있는 최소 실무 형태 샘플이다.
실제 레포 스택에 맞게 실행 job과 커맨드를 교체해 사용한다.

## 1) 목표

- PR body/체크리스트가 비어 있으면 fail
- 최소 1개 이상의 10가지 고민 항목 체크 의무
- Contract/보안/QA/Monitoring 증빙 링크 미기재 시 fail
- runbook 미기재 항목은 예외 조건(영향 없음)이어야 함

## 2) 샘플 워크플로우

```yaml
name: push-pr-gate

on:
  pull_request:
    types: [opened, edited, synchronize]

jobs:
  push-gate-check:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Validate push PR checklist
        env:
          PR_BODY: ${{ github.event.pull_request.body }}
        run: |
          node <<'NODE'
          const body = process.env.PR_BODY || "";
          const mustHave = [
            "Contract Test",
            "Security Scan",
            "QA",
            "Monitoring",
          ];
          const hasConcern = /1\)|2\)|3\)|4\)|5\)|6\)|7\)|8\)|9\)|10\)/.test(body);
          const missing = mustHave.filter(label => !new RegExp(`${label}:\\s*\\S`, "i").test(body));
          const missingRunbook = /Runbook/gi.test(body) ? [] : ["Runbook"];

          if (!hasConcern) {
            console.error("10개 고민 중 최소 1개 체크 필요");
            process.exit(1);
          }
          if (missing.length) {
            console.error(`필수 증빙 누락: ${missing.join(", ")}`);
            process.exit(1);
          }
          if (missingRunbook.length) {
            console.error("Runbook 항목이 없으면 운영/대응 트래킹 누락 가능성");
            process.exit(1);
          }
          console.log("Push PR gate check pass");
          NODE
```

## 3) 운영용 체크 포인트

- 실패 로그를 PR 리뷰 코멘트로 변환하려면 `peter-evans/create-or-update-comment`로 결합
- CI 단계별 실패(Contract/Security/QA)은 별도 필수 브랜치 제한 규칙으로 병합 차단
- 긴급 hotfix는 `critical` 라벨과 Tech Lead 승인 1인 승인으로 우회 가능하도록 별도 정책 분기

