# 소셜 인증 운영 런북

이 문서는 `028_social_auth_security.sql` 적용, Sign in with Apple/Kakao 모바일 로그인, 세션 회전과 계정 탈퇴를 운영하기 위한 절차입니다. 애플리케이션 배포와 DB 마이그레이션은 별도 승인된 변경 창에서 실행합니다.

## 배포 전 확인

1. DB 백업과 복구 연습 시점을 기록합니다.
2. 아래 중복 검사가 0행인지 확인합니다. 결과가 있으면 자동 병합하지 말고 회원별로 정리합니다.

```sql
SELECT USR_SEQ, NMS_GATE, COUNT(*) AS cnt
FROM WEO_MEMBER_SOCIAL
GROUP BY USR_SEQ, NMS_GATE
HAVING COUNT(*) > 1;
```

3. `SOCIAL_CREDENTIAL_ENCRYPTION_KEY`는 32바이트 random key의 standard base64여야 합니다. 키가 없으면 신규 소셜 로그인/연결은 credential 보관 단계에서 실패하도록 되어 있습니다.
4. `JWT_SECRET`가 기본값이 아닌지, `ACCESS_TOKEN_TTL=15m`, `REFRESH_TOKEN_TTL=720h`인지 확인합니다.
5. Kakao REST key/secret, 정확한 redirect allowlist, Apple Team/Key/Client ID, audience, `.p8` key와 server notification URL을 확인합니다.
6. 기존 `WEO_MEMBER_SOCIAL`의 `(NMS_GATE, NMS_ID)` unique key는 유지됩니다. 028은 한 회원이 같은 provider를 중복 연결하지 못하도록 `(USR_SEQ, NMS_GATE)` unique key를 추가합니다.

## 마이그레이션

현재 DB가 027까지 순서대로 적용됐다면 028 적용 후 029를 별도 변경 창에서 검토해 실행합니다.

```bash
mysql --host=<host> --user=<user> --password <database> \
  < backend/migrations/028_social_auth_security.sql

mysql --host=<host> --user=<user> --password <database> \
  < backend/migrations/029_canonical_member_phone.sql
```

신규 환경의 통합 적용에는 `backend/migrations/apply_all.sql`을 사용합니다. 개별 028 파일의 `ALTER TABLE`은 one-time migration입니다. 재실행이 필요한 경우 통합 파일의 idempotent helper를 사용하거나 `information_schema`로 각 column/index 존재 여부를 먼저 확인합니다.

029는 `USR_PHONE`의 하이픈/공백을 제거해 신규 입력과 같은 digits-only 값으로
backfill하고 일반 인덱스 `IDX_WEO_MEMBER_PHONE`을 추가합니다. 애플리케이션은
029 적용 전에도 중앙화된 legacy fallback 비교를 수행하지만, 함수가 적용된
fallback은 인덱스를 사용할 수 없습니다. 따라서 평상시 검색 성능은 canonical
exact match와 029 인덱스로 보장하고, fallback은 마이그레이션 전/누락 행의
호환성에만 사용합니다. 전화번호는 계정 통합 권한이 아니며 후보 탐색과 중복
확인에만 쓰입니다.

### 사후 검증

아래 쿼리는 모든 `found`가 1이어야 합니다.

```sql
SELECT 'NMS_EMAIL' AS chk, COUNT(*) AS found
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND COLUMN_NAME='NMS_EMAIL'
UNION ALL
SELECT 'UK_USR_PROVIDER', COUNT(*)
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER_SOCIAL' AND INDEX_NAME='UK_USR_PROVIDER'
UNION ALL
SELECT 'ALUMNI_MOBILE_REFRESH_TOKEN', COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_MOBILE_REFRESH_TOKEN'
UNION ALL
SELECT 'ALUMNI_APPLE_NONCE_CHALLENGE', COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_APPLE_NONCE_CHALLENGE'
UNION ALL
SELECT 'ALUMNI_APPLE_CODE_REPLAY', COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_APPLE_CODE_REPLAY'
UNION ALL
SELECT 'ALUMNI_SOCIAL_CREDENTIAL', COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_SOCIAL_CREDENTIAL'
UNION ALL
SELECT 'ALUMNI_SOCIAL_REVOCATION_OUTBOX', COUNT(*)
FROM information_schema.TABLES
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ALUMNI_SOCIAL_REVOCATION_OUTBOX';

SELECT 'IDX_WEO_MEMBER_PHONE', COUNT(*)
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='WEO_MEMBER' AND INDEX_NAME='IDX_WEO_MEMBER_PHONE';
```

