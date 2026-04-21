# Stage 4 — generate

`plan.md`(또는 최신 차수 `plan.v<N>.md`)를 따라 실제 코드를 작성한다.

## Performer

**메인 에이전트.** Edit/Write 도구와 validate-code-file.sh hook 피드백을 받아야 하므로 위임 불가.

## Inputs

- `${RUN_DIR}/plan.md` (또는 가장 높은 버전 `plan.v<N>.md`)
- `${RUN_DIR}/gather.md` (재사용 함수 시그니처 확인용)

## Procedure

### 1. Determine version suffix

generate 산출물 파일명도 plan과 동기화:

- 첫 generate: `generate.md`
- 루프 시: `generate.v<N>.md` (`state.json.loop_count + 1`)

### 2. Initialize generate.md

빈 `${RUN_DIR}/generate.md`를 다음 헤더로 Write:

```markdown
# Generate — <run-id> (v<N>)

Plan source: `plan.md` (또는 `plan.v<N>.md`)

## Changes
```

### 3. Execute plan items in order

`plan.md`의 **Files to Create** → **Files to Modify** 순서로 실행. 각 항목마다:

1. **Files to Create**: Write 도구로 파일 생성
   - 헤더 주석 잊지 말 것: `// FileName — brief purpose description`
   - .ts/.tsx 파일은 150-500 라인 권장 (validate-code-file.sh 훅 발동)
2. **Files to Modify**: Read로 먼저 읽고 (Edit는 Read 후에만 가능) Edit 도구로 수정
3. 작업 직후 `generate.md`에 한 항목 append:
   ```markdown
   - [x] `<path>` — <한 줄 설명> — <created | modified>
   ```

### 4. Handle hook feedback inline

`.ts/.tsx` 파일 작성 후 PostToolUse 훅(validate-code-file.sh)이 경고를 반환할 수 있다:

- **[FIX REQUIRED]** 항목은 즉시 인라인 수정. 사용자에게 묻지 않는다 (CLAUDE.md Rule 1, 2와 훅 footer가 그렇게 지시).
- **[INFO ONLY]** (예: 라인 수 30-150 사이)는 무시 가능. 단, 의도적으로 짧은 게 아니라면 합치는 걸 고려.
- 인라인 수정 후 `generate.md`에 짧게 기록: `  - hook fix: split <file> into <fileA>, <fileB>`

### 5. Reuse, don't recreate

`gather.md`에서 발견한 함수/컴포넌트를 plan이 재사용으로 명시했다면:

- 새로 만들지 말 것
- import 후 호출만
- 기존 함수 시그니처가 부족하면 plan을 무시하고 기존 함수를 확장하는 쪽이 보통 더 낫다 (단, evaluate에서 fail 처리될 수 있으므로 차라리 다음 plan 라운드에서 재설계)

### 6. Don't add scaffolding

CLAUDE.md "Doing tasks" 섹션의 규칙을 강제 준수:

- 요구되지 않은 에러 핸들링/검증/폴백 추가 금지
- 미래 확장을 위한 추상화 금지
- 주석은 WHY가 비자명할 때만 (WHAT 설명 금지)
- Backwards-compat 셔임 금지

### 7. Update state.json

`current_stage`를 `"evaluate"`로, `stages_completed`에 `"generate"` 추가.

## Output

- 실제 코드 변경 (commit은 하지 않음 — 사용자 승인 후)
- `${RUN_DIR}/generate.md` — 변경 내역 체크리스트

## Common Pitfalls

- **plan에서 벗어남**: 작업 중 "더 좋은 방법"이 떠올라도 일단 plan을 따른다. 본질적으로 plan이 잘못됐으면 evaluate에서 fail 처리되어 plan 라운드로 돌아간다.
- **commit/push 자동 실행**: 절대 금지. 사용자 명시 요청 시에만.
- **여러 파일 한 번에 Edit하다가 hook 피드백 누락**: 각 파일 수정 후 hook 응답을 확인하고 진행.
- **Read 없이 Edit 시도**: Edit 도구는 같은 세션 내 Read 선행 필수. 모르면 먼저 Read.
