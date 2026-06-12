# Branch Protection 설정 샘플 (10가지 고민 기준용)

목표: `push-pr-gate`가 통과되지 않으면 `main` 병합이 불가능하게 만든다.

## 1) 권장 필수 체크(필수)
- `Push PR Gate` (`push-pr-gate` 워크플로우)
- 서비스 핵심 테스트(`backend test` 등 기존 job 이름)
- 보안/CI 기본 체크(기존 lint/sast job 이름)

## 2) GitHub CLI 적용 예시

```bash
# 저장소/브랜치 변수
OWNER=your-org-or-user
REPO=dflh-saf-v2
BRANCH=main

gh api \
  -X PATCH \
  -H "Accept: application/vnd.github+json" \
  "/repos/$OWNER/$REPO/branches/$BRANCH/protection" \
  -f required_status_checks='{"strict":true,"contexts":["Push PR Gate"]}' \
  -f enforce_admins=true \
  -f allow_force_pushes=false \
  -f allow_deletions=false \
  -f required_approving_review_count=1 \
  -f require_code_owner_reviews=false \
  -f require_linear_history=false \
  -f required_conversation_resolution=true \
  -f dismiss_stale_reviews=true
```

## 3) 다중 체크 요구 시 API 바디 예시

```bash
cat > /tmp/protection.json <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "Push PR Gate",
      "ci/test",
      "ci/security"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 1,
    "require_last_push_approval": false,
    "bypass_pull_request_allowances": { "users": [], "teams": [] }
  },
  "restrictions": null
}
JSON

gh api \
  -X PUT \
  -H "Accept: application/vnd.github+json" \
  "/repos/$OWNER/$REPO/branches/$BRANCH/protection" \
  --input /tmp/protection.json
```

## 4) 운영 체크리스트
- PR 템플릿/체크리스트는 필수지만, PR Gate가 최종 게이트.
- PR Gate는 템플릿 누락 또는 링크 미기재 시 fail.
- 필요 시 긴급 경로는 `critical` 라벨 + Tech Lead 1인 승인 규칙으로 별도 분기.

