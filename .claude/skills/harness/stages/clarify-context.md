# Stage 1 — clarify-context

요구사항을 구조화된 질문으로 확정한다. 여기서 모호한 부분을 남기면 이후 단계가 모두 흔들린다.

## Performer

**메인 에이전트만.** 서브에이전트는 AskUserQuestion을 호출할 수 없으므로 위임 금지.

## Inputs

- 사용자의 최초 요청 텍스트 (스킬을 호출하게 만든 메시지)

## Procedure

### 1. Initialize the run

```bash
RUN_ID=$(date +%Y%m%d-%H%M%S)
RUN_DIR=".claude/harness/${RUN_ID}"
mkdir -p "${RUN_DIR}"
```

`templates/state.json.tmpl`을 Read로 읽고 다음 필드를 채워서 `${RUN_DIR}/state.json`에 Write:

- `run_id`: 위에서 생성한 값
- `current_stage`: `"clarify-context"`
- `loop_count`: `0`
- `max_loops`: `3`
- `stages_completed`: `[]`
- `started_at`: `date -Iseconds` 결과
- `user_request`: 사용자 요청을 한 문장으로 압축

### 2. Identify ambiguities

사용자 요청을 분석하여 다음 차원에서 모호한 점을 추린다:

- **Scope**: 어디까지 변경하나? (예: 백엔드만? 프론트도?)
- **Tech choice**: 어떤 라이브러리/패턴을 쓰나? 기존 것 재사용 가능한가?
- **UX decisions**: UI 변경이 있다면 동작·시각 디자인은?
- **Data shape**: 입출력 형식, DB 스키마 변경 여부
- **Edge cases**: 빈 상태, 에러 처리, 권한

이 중 **가장 결정적인 4개**를 골라 AskUserQuestion 옵션으로 만든다. 자명한 것은 묻지 않는다.

### 3. Call AskUserQuestion (한 번만)

- 질문은 1~4개. 각 질문은 2~4개 선택지.
- 추천하는 옵션은 첫 번째 위치에 두고 `(Recommended)` 접미사.
- **중복 호출 금지.** 한 라운드로 끝낸다. 답변 후 추가로 모호한 점이 발견되면 본문 텍스트로 짧게 확인하고 마무리.

### 4. Write clarify.md

`${RUN_DIR}/clarify.md`를 다음 구조로 Write:

```markdown
# Clarify — <run-id>

## Original Request
> <사용자 원문 그대로 인용>

## Goal
<한두 문장으로 무엇을 달성할지>

## Constraints
- <기술/성능/호환성 제약>
- <CLAUDE.md/팀 규칙 관련>

## Decisions (from AskUserQuestion)
- **<질문 헤더>**: <선택된 답변> — <간단한 근거>
- ...

## Out of Scope
- <명시적으로 이번 작업에서 제외하는 것>

## Acceptance Criteria
- [ ] <검증 가능한 조건 1>
- [ ] <검증 가능한 조건 2>
- [ ] <evaluate 단계가 이걸 보고 pass/fail 판정함>
```

### 5. Update state.json

`current_stage`를 `"gather"`로, `stages_completed`에 `"clarify-context"` 추가.

## Output

`${RUN_DIR}/clarify.md` — 다음 단계(gather)의 입력으로 쓰임.

## Common Pitfalls

- **너무 많은 질문**: 4개 초과는 사용자 피로도를 높인다. 정말 결정적인 것만.
- **자명한 것 묻기**: "TypeScript로 작성할까요?" 같은 건 묻지 말 것 (CLAUDE.md에 이미 명시됨).
- **Acceptance Criteria 누락**: 이게 없으면 evaluate가 무엇을 검증할지 모른다. 반드시 작성.
