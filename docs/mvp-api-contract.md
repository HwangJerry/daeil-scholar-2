# DFLH SAF v2 MVP 공통 API 계약

> - 작업 ID: `CONTRACT-01`
> - 계약 버전: `mvp-v1`
> - 기준일: 2026-07-28
> - 소비자: Go backend, 관리자 React SPA, Android, iOS
> - 선행 문서: `docs/donation-legacy-contract.md`, 승인 Seed, 구현 계획 `CONTRACT-01`

## 1. 계약 원칙

1. Android와 iOS는 `/api` 아래 같은 endpoint와 JSON wire format을 사용한다.
2. 기존 endpoint는 가능한 한 additive하게 확장하고, 승인 계약과 충돌하는 호환 필드는 release 전에 제거한다.
3. 인증 성공 여부와 동문 인증 상태를 별도 필드로 다룬다.
4. 유효한 계정의 동문 인증 미제출·승인 대기·반려·재승인 대기 사용자에게는 제한 세션을 발급한다.
5. 동문 검색·상세·채팅·차단·push는 서버의 승인 동문 middleware가 강제한다.
6. `root`·`operator` 권한은 회원 상태와 분리하며 backend가 최종 인가한다.
7. 모든 시간은 별도 표기가 없으면 RFC 3339 UTC 문자열이다. token expiry는 기존 호환을 위해 Unix seconds를 유지한다.
8. 식별자는 양의 정수이며 상대 회원 식별자 이름은 모든 신규 계약에서 `userSeq`로 통일한다.
9. 개인정보, access/refresh token, provider credential, device token을 오류·일반 로그에 기록하지 않는다.
10. 공개 웹 API는 동문 이름이나 기부자 행을 반환하지 않는다.

## 2. 공통 HTTP 규칙

### 2.1 요청

- `Content-Type: application/json; charset=utf-8`
- 모바일 인증: `Authorization: Bearer <access-token>`
- `POST`·`PUT`의 알 수 없는 필드는 전환 기간에는 무시할 수 있으나, 필수 필드 누락은 오류다.
- 일반 API body limit는 기존 2 MiB를 유지한다.
- 관리자 Excel import만 인증·권한 확인 뒤 별도 import body limit를 적용한다.

### 2.2 성공 응답

- 조회·생성·변경: JSON body와 `200` 또는 `201`
- body 없는 멱등 삭제: `204`
- 소셜 계정 연결 필요: `202`
- 차단 상태와 관계없는 메시지 접수: 항상 동일한 `200` body

### 2.3 오류 envelope

```json
{
  "code": "ALUMNI_APPROVAL_REQUIRED",
  "message": "동문 인증 승인 후 이용할 수 있습니다.",
  "details": {
    "verificationStatus": "pending"
  }
}
```

- `code`: 클라이언트 분기용 고정 ASCII 대문자 snake case
- `message`: 사용자 표시 가능한 한국어 기본 문구. 분기 기준으로 사용하지 않는다.
- `details`: optional object. field/row/status 등 구조화 정보만 포함한다.
- stack trace, SQL, 토큰, provider 응답 원문은 포함하지 않는다.

### 2.4 공통 HTTP status

| HTTP | 사용 |
|---:|---|
| `400` | JSON 파싱, 필수값, 형식 오류 |
| `401` | 잘못된 자격증명, 만료·무효 token, refresh replay |
| `403` | 정지·탈퇴 계정, 동문 미승인, 역할 부족 |
| `404` | 접근 가능한 범위에서 resource 없음 |
| `409` | 상태 전이 충돌, unique 충돌, 마지막 root 제거, 지원하지 않는 계정 병합 |
| `413` | body limit 초과 |
| `422` | Excel 전체 validation 실패 등 구조화 domain validation |
| `429` | login·provider rate limit |
| `500` | 내부 처리 실패. 세부 원인 비노출 |
| `503` | provider revoke·push provider 등 외부 서비스 일시 장애 |

## 3. 상태와 enum

### 3.1 인증 결과 `AuthResult.status`

| 값 | 의미 | session |
|---|---|---|
| `authenticated` | 기존 계정 credential 검증 성공 | 필수. 동문 상태와 무관 |
| `linkRequired` | 검증된 provider가 아직 회원 계정과 연결되지 않음 | 없음 |

기존 top-level `pending`, `rejected` 응답은 전환 호환만 허용한다. canonical 응답은 `authenticated` session 내부의 `verification.status`로 분기한다.

### 3.2 동문 인증 `verification.status`

| 값 | 의미 | 허용 범위 |
|---|---|---|
| `unsubmitted` | 동문 인증 신청 미제출 | 본인 상태·최초 신청, 로그아웃, 탈퇴 |
| `pending` | 최초 심사 대기 | 본인 상태·신청 조회, 수정 제출, 로그아웃, 탈퇴 |
| `rejected` | 사유 있는 반려 | 위 범위 + 반려 사유 조회·재신청 |
| `approved` | 승인 | 모바일 MVP 회원 기능 |
| `reapproval_pending` | 승인 후 학적 필드 변경 심사 | pending과 동일한 제한 범위 |

### 3.3 관리자 역할 `adminRole`

| 값 | 권한 |
|---|---|
| `root` | 전체 관리자 read/write, 운영자 역할 관리 |
| `operator` | 운영자 역할 관리 외 관리자 read/write |
| `null` | 관리자 API 전부 거부 |

### 3.4 기부 원장 enum

