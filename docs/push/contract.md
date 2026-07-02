# Push Contract (dflh-saf-v2)

Version: 1.1  
Last Updated: 2026-07-02  
Owner: Backend Team / Mobile Team Lead  

본 문서는 푸시 알림 파이프라인의 계약(Contract)을 고정한다.  
`dflh-saf-v2-swift` 및 `dflh-saf-v2-kotlin` 마이그레이션/운영에서 아래 규칙을 코드/DB/CI에서 강제한다.

## 1) 핵심 원칙

1. `푸시`는 사용자에게 보이는 상태를 직접 저장하는 진실(source-of-truth)이 아니다.  
   서버는 알림 전송 힌트만 발송하고, 최종 일치성은 앱 동기화로 보장한다.
2. 모든 이벤트는 멱등적으로 처리된다.  
3. 모든 페이로드는 최소·안전 규격을 따른다.
4. 실패/재시도/중복은 자동화된 관측 지표로 추적된다.

## 2) 이벤트 타입 정의

| Event Type | 용도 | 기본 우선순위 | collapse_key |
|---|---|---:|---|
| `message.new` | 신규 메시지 알림 | 10 | `message` |
| `message.read` | 메시지 읽음 동기화 힌트 | 20 | `message` |
| `community.post_reply` | 게시글/댓글 응답 | 20 | `community` |
| `admin.notice` | 공지/안내 | 30 | `admin.notice` |
| `auth.security` | 로그인/보안 관련 중요 알림 | 40 | `auth.security` |

> 타입은 신규가 필요할 때만 추가하고, 추가 시 백엔드/클라이언트/테스트/운영 지표에 모두 반영한다.

## 3) Payload Schema (최소 스키마)

```json
{
  "event_type": "message.new",
  "event": "message.new",
  "event_id": "ulid-or-uuid",
  "user_id": "1234",
  "template_key": "push.message.new",
  "template_version": 1,
  "ttl_sec": 86400,
  "collapse_key": "message",
  "args": {},
  "deep_link": "/messages",
  "sent_at": "2026-07-02T00:00:00Z"
}
```

- `user_id`는 수신자 식별용, 노출용 PII는 포함 금지.
- `event`는 Android/iOS 하위 호환 라우팅 alias이며 `event_type`과 같은 값을 사용한다.
- `template_version`은 현재 `1`이다.
- `args`는 라우팅에 필요한 최소 값만 포함하며, DM 본문/발신자 이름/전화번호/주소/개인식별 토큰은 포함하지 않는다.
- 현재 지원 payload:
  - `message.new`: `args.sender_seq`, `args.recvr_seq`, `deep_link=/messages/{senderSeq}`
  - `admin.notice`: `args.post_seq`, `deep_link=/feed/{postSeq}`

## 4) 멱등성 규칙

- `event_id`는 각 푸시 단위의 글로벌 유니크 ID.
- 같은 `event_id`에 대해 최소 24시간 동안 중복 삽입/발송을 허용하지 않는다(서버 unique 제약 또는 outbox unique 적용).
- 클라 역시 최근 `event_id` 캐시(권장 TTL 24h)로 표시 중복 방지.
- collapse_key는 동일 유형 이벤트가 과도하게 쌓일 경우 마지막 상태만 반영하도록 사용.

## 5) TTL 및 유효기간

- `ttl_sec` 기본값: 86400초.
- 단발성 이벤트: 300~3600초.
- 수명 종료된 이벤트는 서버/클라 모두 수신하더라도 처리하지 않는다.

## 6) 유효성 검사 (필수)

다음 필드는 필수.
- `event_type`, `event`, `event_id`, `user_id`, `template_key`, `template_version`, `ttl_sec`, `collapse_key`, `args`, `deep_link`, `sent_at`
필드 유효성:
- `event_type`은 계약된 값만 허용.
- `event`는 `event_type`과 같아야 함.
- `ttl_sec`는 1 ~ 86400 범위.
- `event_id`는 비어있지 않고 중복되지 않아야 함.

## 6-1) Device Registration

인증된 사용자는 아래 endpoint를 사용한다.

- `POST /api/push/device/register`
- `POST /api/push/device/unregister`

등록 요청은 다음 플랫폼만 허용한다.

- `ios`: APNs device token, `apnsEnvironment=sandbox|production`
- `android`: FCM registration token

Device token은 최대 512자까지 저장하며 절대 truncate하지 않는다. 초과 토큰은 `400 INVALID_TOKEN`으로 거절한다.

Android 등록 row는 APNs 전용 routing metadata를 저장하지 않으므로 `APNS_ENVIRONMENT`, `BUNDLE_ID`는 `NULL`이어야 한다. 기존 Android row는 backend migration `033_backfill_android_push_token_metadata_and_length.sql`에서 backfill한다.

Android registration-only smoke는 Firebase credentials 없이 가능하다. Android delivery smoke는 Firebase project, 앱 Firebase config, backend service-account credentials, 실제 FCM token이 모두 있어야 가능하다.

## 7) 에러/재시도 규칙

서버는 FCM/APNs 응답에 따라 다음 동작.

| 응답 클래스 | 동작 |
|---|---|
| `Invalid/NotRegistered` | 토큰 상태를 `REVOKED`로 즉시 전환 |
| `Transient` (5xx/일시 오류) | 지수 backoff + 최대 3회 retry |
| `Success` | 통계 카운트 + event 처리 완료 |
| `Payload rejected` | dead-letter 큐 적재 + 조사 알람 |

## 8) 하위호환성

- 계약 변경 시 하위버전(`template_version`)을 유지하지 못할 경우 해당 이벤트는 삭제하고 fail-safe 동기화 경고를 발생시킨다.
- 템플릿 추가/수정 시 `dflh-saf-v2-swift`에서 fallback을 제공해야 한다.

## 9) 변경 프로세스

1. 제안자: 변경 사유 및 영향 범위 작성
2. Backend + iOS owner 동시 리뷰
3. QA Matrix 항목 갱신
4. PR Gate 통과 후 배포
