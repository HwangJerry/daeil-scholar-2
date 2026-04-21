# Stage 3 — plan

clarify + gather 결과를 바탕으로 **파일 단위 구현 계획**을 만든다. generate 단계가 이 계획을 그대로 실행할 수 있을 만큼 구체적이어야 한다.

## Performer

`Plan` 서브에이전트 (subagent_type=Plan).

## Inputs

- `${RUN_DIR}/clarify.md`
- `${RUN_DIR}/gather.md`
- (루프 시) 직전 차수의 `evaluate.v<N>.md` — 무엇이 fail이었는지

## Procedure

### 1. Determine version suffix

- 첫 plan: 파일명 `plan.md`
- 루프된 plan: `plan.v2.md`, `plan.v3.md` ... (`state.json.loop_count + 1` 사용)

### 2. Launch Plan agent

```
subagent_type: Plan
description: "Plan implementation for <짧은 작업 요약>"
prompt: |
  다음 정보를 바탕으로 구현 계획을 작성해줘.

  ## Context (from clarify.md)
  <clarify.md 내용 그대로 붙임>

  ## Existing Code (from gather.md)
  <gather.md 내용 그대로 붙임>

  ## (루프인 경우) Previous Failure
  <evaluate.v<N-1>.md의 fail 사유 — 이번 plan은 이걸 반드시 해결해야 함>

  ## Required Output Structure
  파일별로 다음을 포함해 markdown으로 작성:

  ### Files to Create
  - `<path>` — 목적 1줄, 파일 크기 예측 (150-500라인 권장)
    - 헤더 주석: `// FileName — brief purpose description` (필수)
    - 주요 export: 함수/타입 목록
    - 단일 책임 명시: 이 파일이 책임지는 한 가지 도메인/관심사

  ### Files to Modify
  - `<path>:<line>` — 무엇을 어떻게 바꿀지
    - 변경 전/후 시그니처 (해당되면)

  ### Reused from gather.md
  - `<path:line>` — `<함수명>` — 어떻게 호출할지

  ### Test Strategy
  - 단위 테스트 추가 위치 (있다면)
  - 수동 검증 절차

  ### CLAUDE.md Compliance Checklist
  - [ ] 모든 .ts/.tsx 파일에 헤더 주석 (validate-code-file.sh)
  - [ ] 파일당 150-500 라인 (validate-code-file.sh)
  - [ ] 단일 책임 (CLAUDE.md Rule 2)
  - [ ] UI는 UI_DESIGN_DOC.md 디자인 토큰 준수 (해당 시)
  - [ ] MariaDB 10.1 제약 (CTE/Window function/JSON 컬럼 사용 안 함)
  - [ ] 백엔드 레이어 분리 (handler→service→repository)
  - [ ] 기존 패턴 재사용 (gather.md에서 발견한 것)

  ### Acceptance Criteria Mapping
  clarify.md의 각 Acceptance Criteria가 어떤 파일/변경으로 충족되는지 매핑.

  ## Constraints
  - 새로 만들기 전에 gather.md의 재사용 후보를 먼저 활용
  - 추측 금지 — 모르는 부분은 "확인 필요"로 표시
  - YAGNI: 미래 가능성을 위한 추상화 금지
```

### 3. Save plan output

Plan 에이전트가 자신의 plan 파일에 결과를 쓰면, 그 내용을 Read해서 `${RUN_DIR}/plan.md` (또는 v2, v3...)로 복사. **Plan 에이전트는 자체 plan 디렉토리에 쓰므로 반드시 옮겨와야 한다.**

만약 Plan 에이전트가 응답 본문으로 plan을 반환했다면 그 내용을 그대로 Write.

### 4. Update state.json

`current_stage`를 `"generate"`로, `stages_completed`에 `"plan"` 추가 (이미 있으면 중복 추가 안 함).

## Output

`${RUN_DIR}/plan.md` (또는 `plan.v2.md` 등) — generate 단계의 실행 사양.

## Common Pitfalls

- **추상적인 plan**: "유저 등록 기능 구현" 같은 한 줄 plan은 generate가 또 추측해야 함. 파일·함수·시그니처까지 내려가야 함.
- **CLAUDE.md 체크리스트 빠뜨림**: validate-code-file.sh에 걸려서 다시 수정하게 됨.
- **루프 plan에서 이전 실패를 무시**: Previous Failure를 명시적으로 해결하지 않으면 같은 evaluate fail이 반복됨.