- `source`: `happy_nanum`, `bank_transfer`, `other`
- `status`: `scheduled`, `pending`, `completed`, `partially_refunded`, `cancelled`, `fully_refunded`
- `paymentMethod`: provider 원문을 그대로 노출하지 않고 canonical 문자열 사용. 최소 `card`, `bank`, `virtual_bank`, `mobile`, `admin`, `other`.

## 4. 인증 계약

### 4.1 공통 session

```json
{
  "user": {
    "usrSeq": 101,
    "usrId": "legacy-compatible-id",
    "usrName": "예시 동문",
    "email": "member@example.com",
    "adminRole": null,
    "verification": {
      "status": "approved",
      "graduationYear": 2003,
      "cohort": "18",
      "department": "영어",
      "rejectionReason": null,
      "submittedAt": "2026-07-27T01:00:00Z",
      "reviewedAt": "2026-07-27T02:00:00Z"
    }
  },
  "accessToken": "fixture-access-token",
  "refreshToken": "fixture-refresh-token",
  "accessIssuedAt": 1785114000,
  "accessExpiresAt": 1785117600,
  "refreshExpiresAt": 1787706000,
  "sid": "fixture-session-id",
  "jti": "fixture-refresh-jti"
}
```

- `usrId`는 PHP 공존 호환 필드이며 이메일 login identifier가 아니다.
- `graduationYear`, `cohort`는 서로 다른 개념이다.
- `unsubmitted`에서는 `graduationYear`, `cohort`, `department`, `submittedAt`, `reviewedAt`이 `null`이다.
- `rejectionReason`은 `rejected`일 때만 non-null이다.
- 제한 세션도 token shape는 동일하며 서버에서 현재 DB 상태를 재확인한다.

### 4.2 공통 login result

```json
{
  "status": "authenticated",
  "session": { "user": {}, "accessToken": "...", "refreshToken": "..." }
}
```

모든 이메일·Kakao·Apple login은 같은 top-level envelope를 반환한다.

### 4.3 이메일/비밀번호

`POST /api/auth/mobile/login`

```json
{
  "email": "member@example.com",
  "password": "fixture-password"
}
```

- Android와 iOS 모두 `email`을 보낸다.
- 전환 기간 backend는 legacy `usrId`를 optional alias로 수용할 수 있다.
- release client는 `usrId`를 보내지 않는다.
- 유효 credential + pending/rejected/reapproval 계정은 `200 authenticated` 제한 session이다.
- 잘못된 credential은 `401 INVALID_CREDENTIALS`다.
- 정지·탈퇴는 각각 `403 ACCOUNT_SUSPENDED`, `403 ACCOUNT_WITHDRAWN`이다.

### 4.4 Kakao

`POST /api/auth/kakao/mobile`

SDK access token:

```json
{
  "grantType": "access_token",
  "accessToken": "fixture-kakao-token"
}
```

allowlist redirect authorization code:

```json
{
  "grantType": "authorization_code",
  "code": "fixture-code",
  "redirectUri": "dflhsaf://oauth/kakao"
}
```

응답은 공통 login result 또는 `202 linkRequired`다.

### 4.5 Apple

`POST /api/auth/apple/challenge`

```json
{
  "challengeId": "fixture-challenge-id",
  "nonce": "fixture-raw-nonce",
  "expiresAt": 1785114300
}
```

`POST /api/auth/apple/mobile`

```json
{
  "challengeId": "fixture-challenge-id",
  "identityToken": "fixture-identity-token",
  "authorizationCode": "fixture-authorization-code",
  "givenName": "예시",
  "familyName": "동문"
}
```

Apple은 iOS만 노출한다. Android가 이 endpoint를 호출할 필요는 없다.

### 4.6 연결 필요

HTTP `202`:

```json
{
  "status": "linkRequired",
  "linkRequired": {
    "linkToken": "fixture-link-token",
    "provider": "KT",
    "expiresAt": 1785114300,
    "profile": {
      "displayName": "예시 동문",
      "email": "member@example.com"
    }
  }
}
```

- provider email만으로 기존 계정을 자동 병합하지 않는다.
- `linkRequired`는 신규 회원가입에만 사용한다.

```http
POST /api/auth/social/link
Content-Type: application/json
```

Canonical mobile request는 신규 회원 정보를 포함한다.

```json
{
  "token": "fixture-link-token",
  "mode": "new",
  "client": "mobile",
  "name": "예시 동문",
  "phone": "01012345678",
  "email": "member@example.com",
  "fn": "31",
  "fmDept": "영어",
  "usrPhonePublic": "N",
  "usrEmailPublic": "N"
}
```

- backend는 link token의 검증된 social identity와 함께 새 회원 row를 생성한다.
- 성공 응답은 HTTP `200`의 canonical `authenticated` 결과이며 4.3의 nested
  `session` envelope를 그대로 사용한다.
- 만료·잘못된 token은 `400 INVALID_TOKEN`, 이미 처리 중인 token은
  `409 TOKEN_IN_PROGRESS`, 이미 사용된 token은 `409 TOKEN_ALREADY_USED`다.

### 4.7 refresh/logout

- `POST /api/auth/refresh` request: `{ "refreshToken": "..." }`
- 성공 response: 공통 session object
- replay: `401 REFRESH_REPLAY_DETECTED`, 같은 `sid` family revoke
- 무효 token: `401 INVALID_REFRESH_TOKEN`
- `POST /api/auth/logout`: 현재 session revoke, `204`
- `POST /api/auth/logout/all`: 모든 session·device association revoke, `204`

