---
name: harness
description: "5단계(clarify-context → gather → plan → generate → evaluate) 구조화된 구현 워크플로. 새 기능 구현, 리팩토링, 복잡한 태스크 요청 시 자동 권장. 사용자가 '구현해줘', '만들어줘', '기능 추가', 'implement', 'build a', '리팩토링' 등의 복잡한 요청을 할 때, 또는 /harness로 명시 호출할 때 사용."
---

# Harness — 5-Stage Implementation Workflow

복잡한 구현 요청을 일관된 품질로 처리하기 위한 5단계 강제 워크플로다. 즉흥적인 코딩으로 인한 요구사항 누락·맥락 오염·검증 누락을 방지한다.

## When to Use

- 사용자가 새 기능, 리팩토링, 복잡한 버그 수정을 요청할 때
- 여러 파일·레이어에 걸친 변경이 필요할 때
- `/harness` 슬래시 명령으로 명시 호출되었을 때

**사용하지 않을 때**: 한 줄 수정, 오타 수정, 단순 질의응답, 파일 읽기/탐색만 하는 작업.

## The 5 Stages

| 단계 | 수행자 | 산출물 |
|------|--------|--------|
| 1. clarify-context | 메인 (AskUserQuestion) | `clarify.md` |
| 2. gather | `Explore` 서브에이전트 | `gather.md` |
| 3. plan | `Plan` 서브에이전트 | `plan.md` |
| 4. generate | 메인 (Edit/Write) | `generate.md` |
| 5. evaluate | 메인 (Bash) + `Explore` | `evaluate.md` |

각 단계의 상세 절차는 `stages/<name>.md`에 정의되어 있다. **각 단계 시작 시 해당 파일을 Read로 반드시 먼저 읽고** 절차를 정확히 따른다.

- [stages/clarify-context.md](stages/clarify-context.md)
- [stages/gather.md](stages/gather.md)
- [stages/plan.md](stages/plan.md)
- [stages/generate.md](stages/generate.md)
- [stages/evaluate.md](stages/evaluate.md)

## Run Lifecycle

### 1. Initialize Run

스킬 호출 직후, 가장 먼저 run-id를 생성하고 상태 디렉토리를 만든다:

```bash
RUN_ID=$(date +%Y%m%d-%H%M%S)
RUN_DIR=".claude/harness/${RUN_ID}"
mkdir -p "${RUN_DIR}"
```

`templates/state.json.tmpl`을 복사해 `${RUN_DIR}/state.json`을 생성하고 `run_id`, `started_at`을 채운다.

### 2. Run Stages Sequentially

`stages/clarify-context.md`부터 `evaluate.md`까지 순서대로 실행. 각 단계 완료 시:

- 산출물 md를 `${RUN_DIR}/`에 저장
- `state.json`의 `current_stage`, `stages_completed`를 갱신

### 3. Loop or Exit

`evaluate` 결과가 fail이면 `state.json`의 `loop_count`를 증가시키고 `plan` 단계로 자동 복귀.

## Loop Control

- 기본 `max_loops = 3`. `state.json`에서 관리.
- `loop_count >= max_loops` 도달 시: AskUserQuestion으로 "강제 종료 / 추가 루프 / 수동 개입" 선택을 받는다.
- 루프 시 이전 `plan.md`, `generate.md`, `evaluate.md`는 보존하고 신규 차수는 `plan.v2.md`, `generate.v2.md`, `evaluate.v2.md` 식으로 버전 접미사를 붙인다.

## State File Schema

`${RUN_DIR}/state.json`:

```json
{
  "run_id": "20260419-225400",
  "current_stage": "evaluate",
  "loop_count": 1,
  "max_loops": 3,
  "stages_completed": ["clarify-context", "gather", "plan", "generate"],
  "started_at": "2026-04-19T22:54:00+09:00",
  "user_request": "최초 사용자 요청 원문 (한 문장으로 압축)"
}
```

## Outputs

모든 산출물은 `.claude/harness/<run-id>/` 아래에 저장된다:

```
.claude/harness/20260419-225400/
├── state.json
├── clarify.md
├── gather.md
├── plan.md            (루프 시 plan.v2.md, plan.v3.md ...)
├── generate.md        (루프 시 generate.v2.md ...)
└── evaluate.md        (루프 시 evaluate.v2.md ...)
```

이 디렉토리는 `.gitignore`에 추가하는 것을 권장한다 (실행 산출물이라 커밋 대상 아님).

## Important Rules

1. **단계를 건너뛰지 않는다.** clarify-context 없이 gather로 가지 말 것.
2. **clarify-context는 메인 에이전트만 수행한다.** AskUserQuestion이 서브에이전트에서 작동하지 않기 때문.
3. **gather/plan은 반드시 서브에이전트에 위임한다.** 메인 컨텍스트 보호 목적.
4. **generate는 메인 에이전트가 직접 수행한다.** Edit/Write 권한과 hook 피드백을 받기 위해.
5. **evaluate fail → plan으로 복귀.** clarify로 가지 않는다 (요구사항은 변하지 않음).
6. **state.json을 반드시 갱신한다.** 중간 실패 시 재개 가능하도록.
