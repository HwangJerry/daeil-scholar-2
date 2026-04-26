# SOP — EasyPay 빌링키 수동 폐기

## 배경

`EasyPayService.RevokeBillingKey`(backend/internal/service/easypay_service.go)는 현재 PG 측 폐기 트랜잭션 코드(`tr_cd`)가 머천트 계정에 대해 검증되지 않은 상태로 **stub** 구현이다. 따라서:

- 사용자가 정기후원 취소를 요청하면 백엔드는 로컬 DB의 `WEO_SUBSCRIPTION` 레코드를 `cancelled`로 표시하고 결제 시도를 중단한다.
- **그러나 EasyPay 머천트 계정에 등록된 빌링키 자체는 살아 있다.** 이론적으로 같은 빌링키로 외부에서 결제 요청이 가능하다 (운영 절차상 그럴 일은 없지만 컴플라이언스 갭).
- 따라서 운영자가 EasyPay 머천트 콘솔에서 직접 빌링키를 폐기해야 한다.

코드 측 audit log는 `RevokeBillingKey`가 sentinel error(`ErrManualRevocationRequired`)를 반환하므로 모든 취소 건이 `billing_key_revoke_skipped`로 기록된다. 운영자는 이 로그를 SOP의 트리거로 사용한다.

## 트리거

다음 중 하나가 발생하면 본 SOP를 실행한다:

1. **PG 감사 로그에 `billing_key_revoke_skipped` 항목이 기록됨** — 일반적인 사용자 측 취소.
2. 어드민이 의심스러운 정기후원을 강제 취소한 경우.
3. 사용자가 채널톡/이메일로 "정기후원 카드/빌링키 완전 삭제"를 요청한 경우.

## 절차

### 1. 폐기 대상 식별 (백엔드)

```sql
-- 최근 24시간 동안 취소된 모든 구독의 billing key 추출
SELECT
  s.SUB_SEQ,
  s.USR_SEQ,
  s.BILLING_KEY,
  s.UPDATED_AT
FROM WEO_SUBSCRIPTION s
WHERE s.STATUS = 'cancelled'
  AND s.UPDATED_AT >= NOW() - INTERVAL 1 DAY
  AND s.BILLING_KEY IS NOT NULL
  AND s.BILLING_KEY != ''
ORDER BY s.UPDATED_AT DESC;
```

대상이 PG audit log의 `billing_key_revoke_skipped` 항목과 일치하는지 교차 검증한다 (audit log 경로: `PG_AUDIT_LOG_PATH` env, 기본 `/var/logs/pg/pg-audit.log`).

### 2. EasyPay 머천트 콘솔 로그인

- 콘솔 URL: (운영자 인계서 참조 — `EASYPAY_PROFILE_MALL_ID=05543499` 정기결제용 가맹점)
- 로그인 정보는 운영팀 1Password Vault `EasyPay/Production`에 보관.

### 3. 빌링키 폐기 실행

머천트 콘솔의 정기결제 → 빌링키 관리 메뉴에서 1단계에서 추출한 `BILLING_KEY` 값을 검색하고 "해지/삭제"를 실행한다.

### 4. 처리 기록

처리 완료 후 다음 SQL로 `billing_key_revoked_manual` audit 항목을 추가한다(아직 자동화 안 됨):

```sql
INSERT INTO WEO_PG_AUDIT (REF_KEY, EVENT, PAYLOAD, CREATED_AT)
VALUES ('<SUB_SEQ>', 'billing_key_revoked_manual',
        JSON_OBJECT('operator', '<운영자ID>', 'console_ref', '<콘솔 처리번호>'),
        NOW());
```

또는 사내 Notion 페이지 `Operations / EasyPay 빌링키 폐기 대장`에 기록한다.

## 주기

- **권장 SLA**: 사용자 취소 후 24시간 이내 폐기.
- **모니터링**: 주 1회(매주 월요일) 운영자가 위 SQL을 실행하여 미처리 누적분 0건 확인.

## 자동화 로드맵 (이 SOP 종료 조건)

EasyPay 측에 정확한 폐기 `tr_cd`(후보: `00301000`, `00501000`)와 요청 페이로드 스펙을 문의·검증한 후, `RevokeBillingKey`에서 `ep_cli`로 실제 호출을 구현한다. 그 시점에:

1. `ErrManualRevocationRequired` 제거 + 본 SOP 폐기.
2. `billing_key_revoke_skipped` → `billing_key_revoke_success` 자동 전환 검증.
3. 본 문서를 `docs/archive/` 로 이동하거나 삭제.

## 참고 코드

- `backend/internal/service/easypay_service.go` — `RevokeBillingKey` stub 정의
- `backend/internal/service/subscription_service.go:130-137` — audit log 호출부
- `backend/internal/service/subscription_service.go:117-120` — 취소 흐름 주석