## 5. 동문 인증 계약

### 5.1 본인 상태

`GET /api/alumni/verification`

```json
{
  "status": "rejected",
  "graduationYear": 2003,
  "cohort": "18",
  "department": "영어",
  "rejectionReason": "확인 가능한 학적 정보를 다시 입력해주세요.",
  "submittedAt": "2026-07-27T01:00:00Z",
  "reviewedAt": "2026-07-27T02:00:00Z"
}
```

제한 session에서 접근 가능하다.

### 5.2 최초 신청·수정 재신청

`PUT /api/alumni/verification`

```json
{
  "graduationYear": 2003,
  "cohort": "18",
  "department": "영어"
}
```

- 최초 제출, rejected 재신청을 같은 멱등 endpoint로 처리한다.
- 결과: `200` verification object, status `pending`
- approved 학적 변경은 status `reapproval_pending`으로 바뀐다.
- 기존 승인값은 심사 완료 전까지 audit 비교용으로 보존하되 검색·채팅 접근은 즉시 제한한다.
- 이름·사진·직종·직무 변경은 재승인 대상이 아니다.

### 5.3 관리자 심사

| Method | Endpoint | 역할 |
|---|---|---|
| GET | `/api/admin/alumni-verifications?status=pending` | root/operator |
| GET | `/api/admin/alumni-verifications/{userSeq}` | root/operator |
| POST | `/api/admin/alumni-verifications/{userSeq}/approve` | root/operator |
| POST | `/api/admin/alumni-verifications/{userSeq}/reject` | root/operator |

목록 response:

```json
{
  "items": [
    {
      "userSeq": 42,
      "userName": "예시 동문",
      "status": "pending",
      "graduationYear": 2003,
      "cohort": "18",
      "department": "영어",
      "rejectionReason": null,
      "submittedAt": "2026-07-27T01:00:00Z",
      "reviewedAt": null,
      "updatedAt": "2026-07-27T01:00:00Z"
    }
  ]
}
```

상세 response는 목록의 item object와 같은 shape다. 관리자 client는 응답의 `updatedAt`을 승인·반려 request의 `expectedUpdatedAt`으로 그대로 전송한다.

승인 request:

```json
{
  "expectedUpdatedAt": "2026-07-27T01:00:00Z"
}
```

반려 request:

```json
{
  "reason": "확인 가능한 학적 정보를 다시 입력해주세요.",
  "expectedUpdatedAt": "2026-07-27T01:00:00Z"
}
```

- 승인·반려 성공: body 없는 `204`
- 빈 reason: `400 REJECTION_REASON_REQUIRED`
- stale review: `409 VERIFICATION_STALE`
- 잘못된 상태 전이: `409 VERIFICATION_STATE_CONFLICT`

## 6. 승인 동문 middleware

다음 endpoint 전체에 Auth + AlumniApproved middleware를 적용한다.

- `/api/alumni`, `/api/alumni/filters`, `/api/alumni/{userSeq}`, `/api/alumni/widget`
- `/api/messages/*`, `/api/badges`
- `/api/blocks/*`
- `/api/push/*`

미승인 response:

```json
{
  "code": "ALUMNI_APPROVAL_REQUIRED",
  "message": "동문 인증 승인 후 이용할 수 있습니다.",
  "details": { "verificationStatus": "reapproval_pending" }
}
```

비인증 `/api/alumni/widget`은 더 이상 공개 이름을 반환하지 않는다. 같은 path를 승인 middleware 아래로 옮겨 기존 승인 client만 호환한다.

## 7. 동문 검색·상세 계약

### 7.1 검색

`GET /api/alumni`

query:

- `name`
- `graduationYear`
- `cohort`
- `department`
- `jobCategory`
- `jobRole`
- `page` 기본 1
- `size` 기본 20, 최대 50

```json
{
  "items": [
    {
      "userSeq": 202,
      "name": "예시 동문",
      "photoUrl": "/files/profile/example.jpg",
      "cohort": "18",
      "department": "영어",
      "jobCategory": "교육",
      "jobRole": "교사"
    }
  ],
  "page": 1,
  "size": 20,
  "totalCount": 1,
  "totalPages": 1
}
```

검색 결과의 허용 필드는 이름·사진·기수·학과·직종·직무와 `userSeq`뿐이다. 연락처·이메일·회사 상세·명함·주석은 반환하지 않는다.

### 7.2 상세

`GET /api/alumni/{userSeq}`

MVP 상세 response의 허용 필드는 검색 결과 필드, 조건부 `phone`·`email`, `blockState.blockedByMe`로 고정한다. 자기소개·회사 상세·명함·태그·관리자 주석 등은 별도 계약 개정 전까지 포함하지 않는다.

```json
{
  "userSeq": 202,
  "name": "예시 동문",
  "photoUrl": null,
  "cohort": "18",
  "department": "영어",
  "jobCategory": "교육",
  "jobRole": "교사",
  "phone": "01000000000",
  "email": "member@example.com",
  "blockState": { "blockedByMe": false }
}
```

- `phone`은 기존 전화 공개 설정이 `Y`이고 값이 비어 있지 않을 때만 property를 포함한다.
- `email`은 기존 이메일 공개 설정이 `Y`이고 값이 비어 있지 않을 때만 property를 포함한다.
- 공개하지 않는 `phone`·`email`은 `null`이나 빈 문자열로 보내지 않고 property 자체를 생략한다. 두 필드의 공개 여부는 서로 독립적이다.
- 검색 목록 response에는 공개 설정과 관계없이 `phone`·`email`을 포함하지 않는다.
- 상대가 나를 차단했는지는 반환하지 않는다.

