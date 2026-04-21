# Stage 5 — evaluate

생성된 코드가 (1) 빌드/린트/타입/테스트를 통과하고, (2) clarify.md의 Acceptance Criteria를 충족하며, (3) CLAUDE.md 규칙을 어기지 않는지 검증한다. 하나라도 fail이면 plan 단계로 자동 복귀.

## Performer

- 메인 에이전트 (Bash로 검증 명령 실행)
- 부합성 검증은 `Explore` 서브에이전트에 위임 (메인 컨텍스트 보호)

## Inputs

- `${RUN_DIR}/clarify.md`
- `${RUN_DIR}/plan.md` (최신 차수)
- `${RUN_DIR}/generate.md` (최신 차수)
- `${RUN_DIR}/state.json` (loop_count 확인)

## Procedure

### 1. Determine version suffix

산출물 파일명: `evaluate.md` 또는 `evaluate.v<N>.md`.

### 2. Identify which stacks changed

`generate.md`의 변경 파일 목록을 보고 어느 영역이 바뀌었는지 판단:

- `backend/**/*.go` 변경 → Go 검증 필요
- `frontend/**/*.{ts,tsx}` 변경 → 프론트 검증 필요
- `admin/**/*.{ts,tsx}` 변경 → 어드민 검증 필요
- `backend/migrations/*.sql` 변경 → MariaDB 10.1 호환성 수동 검토 (CLAUDE.md 표 참조)

### 3. Run quality checks (Bash)

해당 영역만 실행. 결과를 `evaluate.md`에 기록.

**Backend (Go)**:
```bash
cd backend && go vet ./...
cd backend && go build ./cmd/server
# 관련 테스트가 있으면:
cd backend && go test ./internal/<changed-package>/...
```

**Frontend**:
```bash
cd frontend && npm run lint
cd frontend && npx tsc -b --noEmit
```

**Admin**:
```bash
cd admin && npm run lint
cd admin && npx tsc -b --noEmit
```

각 명령의 종료 코드와 핵심 stderr를 `evaluate.md`에 기록:

```markdown
### Code Quality
- [x] `go vet ./...` — pass
- [x] `go build ./cmd/server` — pass
- [ ] `npm run lint` — **fail**
  - `frontend/src/foo.tsx:42` — react-hooks/exhaustive-deps
```

### 4. Acceptance Criteria check (Explore agent)

`Explore` 서브에이전트에 위임:

```
subagent_type: Explore
description: "Verify acceptance criteria for harness run <run-id>"
prompt: |
  다음 두 파일을 비교해 누락/이탈된 Acceptance Criteria가 있는지 검증해줘.

  ## Source of Truth
  Read: .claude/harness/<run-id>/clarify.md
  특히 "Acceptance Criteria" 섹션의 각 체크박스를 항목별로.

  ## What was built
  Read: .claude/harness/<run-id>/generate.md
  그리고 거기 나열된 변경된 파일들을 실제로 Read해서 확인.

  ## Report
  각 Acceptance Criteria마다:
  - [x] 충족 — 어느 파일/라인에서 충족되는지
  - [ ] 미충족 — 무엇이 빠졌는지
  - [?] 부분 충족 — 어디까지 됐고 무엇이 부족한지

  300단어 이내. 추측 금지, 실제 파일 내용 기반으로만.
```

응답을 `evaluate.md`의 "Acceptance Criteria" 섹션에 기록.

### 5. CLAUDE.md compliance check

다음 항목 직접 검토:

- **단일 책임**: 새로 만든 .ts/.tsx 파일이 하나의 도메인/관심사만 다루는가? (mixed concerns 발견 시 fail)
- **validate-code-file.sh hook**: generate 단계에서 인라인 처리되었는가? (남은 [FIX REQUIRED] 있으면 fail)
- **MariaDB 10.1 제약**: SQL/migration 변경이 있다면 CTE/Window function/JSON 컬럼 사용 여부 확인
- **레이어 분리**: Go 변경이 handler→service→repository 분리를 어겼는가? (예: handler에서 직접 SQL)
- **헤더 주석**: 새 .ts/.tsx 파일 첫 줄이 `// <FileName> — <purpose>` 형식인가?
- **YAGNI**: 요구되지 않은 추상화/에러처리/폴백이 추가되었는가?

### 6. Aggregate verdict

`evaluate.md` 마지막에 다음 섹션 추가:

```markdown
## Verdict

- Code Quality: pass | fail
- Acceptance Criteria: pass | fail | partial
- CLAUDE.md Compliance: pass | fail

**Overall: PASS | FAIL**

### Failures (if any)
1. <항목> — <사유> — <어떻게 고쳐야 할지 한 줄>
2. ...

### Next Action
- PASS → 종료, 사용자에게 요약 보고
- FAIL → loop_count++, plan 단계로 복귀 (단, max_loops 도달 시 AskUserQuestion)
```

### 7. Loop or exit

**Pass인 경우**:
- `state.json.current_stage`를 `"done"`으로, `stages_completed`에 `"evaluate"` 추가
- 사용자에게 짧은 완료 메시지 (변경된 파일 수, 통과한 검증 항목, 산출물 디렉토리 경로)

**Fail이고 `loop_count < max_loops`**:
- `state.json.loop_count++`, `current_stage`를 `"plan"`으로
- 즉시 stage 3 (plan) 재실행. 이때 prompt에 "Previous Failure"로 이번 evaluate.md의 Failures 전달

**Fail이고 `loop_count >= max_loops`**:
- AskUserQuestion으로 다음 선택:
  1. 강제 종료 (현재 상태로 마무리, 사용자가 수동 수정)
  2. 추가 1회 루프
  3. 디버깅 모드 — evaluate.md를 사용자와 함께 검토
- 사용자 선택대로 진행

## Output

`${RUN_DIR}/evaluate.md` (또는 `evaluate.v<N>.md`)

## Common Pitfalls

- **검증 명령을 영역과 무관하게 다 돌림**: Go만 바뀌었는데 npm 빌드까지 돌리면 시간 낭비. 변경 영역만.
- **Acceptance Criteria를 사용자 요청 원문과 혼동**: clarify.md에 기록된 체크박스가 SOT다.
- **부합성 검증을 메인이 직접 함**: 메인 컨텍스트가 코드 비교로 가득 찬다. Explore에 위임.
- **루프 무한**: max_loops 체크 빠뜨리면 영원히 루프. 반드시 state.json 확인.
- **Pass인데 다시 plan으로 감**: Pass 시점에서 종료해야 함. 의심나면 evaluate.md의 Verdict를 다시 확인.
