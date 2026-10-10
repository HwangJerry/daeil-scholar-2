# 인증 QA 후 탈퇴 보완 — 적용·검토 메모

작업 4(AUTH-QA-11), 5(AUTH-QA-10)의 서버/관리자 수정이다. 운영 DB와 계정은 이 작업에서 변경하지 않았다.

## 스키마와 적용 순서

1. 기존 migration 082까지의 실제 운영 적용 상태와 승인 lineage를 확인한다. 기존 migration 계약 테스트 세 개의 불일치가 남아 있다. 이를 승인 절차 없이 기대값 변경으로 덮지 않는다.
2. additive migration `083_add_phone_grant_ownership.sql`, `084_create_erasure_subscription_review.sql`을 검토한다. 후보 manifest에는 이 두 파일의 checksum만 추가했다. 운영 승인 환경값이나 기존 계약의 승인 digest는 바꾸지 않았다. 현재 migration runner의 승인 검사를 우회하지 않는다.
3. worker를 중지하고 승인된 스키마 확장을 적용한 뒤 새 backend와 관리자 웹을 배포한다. schema-before-server 순서를 지킨다. 부분 DDL 실패 시 실제 컬럼/인덱스 상태와 migration journal을 대조한 뒤 복구한다.
4. 합성 QA 회원으로 grant 소비 소유권, 미리보기, 구독 종료 근거 기록, worker 재시도를 확인한 뒤 worker를 재개한다. 원본 root·iOS 계정은 건드리지 않는다.
5. `database_erased`의 기존 요청은 저장된 암호화 context를 이용해 자동 재검증한다. context가 만료되었으면 기존 수동 검토 절차를 사용한다. 증거를 만들어 강제로 완료하지 않는다.

두 migration은 MariaDB 10.1.38에서 검증한다. `CONSUMED_USR_SEQ`와 `CONSUMED_AT`은 nullable이고 과거 SMS 기록의 소유자를 추정해서 backfill하지 않는다. 회원을 지울 때 소유 정보가 먼저 소실되는 FK를 만들지 않았다. 이전 앱의 요청 형식은 바뀌지 않는다.

롤백 시 확장 컬럼/테이블을 즉시 삭제하지 않는다. 이전 서버는 신규 grant 소유권을 기록하지 못하고 새로운 구독 검토 근거를 활용하지 못하므로, 탈퇴 worker를 중지한 채 rollback한다. 미완료 요청과 복구 증거를 보존하고 새 서버 복구 후 재시도한다.

## SMS 완료 대기

- 가입이 commit된 회원의 canonical 전화번호와 grant 전화번호를 동일 transaction에서 확인해 소비와 소유 관계를 함께 기록한다. 전화번호가 다른 grant는 소비하지 않는다. native·social 신규 가입 모두 이 경로를 사용한다.
- 회원 소유가 증명된 소비 기록만 회원 삭제 계획에 포함한다. 소유가 불명확한 과거 SMS 기록은 기존 24시간 정책의 cleanup을 기다린다. 아직 사용할 수 있는 코드/grant는 cleanup에서 보호한다.
- 미리보기의 `completionWaits`는 DB 단계의 `blockers`와 별개다. SMS 건수와 예상 시각만 제공한다. 발급 24시간, 시간당 cleanup, 5분 재시도와 worker polling을 고려한다. cleanup 장애·대기열 지연은 예상 시각을 늦출 수 있다.
- 결과의 `other_identifiers`에 `PHONE_VERIFICATION_RETENTION_PENDING`, `waitCount`, `waitUntil`을 저장하고 관리자 목록에 표시한다. 미확인 개인정보나 OTP/token은 메타데이터로 보내지 않는다.
- 새 회원의 typed identity/phone claim/소비 grant는 현재 회원 소유 관계가 확인되는 행만 이전 탈퇴의 잔류 검사에서 제외한다. revoked·orphan·소유 불명 기록 및 자유 텍스트는 계속 잔류로 판단한다.
- SMS 대기 행, 처리할 행, 정리 보류 참조, 검토 근거의 원본 hash를 승인 digest에 반영한다. 승인 뒤 대상 기록이 추가되거나 바뀌면 재검토한다.

## 구독 참조 검토

관리자 root용 기존 `PUT /api/admin/account-deletions/{id}`에 `subscription_review` 동작을 추가한다. 요청에는 대상 구독, 현재 source fingerprint, 공급자의 최종 실패·취소 상태 및 외부 종료 확인와 개인정보 없는 근거가 필요하다. operator는 인증된 root의 회원 번호로 기록한다.

공급자 조회를 새로 자동화했다고 간주하지 않는다. 담당자가 공급자 관리 화면/확인 기록에서 외부 결제·청구 종료를 검증한 뒤 명시적으로 기록하는 절차다. 조회 실패·미확인은 종료 증거가 아니다. 로컬에 pending으로 남았더라도 청구키·미확정 주문이 없고 공급자의 최종 종료 상태와 근거가 명시된 경우에만 정리할 수 있다. 종료 미확인 pending, 저장된 청구키, 활성 구독, 미확정 결제 또는 연결된 주문의 소유/종료 상태를 확인하지 못한 경우에는 기록 또는 삭제를 허용하지 않는다.

구독의 전체 원본 행, 관련 주문과 PG 기록을 fingerprint로 묶는다. 동일 source의 재시도는 최초 operator·시각·근거를 보존한다. source가 바뀌면 기존 검토를 사용하지 않는다. preview와 worker는 같은 판정을 사용하며 worker가 정식 transaction에서 참조를 지운다. 근거 기록은 삭제 후에도 유지한다. 기부금 법정 보존·영수증·제3자 파일 보호는 기존 절차를 유지한다.

상태를 읽을 때 오래된 repeatable-read snapshot에 의존하지 않도록 관련 원본을 현재 locking read로 검증한다. 관리자/worker lock은 검토와 삭제를 직렬화하고, row/range lock은 결제 상태가 검증과 삭제 사이에 바뀌지 않게 한다.