### 7.3 검색 filter 선택지

`GET /api/alumni/filters`

Auth + AlumniApproved middleware를 적용하며 body 없는 `GET` 요청에 다음 `200` response를 반환한다.

```json
{
  "graduationYears": [2004, 2003],
  "cohorts": ["18", "19"],
  "departments": ["영어", "독일어"],
  "jobCategories": [
    { "seq": 3, "name": "교육" }
  ],
  "jobRoles": ["교사", "연구원"]
}
```

- 모든 top-level 배열은 항상 존재하며 값이 없으면 `[]`를 반환한다. `null`은 반환하지 않는다.
- `graduationYears`, `cohorts`, `departments`는 `ALUMNI_VERIFICATION.status = approved`인 현재 학적의 중복 없는 값만 반환한다.
- `graduationYears`는 정수 내림차순이다.
- `cohorts`는 문자열로 반환하며 숫자로 해석 가능한 값은 숫자 오름차순, 그 밖의 값은 뒤에서 문자열 오름차순으로 정렬한다.
- `departments`와 `jobRoles`는 비어 있지 않은 문자열만 오름차순으로 반환한다.
- `jobRoles`는 승인 동문의 현재 비학적 프로필 직무에서 중복을 제거한다.
- `jobCategories`는 `ALUMNI_JOB_CATEGORY.OPEN_YN = 'Y'`인 canonical 직종 taxonomy이며 정의된 표시 순서를 유지한다. 각 항목은 양의 정수 `seq`와 비어 있지 않은 `name`을 가진다.
- 검색 query의 `jobCategory` 값은 `jobCategories[].seq`인 양의 정수이며 검색 response의 `jobCategory`는 해당 표시명 문자열이다.
- filter response에는 회원 수, 미승인 회원에서만 유래한 학적·직무 값 또는 개인정보를 포함하지 않는다.

## 8. 채팅 계약

### 8.1 범위

- 승인 동문 간 1:1 텍스트
- 최대 1,000 Unicode code points
- 그룹·이미지·파일·입력중·신고 없음
- 일반 메시지는 임의 만료하지 않음
- 탈퇴 시 작성자 연결만 제거하고 표시명 `탈퇴한 회원`

### 8.2 전송

`POST /api/messages`

```json
{
  "userSeq": 202,
  "clientMessageId": "018f1f1a-7c65-7b65-b845-123456789abc",
  "content": "안녕하세요."
}
```

호환 기간에는 `recvrSeq`를 `userSeq` alias로 수용할 수 있다.

정상 또는 상대가 발신자를 차단한 경우 모두 동일한 HTTP `200`:

```json
{
  "messageId": 9001,
  "clientMessageId": "018f1f1a-7c65-7b65-b845-123456789abc",
  "status": "accepted",
  "createdAt": "2026-07-28T01:00:00Z"
}
```

- `clientMessageId`는 발신자별 idempotency key다.
- 차단 여부를 암시하는 status·오류·지연 차이를 만들지 않는다.
- 차단 메시지는 발신자 관점 기록만 유지하고 수신 조회·SSE·push 대상에서 제외한다.

### 8.3 대화 목록

`GET /api/messages/conversations?cursor=<opaque>&size=20`

```json
{
  "items": [
    {
      "userSeq": 202,
      "name": "예시 동문",
      "lastMessage": "안녕하세요.",
      "lastMessageAt": "2026-07-28T01:00:00Z",
      "unreadCount": 0,
      "blockedByMe": false
    }
  ],
  "nextCursor": null,
  "hasMore": false
}
```

### 8.4 대화 메시지

`GET /api/messages/conversations/{userSeq}?before=<opaque>&size=30`

```json
{
  "items": [
    {
      "messageId": 9001,
      "clientMessageId": "018f1f1a-7c65-7b65-b845-123456789abc",
      "sender": { "userSeq": 101, "name": "예시 동문" },
      "recipientUserSeq": 202,
      "content": "안녕하세요.",
      "read": true,
      "createdAt": "2026-07-28T01:00:00Z",
      "readAt": "2026-07-28T01:01:00Z"
    }
  ],
  "nextCursor": null,
  "hasMore": false
}
```

최신 page를 먼저 반환하고 `before`로 과거를 읽는다. 안정 cursor는 `(createdAt, messageId)`를 encode한다.

### 8.5 읽음

- `PUT /api/messages/conversations/{userSeq}/read`
- request: `{ "throughMessageId": 9001 }`
- 성공: `204`

### 8.6 SSE

`GET /api/messages/stream`

- `Accept: text/event-stream`
- 재연결: `Last-Event-ID` header
- 각 event에 SSE `id:`와 JSON `eventId`를 함께 제공
- 유실·만료 event는 REST 재동기화

```text
id: 12001
event: message.created
data: {"eventId":12001,"messageId":9001,"conversationUserSeq":202,"sender":{"userSeq":202,"name":"예시 동문"},"preview":"안녕하세요.","createdAt":"2026-07-28T01:00:00Z"}
```

필수 event:

- `message.created`
- `message.read`
- `conversation.updated`

기존 `message.new`, `message.sent`, `message.read`의 불완전 payload는 canonical event로 수렴한다.

## 9. 단방향 차단 계약

