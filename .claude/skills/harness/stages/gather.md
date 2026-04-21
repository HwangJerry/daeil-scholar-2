# Stage 2 — gather

기존 코드베이스에서 재사용 가능한 패턴·함수·컴포넌트를 찾는다. **새로 만들기 전에 있는 것을 먼저 본다**는 CLAUDE.md 원칙을 강제하는 단계.

## Performer

`Explore` 서브에이전트 (subagent_type=Explore). 메인 컨텍스트가 코드 탐색 결과로 가득 차지 않도록 위임.

## Inputs

- `${RUN_DIR}/clarify.md` — Goal, Constraints, Acceptance Criteria

## Procedure

### 1. Read clarify.md

`${RUN_DIR}/clarify.md`를 Read로 읽고 핵심을 파악한다.

### 2. Launch Explore agent

Agent 도구를 다음과 같이 호출:

```
subagent_type: Explore
description: "Gather reusable code for <짧은 작업 요약>"
prompt: |
  다음 작업을 위한 기존 코드/패턴을 조사해서 재사용 후보를 보고해줘.

  ## Goal
  <clarify.md의 Goal 그대로>

  ## Constraints
  <clarify.md의 Constraints 그대로>

  ## What to find
  1. 동일/유사 기능을 이미 구현한 함수/컴포넌트/유틸리티 (있으면 재사용)
  2. 이 작업에서 따라야 할 기존 패턴 (예: 백엔드 핸들러 구조, React Query 훅 패턴, API 응답 형식)
  3. 영향 받을 파일 (수정해야 할 곳)
  4. 관련 타입/모델 정의 위치
  5. 관련 테스트 파일 (있다면)

  ## How to report
  - 각 항목에 **파일 경로:라인 번호** 포함
  - 재사용 가능한 함수는 시그니처도 함께
  - 200-300 단어 이내로 요약
  - 관련 코드를 못 찾았으면 "없음"이라고 명시 (추측 금지)

  CLAUDE.md를 먼저 읽어 프로젝트 구조와 컨벤션을 파악한 뒤 조사를 시작해줘.
```

`thoroughness`는 작업 복잡도에 따라 "quick" / "medium" / "very thorough" 중 선택. 기본은 "medium".

### 3. Write gather.md

Explore의 응답을 `${RUN_DIR}/gather.md`에 다음 구조로 Write:

```markdown
# Gather — <run-id>

## Reusable Functions / Components
- `<path/to/file.go:42>` — `func DoX(ctx, id) error` — 이번 작업에서 그대로 사용 가능
- ...

## Patterns to Follow
- 백엔드 핸들러: `backend/internal/handler/feed.go` 패턴
- ...

## Files to Modify
- `frontend/src/pages/...` — <왜 수정해야 하는지>
- ...

## Related Types / Models
- `backend/internal/model/...` — <관련 구조체>

## Related Tests
- `<없음 또는 경로>`

## Gaps
- <조사 중 못 찾은 것, plan 단계에서 새로 만들어야 할 것>
```

### 4. Update state.json

`current_stage`를 `"plan"`으로, `stages_completed`에 `"gather"` 추가.

## Output

`${RUN_DIR}/gather.md` — plan 단계의 입력.

## Common Pitfalls

- **메인이 직접 grep/glob을 돌림**: 위임이 의미 없어진다. 반드시 Explore 에이전트로.
- **재사용 후보를 빠뜨림**: 비슷한 이름의 함수가 있는지 충분히 검색하지 않으면 plan 단계에서 중복 구현이 나온다.
- **추측 보고**: Explore가 "아마 ~~에 있을 것"이라고 답하면 받지 말고 "확인되지 않음"으로 처리.
