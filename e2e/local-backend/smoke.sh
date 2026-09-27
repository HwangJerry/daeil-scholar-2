#!/usr/bin/env bash
set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/common.sh"
preflight "$@"
command -v jq >/dev/null
readonly API_URL=http://127.0.0.1:18080

# -q ignores ~/.curlrc; never follow redirects or use a proxy for these requests.
request() {
    curl -q --silent --show-error --fail --noproxy '*' --proto '=http' \
        --connect-timeout 5 --max-time 15 \
        -H 'Content-Type: application/json' -H 'X-App-Platform: ios' \
        -H 'X-App-Version: 1.0.0' -H 'X-App-Build: 100' "$@"
}

request "$API_URL/api/settings/public" | jq -e '
    [.app_update_policy_ios, .app_update_policy_android] |
    all(.[]; fromjson | .forceEnabled == false and .recommendEnabled == false)
' >/dev/null
echo 'PASS settings/public: iOS and Android update policies OFF'

login="$(request -X POST "$API_URL/api/auth/mobile/login" \
    --data '{"usrId":"e2e_member","password":"Synthetic-password-09!"}')"
jq -e '.status == "authenticated" and .session.user.usrId == "e2e_member" and
    .session.user.verification.status == "approved" and (.session.accessToken | length > 0)' \
    <<<"$login" >/dev/null
token="$(jq -er '.session.accessToken' <<<"$login")"
unset login
echo 'PASS mobile login: e2e_member (approved)'

# Pass the token through stdin, keeping it out of curl argv and test output.
printf 'header = "Authorization: Bearer %s"\n' "$token" |
    request --config - "$API_URL/api/auth/me" |
    jq -e '.usrId == "e2e_member" and .usrSeq == 100 and .verification.status == "approved"' >/dev/null
unset token
echo 'PASS auth/me: e2e_member (usrSeq=100)'

request "$API_URL/api/auth/check-phone?phone=01000000002" | jq -e '.available == true' >/dev/null
echo 'PASS check-phone: 01000000002 available=true'
echo 'Smoke: 4/4 passed (no tokens printed)'