| Method | Endpoint | 의미 |
|---|---|---|
| GET | `/api/blocks` | 내가 차단한 회원 목록; 전용 마이페이지 화면은 MVP 제외 |
| GET | `/api/blocks/{userSeq}` | 동문 상세·대화 메뉴 상태 동기화 |
| PUT | `/api/blocks/{userSeq}` | 멱등 차단 |
| DELETE | `/api/blocks/{userSeq}` | 멱등 차단 해제 |

상태 response:

```json
{
  "userSeq": 202,
  "blockedByMe": true,
  "updatedAt": "2026-07-28T01:00:00Z"
}
```

`GET /api/blocks` response:

```json
{
  "items": [
    {
      "userSeq": 202,
      "blockedByMe": true,
      "updatedAt": "2026-07-28T01:00:00Z"
    }
  ]
}
```

- `items`는 항상 존재하며 차단한 회원이 없으면 `[]`다. `null`은 반환하지 않는다.
- 각 item은 위 상태 response와 동일한 `userSeq`, `blockedByMe`, `updatedAt`만 포함하며 `blockedByMe`는 항상 `true`다.
- `updatedAt`은 `blockedByMe=true`일 때 DB의 active block `UPDATED_AT`을 UTC RFC3339로 반환하고, `blockedByMe=false`이면 `null`이다. 동일 PUT 재실행은 기존 timestamp를 바꾸지 않으며 DELETE와 반복 DELETE는 `updatedAt:null`을 반환한다.
- 목록은 `updatedAt` 내림차순, 동률이면 `userSeq` 내림차순이다.
- 목록에는 이름·사진·연락처·상대가 나를 차단했는지 여부를 포함하지 않으며 pagination query나 page metadata를 추가하지 않는다.
- `GET /api/blocks/{userSeq}`, `PUT /api/blocks/{userSeq}`, `DELETE /api/blocks/{userSeq}`는 모두 위 단건 상태 response를 사용한다.
- 모든 단건 method에서 숫자가 아니거나 `0` 이하이거나 자기 자신인 `userSeq`는 `400 INVALID_USER_SEQ`다.
- PUT만 target의 `ALUMNI_VERIFICATION.STATUS='approved'`를 확인한다. 존재하지 않거나 미승인인 target은 동문 상세와 같은 `404 INVALID_USER_SEQ`이며 block row를 만들지 않는다.
- GET과 DELETE는 자기 자신이 아닌 양수 target의 존재·승인 여부를 조회하지 않는다. active directional row가 없으면 둘 다 `{ "userSeq": target, "blockedByMe": false, "updatedAt": null }`을 반환하므로 탈퇴한 상대도 존재 여부를 노출하지 않고 멱등 해제할 수 있다.

- 상대가 나를 차단했는지는 어떤 API에도 노출하지 않는다.
- 차단·해제는 동문 상세와 대화 메뉴에서 같은 API를 사용한다.
- 차단 기간 메시지는 차단 해제 후에도 수신자에게 보이지 않는다.
- 차단 기간 메시지는 저장일로부터 1개월 뒤 작은 batch job으로 영구 삭제한다.
- 정리 job은 backend startup 직후 한 번 실행하고 이후 1분마다 실행한다. 각 실행은 `PURGE_AT <= UTC_TIMESTAMP()`, `AM_VISIBLE_RECVR='N'`, `AM_SUPPRESSION_REASON='recipient_blocked'`를 모두 만족하는 행을 `(PURGE_AT, AM_SEQ)` 순으로 최대 100개만 영구 삭제한다. 실패한 실행은 행 상태를 바꾸지 않고 다음 실행에서 재시도하며 일반 메시지는 삭제하지 않는다.

## 10. push 계약

### 10.1 device 등록

`POST /api/push/device/register`

```json
{
  "platform": "ios",
  "deviceToken": "fixture-device-token",
  "locale": "ko_KR",
  "apnsEnvironment": "sandbox",
  "bundleId": "com.daeil.dflhsafv2"
}
```

- `platform`: `android` 또는 `ios`
- `platform`, `deviceToken`, `locale`은 필수다. `deviceToken`은 공백 없는 ASCII 1–512자, `locale`은 1–20자다.
- Android에서 `apnsEnvironment`, `bundleId`는 반드시 생략한다. iOS에서는 두 필드가 필수이고 `apnsEnvironment`는 `sandbox` 또는 `production`, `bundleId`는 공백 없는 1–255자다.
- register request는 위 필드 외 property를 허용하지 않으며 위반 시 `400 INVALID_REQUEST`다.
- `(platform, deviceToken)` upsert이며 동일 token은 현재 인증 계정으로 원자 재연결한다. installation ID가 계약에 없으므로 새 token만으로 알 수 없는 과거 token을 추측 삭제하지 않는다. client가 알고 있는 이전 token은 unregister하고 provider의 invalid-token 응답으로 stale row를 제거한다.

response:

```json
{ "status": "registered" }
```

### 10.2 device 해제

`POST /api/push/device/unregister`

```json
{ "deviceToken": "fixture-device-token" }
```

연결되지 않은 token도 `200 { "status": "unregistered" }`로 멱등 처리한다.

- unregister request는 `deviceToken`만 허용하고 공백 없는 ASCII 1–512자가 아니면 `400 INVALID_REQUEST`다.
- platform field가 없으므로 현재 인증 계정이 소유한 동일 token의 Android/iOS row만 삭제한다. 다른 계정으로 재연결된 token은 이전 계정의 unregister로 삭제되지 않는다.

### 10.3 계정 단위 설정

- `GET /api/push/preferences`
- `PUT /api/push/preferences`