## 기능 스모크 테스트

- Kakao: SDK 취소와 provider 오류가 구분되는지, 조작된 access token이 거부되는지, linked/pending/new-account 흐름을 각각 확인합니다.
- Apple: 첫 로그인에서 이름이 전달되는 경우와 재로그인에서 이름/이메일이 nil인 경우, Private Relay 이메일, 취소, 잘못된 nonce/audience를 확인합니다.
- 신규 회원가입: 같은 이메일만으로 기존 계정에 연결되지 않고 `mode=new`이 새 회원을 생성하는지 확인합니다.
- 세션: access 만료 후 한 번만 refresh되는지, refresh rotation 후 이전 토큰 재사용이 해당 `sid` 전체를 로그아웃하는지 확인합니다.
- 상태: `CCC`/`ZZZ`만 로그인되고 `BBB`는 pending, `AAA`와 알 수 없는 상태는 거부되는지 확인합니다.
- 로그아웃: 현재 세션 로그아웃과 전체 로그아웃을 서로 다른 기기에서 확인합니다. 오프라인 로그아웃은 기기 Keychain을 정리해야 합니다.
- 탈퇴: 요청 직후 `USR_STATUS=AAA`와 모든 세션 revoke가 반영되는지 확인합니다.

## Revocation 실패 복구

provider API 장애가 발생하면 계정 탈퇴는 로컬 상태와 세션 폐기를 먼저 완료하고 HTTP 202를 반환합니다. 실패 작업은 `ALUMNI_SOCIAL_REVOCATION_OUTBOX`에 `PENDING`으로 기록됩니다.

탈퇴 요청 시 즉시 수행되는 범위는 `USR_STATUS=AAA`, 모든 서비스 세션 폐기, provider revoke 시도입니다. 회원 기본 행, 기부/결제 기록, 감사 목적 기록은 이 API에서 물리 삭제하지 않습니다. 법적·운영 보존기간이 끝난 데이터의 삭제/익명화는 별도 승인된 배치로 수행해야 하며, provider revoke 실패 재처리와도 분리합니다.

현재 outbox는 명시적 운영 복구 상태이며 자동 worker는 포함하지 않습니다. 운영자는 장애 해소 후 다음 순서로 처리합니다.

1. due row를 조회하되 `LAST_ERROR`를 외부 티켓/일반 로그에 복사하지 않습니다.
2. 해당 사용자의 encrypted credential을 애플리케이션과 동일한 vault 경로로 복호화해 provider revoke를 재시도합니다.
3. 성공 시 social link/credential을 삭제하고 outbox를 완료 처리합니다.
4. 실패 시 `ATTEMPT_COUNT`, `NEXT_ATTEMPT_AT`, 짧게 정제한 `LAST_ERROR`, `UPDATED_AT`을 갱신합니다.

```sql
SELECT OUTBOX_ID, USR_SEQ, PROVIDER, ACTION, ATTEMPT_COUNT, NEXT_ATTEMPT_AT
FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
WHERE STATUS='PENDING' AND NEXT_ATTEMPT_AT <= NOW()
ORDER BY NEXT_ATTEMPT_AT, OUTBOX_ID;
```

Apple authorization code는 교환 시도 전에 replay table에 소비 기록됩니다. Apple/network 교환이 실패한 사용자는 이전 code를 재전송하지 말고 앱에서 새 challenge와 새 Apple authorization을 시작해야 합니다.

## 보안/관측 규칙

- access/refresh/provider token, Apple authorization code, raw nonce, private key를 기록하지 않습니다.
- 인증 JSON을 기록해야 하는 별도 도구는 공통 redactor를 통과시킵니다.
- Apple JWKS는 TTL cache되며 모르는 `kid`가 오면 한 번 강제 새로고침합니다.
- credential encryption key rotation은 기존 row 재암호화 절차와 함께 진행해야 합니다. 키만 교체하면 provider revoke가 불가능해집니다.