```json
{
  "messageEnabled": true,
  "messagePreviewEnabled": false
}
```

- MVP에서 계정 동기화 대상은 채팅 알림과 메시지 preview다.
- GET에서 저장 row가 없으면 side effect 없이 `messageEnabled:true`, `messagePreviewEnabled:true`를 반환하고 PUT만 현재 계정 row를 upsert한다.
- PUT request는 두 canonical boolean field가 모두 필수다. 기존 iOS `noticeEnabled`는 additive 호환 optional boolean field로 받을 수 있으나 저장·response에는 반영하지 않는다. 그 외 property나 잘못된 type은 `400 INVALID_REQUEST`다.
- GET·PUT response는 `messageEnabled`, `messagePreviewEnabled` 두 필드만 포함한다.
- 모든 기기에서 같은 계정 값을 읽는다.

device와 preferences endpoint는 모두 인증 및 `ALUMNI_VERIFICATION.STATUS='approved'`를 요구한다. 미승인 session은 `403 ALUMNI_APPROVAL_REQUIRED`다.

### 10.4 채팅 알림 payload

```json
{
  "type": "message",
  "eventId": "12001",
  "messageId": "9001",
  "conversationUserSeq": "202",
  "senderUserSeq": "202",
  "senderName": "예시 동문",
  "preview": "안녕하세요.",
  "createdAt": "2026-07-28T01:00:00Z"
}
```

- provider custom data는 Android/iOS 공통 문자열 값으로 encode한다.
- 기본 알림은 발신자 이름과 preview를 포함한다.
- `eventId`는 durable idempotency를 위해 decimal `messageId` 문자열과 동일하다.
- `messagePreviewEnabled=false`이면 preview를 정확히 `새 메시지가 도착했습니다.`로 대체한다.
- `messageEnabled=false` 또는 수신자가 발신자를 차단했으면 provider 호출 자체를 하지 않는다.
- push에는 access token·이메일·연락처를 넣지 않는다.
- accepted visible message는 sender request 경로에서 provider를 직접 호출하지 않고 process-local 비동기 queue에 enqueue한다. recipient 기준 4개 shard, shard당 최대 256건이며 같은 recipient의 순서를 보존한다.
- enqueue는 non-blocking이다. queue overflow 또는 process crash 시 push는 유실될 수 있으나 message accepted 응답과 REST history는 유지한다. 별도 transactional outbox는 MVP에 추가하지 않는다.
- transient provider 오류는 worker에서 최대 2회 재시도한다. graceful shutdown은 server shutdown timeout 범위에서 대기 작업을 drain한다.
- invalid provider token 응답은 해당 `(platform,deviceToken)` row만 폐기한다.
- provider delivery는 `PUSH_ENABLED=false`가 기본이다. `PUSH_ENABLED=true`이면 FCM·APNs credential을 startup에서 모두 검증하고 누락 시 server startup을 실패시킨다.
- FCM HTTP v1은 `FCM_PROJECT_ID`, `FCM_CREDENTIALS_FILE`; APNs token auth는 `APNS_TEAM_ID`, `APNS_KEY_ID`, `APNS_PRIVATE_KEY_FILE`을 사용한다. credential file 내용, token, payload, provider 원문 response는 로그에 기록하지 않는다.

## 11. 공개 기부 요약

`GET /api/donation/summary`

```json
{
  "displayAmount": 123456789,
  "goalAmount": 200000000,
  "donorCount": 342,
  "achievementRate": 61.7283945,
  "snapshotDate": "2026-08-20",
  "tierThresholds": {
    "sprout": 1,
    "sapling": 10000,
    "tree": 50000,
    "blooming": 100000,
    "fruiting": 300000
  }
}
```

- `displayAmount`, `goalAmount`, 등급 임계값은 KRW 정수다.
- 오늘 스냅샷이 있으면 사용하고, 없으면 최신 스냅샷을 사용한다. 스냅샷이 전혀 없을 때만 주문과 활성 설정으로 동일 값을 실시간 계산한다.
- 일반 모드의 `displayAmount`는 스냅샷 합계와 수동 보정액의 합이다. 수동 덮어쓰기 모드에서는 수동 보정액 자체를 사용한다.
- `donorCount`는 스냅샷 기부자 수이며, 수동 덮어쓰기 모드에서는 활성 설정의 수동 기부자 수다.
- `achievementRate`는 목표액이 양수일 때 `displayAmount / goalAmount * 100`, 아니면 `0`이다.
- `snapshotDate`는 계산 기준일의 `YYYY-MM-DD` 문자열이다.
- donor rows, 이름, 익명/마스킹 이름은 반환하지 않는다.

## 12. 관리자 기부 원장·Excel

### 12.1 거래 목록·상세·변경

| Method | Endpoint | 역할 |
|---|---|---|
| GET | `/api/admin/donation/orders` | root/operator |
| GET | `/api/admin/donation/orders/{orderSeq}` | root/operator |
| POST | `/api/admin/donation/orders` | root/operator |
| PUT | `/api/admin/donation/orders/{orderSeq}` | root/operator |

거래 DTO 최소 필드:

```json
{
  "orderSeq": 3001,
  "source": "bank_transfer",
  "transactionNumber": null,
  "donationDate": "2026-07-28",
  "donor": {
    "name": "예시 동문",
    "cohort": "18",
    "department": "영어",
    "phone": "01000000000"
  },
  "grossAmount": 100000,
  "refundedAmount": 20000,
  "netReceivedAmount": 80000,
  "status": "partially_refunded",
  "paymentMethod": "bank",
  "memo": null,
  "lastEditedBy": 7,
  "lastEditedAt": "2026-07-28T01:00:00Z",
  "lastEditedIp": "192.0.2.1"
}
```

`lastEdited*`만 유지하며 별도 before/after 감사 이력은 만들지 않는다.

관리자 목록 query와 response는 다음으로 고정한다.

- `page`: 기본 1
- `size`: 기본 20, 최대 50
- optional filter: `name`, `phone`, `transactionNumber`, `source`, `status`, `donationType`
- response: `{ "items": [...], "total": 100, "page": 1, "size": 20 }`
- 정렬: `donationDate DESC, orderSeq DESC`

POST와 PUT은 같은 closed full-replacement body를 사용한다. PUT은 partial update가 아니다.

```json
{
  "source": "bank_transfer",
  "transactionNumber": null,
  "donationDate": "2026-07-28",
  "donor": {
    "name": "예시 동문",
    "cohort": "18",
    "department": "영어",
    "phone": "01000000000"
  },
  "donationType": "one_time",
  "grossAmount": 100000,
  "refundedAmount": 20000,
  "status": "partially_refunded",
  "paymentMethod": "bank",
  "memo": null
}
```

- `donationType`: `recurring`, `one_time`, `sponsorship`; DB `O_GATE`의 `P`, `S`, `F`에 각각 매핑한다.
- `orderSeq`, `netReceivedAmount`, `lastEditedBy`, `lastEditedAt`, `lastEditedIp`는 client가 보내지 않고 서버가 생성·계산·기록한다.
- `netReceivedAmount = grossAmount - refundedAmount`이며 두 amount는 0 이상이고 refund가 gross를 초과할 수 없다.
- `completed`는 refund 0, `partially_refunded`는 `0 < refund < gross`, `fully_refunded`는 `refund = gross`, `scheduled`·`pending`·`cancelled`는 refund 0이어야 한다.
- legacy 동기화는 scheduled/pending=`O_STATUS I`, `O_PAYMENT N`; completed/partially_refunded=`Y`,`Y`; cancelled/fully_refunded=`N`,`N`이다. `O_PRICE=grossAmount`, `O_PAY=netReceivedAmount`를 유지한다.
- `transactionNumber`가 없으면 donation date·정규화 phone·name·cohort·department·gross amount의 composite key를 서버가 계산한다.
- 성공한 create/update는 공개 summary cache를 무효화한다. 관리자 ledger CRUD는 Happy Nanum provider mutation을 수행하지 않는다.
- malformed·unknown field·enum·amount/status 불일치는 `400 INVALID_REQUEST`, 없는 거래는 `404 DONATION_ORDER_NOT_FOUND`, source/transaction 또는 composite identity 중복은 `409 DONATION_ORDER_CONFLICT`다.

### 12.2 Excel import

`POST /api/admin/donation/import`

- multipart field: `file`
- XLS/XLSX
- import 전용 body limit는 실제 4 MiB 이상 파일을 수용하도록 설정
- 전체 parse·정규화·validation 성공 후 단일 DB transaction

성공:

```json
{
  "status": "imported",
  "totalRows": 100,
  "insertedRows": 80,
  "updatedRows": 20
}
```

validation 실패 HTTP `422`:

```json
{
  "code": "IMPORT_VALIDATION_FAILED",
  "message": "파일의 오류를 수정한 뒤 다시 업로드해주세요.",
  "details": {
    "errors": [
      { "row": 4, "field": "phone", "code": "INVALID_PHONE", "message": "010 휴대전화 형식이 아닙니다." }
    ]
  }
}
```

- validation 실패 DB write 0건
- 저장 중 실패 HTTP `500 IMPORT_ROLLED_BACK`, 전체 rollback
- 거래번호 우선; 없으면 승인된 6필드 복합키 전부 필수
- Happy Nanum 자동 source와 충돌하면 Happy Nanum 값 우선
- 원본 파일명·원본 셀·개별 donor 값은 로그에 기록하지 않음

## 13. 관리자 역할 계약

| 작업 | root | operator |
|---|:---:|:---:|
| 동문 인증 승인·반려 | 허용 | 허용 |
| 기부 거래·Excel | 허용 | 허용 |
| 공지·소개 운영 | 허용 | 허용 |
| 운영자 목록·역할 부여·회수 | 허용 | 거부 |

운영자 관리 endpoint:

- `GET /api/admin/operators`
- `PUT /api/admin/operators/{userSeq}` request `{ "role": "root" | "operator" }`
- `DELETE /api/admin/operators/{userSeq}`

오류:

- operator 접근: `403 ADMIN_ROLE_REQUIRED`
- 마지막 root 제거: `409 LAST_ROOT_REQUIRED`
- stale role change: `409 ADMIN_ROLE_STALE`

## 14. 탈퇴·보존 계약

`DELETE /api/auth/account`

성공 전 단일 transaction에서:

- 회원 개인정보·동문 인증 정보 삭제/비식별화
- social provider ID와 credential 제거 또는 revoke outbox 처리
- mobile refresh session과 push token 제거
- 메시지 작성자 연결 제거, 표시명 `탈퇴한 회원`
- 법정 거래 기록은 목적별 최소 필드와 만료시각만 보존

보존기간:

- 계약·청약철회 5년
- 결제·재화 공급 5년
- 소비자 불만·분쟁 3년
- 표시·광고 6개월

## 15. 오류 코드 목록

### 인증

`INVALID_BODY`, `INVALID_REQUEST`, `INVALID_CREDENTIALS`, `ACCOUNT_SUSPENDED`, `ACCOUNT_WITHDRAWN`, `INVALID_REFRESH_TOKEN`, `REFRESH_REPLAY_DETECTED`, `KAKAO_VERIFICATION_FAILED`, `APPLE_VERIFICATION_FAILED`, `LINK_REQUIRED`, `INVALID_TOKEN`, `TOKEN_ALREADY_USED`, `TOKEN_IN_PROGRESS`

### 동문·권한

`UNAUTHORIZED`, `ALUMNI_APPROVAL_REQUIRED`, `VERIFICATION_NOT_FOUND`, `REJECTION_REASON_REQUIRED`, `VERIFICATION_STALE`, `VERIFICATION_STATE_CONFLICT`, `ADMIN_ROLE_REQUIRED`, `LAST_ROOT_REQUIRED`, `ADMIN_ROLE_STALE`

### 채팅·차단·push

`INVALID_USER_SEQ`, `INVALID_MESSAGE`, `MESSAGE_TOO_LONG`, `MESSAGE_NOT_FOUND`, `INVALID_CURSOR`, `INVALID_DEVICE_TOKEN`, `INVALID_PLATFORM`, `PUSH_PROVIDER_UNAVAILABLE`

차단된 발신자에게는 `BLOCKED` 계열 오류를 반환하지 않는다.

### 기부·Excel

`DONATION_NOT_FOUND`, `INVALID_DONATION_STATUS`, `INVALID_AMOUNT`, `INVALID_REFUND_AMOUNT`, `SOURCE_CONFLICT`, `IMPORT_FILE_REQUIRED`, `IMPORT_FILE_TOO_LARGE`, `IMPORT_INVALID_FORMAT`, `IMPORT_INVALID_HEADER`, `IMPORT_VALIDATION_FAILED`, `IMPORT_ROLLED_BACK`, `INVALID_PHONE`, `DUPLICATE_TRANSACTION`, `COMPOSITE_KEY_INCOMPLETE`

## 16. 호환 전환 표

| 현재 구현 | canonical 계약 | release 조건 |
|---|---|---|
| Android email login request가 `usrId` | `email` | Android request와 backend handler 전환 |
| Android login response가 평면형 | `{status, session}` | Android/iOS/Go 동일 fixture 통과 |
| Kakao만 평면 호환 필드를 추가 반환 | 공통 envelope | 모든 provider 동일 구조 |
| pending login을 session 발급 전 403 | 제한 session + verification status | 검색·채팅 등 승인 middleware 추가 |
| social rejected를 HTTP 403 `{status,rejected}`로 반환하여 iOS 표준 오류 decoder와 충돌 | 유효 credential은 `200 authenticated` 제한 session, 상태는 `verification.status=rejected` | provider별 403 branch 제거와 공통 fixture 통과 |
| logout이 `{status}` body를 반환하고 Android가 이를 decode | `204 No Content` | Android/iOS 모두 no-content 호출로 전환 |
| `USR_STATUS`가 로그인·동문·관리자 역할을 혼용 | verification + adminRole 분리 | migration과 middleware 완료 |
| `/api/alumni/widget` 공개 이름 | 승인 middleware 아래 이동 | 비인증 이름 노출 0건 |
| 검색 결과에 phone/email/bizcard 등 포함 | 확정 6필드 + userSeq | privacy contract test 통과 |
| 메시지 `recvrSeq`, `amSeq`, page | `userSeq`, `messageId`, cursor | alias 기간 뒤 모바일 수렴 |
| SSE event ID·message ID 없음 | canonical event + `eventId` | reconnect/gap test 통과 |
| push/block backend route 없음 | 본 문서 endpoint | Android/iOS fixture·integration 통과 |
| push payload가 `event_type`, `event_id`, `args.*` 등 snake case 중심 | `type`, `eventId`, `messageId`, `conversationUserSeq` 공통 payload | Android/iOS parser와 provider adapter 동시 전환 |
| 기부 summary가 `displayAmount`만 반환 | 스냅샷 기준 금액·목표·기부자 수·달성률·기준일·등급 임계값 | 공통 fixture와 플랫폼 decoder 통과 |
| 관리자 거래 route 비활성 | 통합 원장 CRUD/import | RBAC·원자성 test 통과 |

## 17. canonical fixture 목록

canonical fixture는 `docs/contracts/fixtures/`에 둔다.

- `auth-authenticated.json`
- `auth-unsubmitted.json`
- `auth-pending.json`
- `auth-rejected.json`
- `auth-link-required.json`
- `alumni-search.json`
- `message-send.json`
- `message-event.json`
- `push-device.json`
- `push-preferences.json`
- `donation-summary.json`
- `error-approval-required.json`
- `donation-import-validation-error.json`

각 fixture의 byte SHA-256은 Android `app/src/test/resources/contracts/`와 iOS `Tests/DflhSafV2SwiftTests/Fixtures/contracts/` 복제본이 canonical과 같아야 한다.

## 18. 명시적 제외 검증

계약, fixture, API projection에 다음을 만들지 않는다.

- 공개 기부자 명단·donor array·마스킹 이름
- 그룹 채팅
- 이미지·파일 메시지
- 신고
- 이미 분리된 회원 row 병합
- Android Apple login
- 웹 회원 login UI
- 마이페이지 별도 차단 회원 관리 화면
- 변경 전후 감사 이력
- 원본 Excel 영구 저장
