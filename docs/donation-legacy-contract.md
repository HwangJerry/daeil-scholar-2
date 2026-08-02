# 레거시 기부 거래·엑셀 계약

> 작업 ID: `LEGACY-01`<br>
> 상태: 구현 기준 추출 완료<br>
> 기준일: 2026-07-28<br>
> 범위: `dflh-saf-v1` 읽기 전용 분석과 승인된 MVP 정책의 차이 분류<br>
> 개인정보 처리: 대표 workbook의 개별 이름·연락처·금액은 이 문서에 기록하지 않는다.

## 1. 목적과 적용 우선순위

이 문서는 다음 작업의 입력 계약이다.

- `CONTRACT-01`: 공통 기부 API와 오류 계약
- `DONATION-01`: `WEO_ORDER` 기반 통합 거래 원장
- `EXCEL-01`: 관리자 XLS/XLSX 원자적 증분 업서트
- `ADMIN-03`: 통합 기부 거래·업로드 관리자 UI
- `TEST-02`: 이관·업로드·정합성 리허설

충돌 시 적용 우선순위는 다음과 같다.

1. 사용자가 승인한 결정론적 Seed와 이후 확정 결정
2. 이 문서의 `승인 Seed로 변경` 항목
3. 레거시 `TPL_donation.htm`과 관련 구현
4. 현재 v2 구현

레거시에서 발견한 보안 결함·수동 절차·비원자적 동작은 호환 대상으로 보지 않는다.

## 2. 조사 근거

### 2.1 레거시 목록·수정

| 근거 | 확인 내용 |
|---|---|
| `dflh-saf-v1/dadms/tpl/DONATION/TPL_donation.htm:42-89` | 검색 폼, 기부 방식·상태 필터, 목록 열 |
| `dflh-saf-v1/dadms/DONATION/donation.df:22-39` | 검색 조건과 `WEO_ORDER`/`WEO_MEMBER` 조인 |
| `dflh-saf-v1/dadms/DONATION/donation.df:67-88` | 목록 표시 필드와 수정 dialog 진입 |
| `dflh-saf-v1/dadms/_dialog/vDonation.df:25-47` | 거래 상세 조회와 수정 폼 값 |
| `dflh-saf-v1/dadms/tpl/dialog/TPL_vDonation.htm:17-27` | 기부 방식·금액·진행 상태 입력 |
| `dflh-saf-v1/dadms/_module/mDonationCheck.df:30-40` | 수정 입력 정규화와 금액 필수 검사 |
| `dflh-saf-v1/dadms/_module/mDonationCheck.df:78-95` | `WEO_ORDER` 수정 필드와 마지막 수정자 정보 |
| `dflh-saf-v1/dadms/_sys/sys_config.php:156-166` | 결제 방식 코드 |
| `dflh-saf-v1/dadms/_sys/sys_config.php:197-200` | 상태와 기부 방식 코드의 기준 매핑 |

### 2.2 레거시 엑셀

| 근거 | 확인 내용 |
|---|---|
| `dflh-saf-v1/dadms/donation_excel/excel.html:3-40` | 파일 선택 후 즉시 처리 endpoint 제출 |
| `dflh-saf-v1/dadms/donation_excel/excel_proc.php:8-46` | 업로드 저장, 확장자 기반 reader, active sheet만 배열화 |
| `dflh-saf-v1/dadms/donation_excel/excel_proc.php:50-110` | B~F 열 매핑과 기존 회원 검색·신규 분류 |
| `dflh-saf-v1/dadms/donation_excel/excel_proc.php:112-178` | 사람이 결과를 확인한 후 주석을 풀고 재실행하는 수동 반영 코드 |
| `dflh-saf-v1/html/upload/temp_csv/202507/20250702115815_EPYD4QH9AO.xlsx` | 2019-12~2025-06 월별 원본 workbook 구조 |

### 2.3 현재 v2

| 근거 | 확인 내용 |
|---|---|
| `backend/internal/repository/donation_repo.go:83-107` | `O_PAYMENT='Y'` 기준 합계·기부자 수 집계 |
| `backend/internal/repository/admin_donation_repo.go:41-84` | 거래 목록과 비활성 상태의 수정 구현 |
| `backend/internal/handler/admin_donation_handler.go:66-100` | 거래 목록·수정 handler |
| `backend/cmd/server/routes.go:221-226` | Happy Nanum 전환 사유로 거래 CRUD route가 현재 비활성 |
| `admin/src/components/donation/DonationOrdersSection.tsx:22-119` | 비활성 API를 호출하는 관리자 거래 화면 |

## 3. 레거시 목록·검색 계약

### 3.1 검색

| 입력 | 레거시 동작 | MVP 분류 | MVP 계약 |
|---|---|---|---|
| `schKeyword` | 전화번호·이름·이메일 부분 검색. 화면 placeholder는 이름만 안내 | 승인 Seed로 변경 | 관리자 거래 검색은 이름·정규화 전화번호·거래번호를 명시적으로 분리한다. SQL 문자열 연결은 폐기하고 parameter binding을 사용한다. |
| `schGate` | 기부 방식 필터 | 레거시 유지 | 정기·일시·후원/기타를 source와 별도 차원으로 유지한다. |
| `schStatus` | `O_STATUS` 필터 | 승인 Seed로 변경 | 완료·대기·취소·부분 환불·전액 환불을 명시적 상태로 확장한다. |
| 정렬 | 신청일자 내림차순 | 레거시 유지 | 거래일 내림차순, 동일 일자는 내부 식별자 내림차순으로 안정 정렬한다. |
| 페이지 | DataTables paging 비활성 | 승인 Seed로 변경 | 서버 cursor 또는 page/size를 사용한다. 관리자 기본 20, 상한 50을 유지한다. |

### 3.2 목록 열

| 레거시 표시 | 원천 | MVP 분류 | MVP 표시 |
|---|---|---|---|
| 신청일자 | `O_REGDATE` | 승인 Seed로 변경 | 실제 `donationDate`를 우선 표시하고 등록 시각을 별도 표시한다. |
| 기부자 | `WEO_MEMBER.USR_NAME` | 레거시 유지(관리자 전용) | 관리자만 조회한다. 공개 웹에는 절대 노출하지 않는다. |
| 학과정보 | `USR_DEPT + USR_FN` | 레거시 유지(관리자 전용) | 기수와 학과를 별도 열·필터로 유지한다. |
| 기부방식 | `O_GATE` | 레거시 유지 | 정기·일시·후원/기타를 표시한다. |
| 기부금액 | `O_PRICE` | 승인 Seed로 변경 | 총액·환불액·최종 실수령액을 구분한다. 공개 합계는 최종 실수령액만 사용한다. |
| 결제방식 | `O_PAY_TYPE` | 승인 Seed로 변경 | 해피나눔·계좌이체·기타 source와 카드·은행·가상계좌 등 payment method를 분리한다. |
| 신청상태 | `O_STATUS` | 승인 Seed로 변경 | 거래 lifecycle 상태와 정산 완료 여부를 혼용하지 않는다. |

## 4. 레거시 코드 값

### 4.1 기부 방식 `O_GATE`

| 값 | canonical 의미 | 근거 | MVP 분류 |
|---|---|---|---|
| `P` | 정기기부 | `sys_config.php`와 목록 렌더링 | 레거시 유지 |
| `S` | 즉시기부 | `sys_config.php`와 목록 렌더링 | 레거시 유지 |
| `F` | 후원기부 | `sys_config.php` | 레거시 유지, 관리자 `기타` source와 혼동하지 않음 |

> `TPL_vDonation.htm`은 `S=정기기부`, `P=즉시기부`로 라벨이 뒤바뀌어 있다. 목록과 공통 상수의 `P=정기`, `S=즉시`를 canonical로 사용하고 dialog의 역전은 보존하지 않는다.

### 4.2 레거시 진행 상태 `O_STATUS`

| 값 | 레거시 의미 | MVP 분류 | MVP 매핑 |
|---|---|---|---|
| `Y` | 결제완료 | 레거시 유지 | `completed` 후보. 최종 실수령액이 0보다 큰 완료 거래만 공개 합계에 포함한다. |
| `I` | 결제대기 | 레거시 유지 | `pending`; 공개 합계 제외 |
| `N` | 삭제 | 승인 Seed로 변경 | 신규 원장에서는 물리 삭제 대신 `cancelled` 또는 `fully_refunded`를 원인에 따라 구분한다. 공개 합계 제외 |

### 4.3 결제 완료 플래그 `O_PAYMENT`

`O_STATUS`와 별도로 존재한다. 현재 v2 집계와 관리자 UI는 `O_PAYMENT='Y'`를 완료 판단에 사용하지만, 레거시 관리자 수정 화면은 `O_STATUS`만 바꾸고 `O_PAYMENT`는 바꾸지 않는다. 두 필드가 불일치할 수 있으므로 다음과 같이 변경한다.

- 신규 transaction status를 단일 canonical 상태로 정의한다.
- 호환 기간에는 `O_STATUS`·`O_PAYMENT`를 함께 갱신하고 불일치를 정합성 오류로 보고한다.
- 공개 합계는 단순 `O_PRICE` 합이 아니라 완료 거래의 `grossAmount - refundedAmount`를 합산한다.

### 4.4 결제 방식 `O_PAY_TYPE`

레거시 상수는 `BANK`, `CARD`, `ADMS`, `FREE`, `HP`다. 현재 v2에는 `VBANK` 사용 흔적도 있다. MVP에서는 다음을 구분한다.

- `source`: `happy_nanum`, `bank_transfer`, `other`
- `paymentMethod`: source가 제공하는 카드·은행·가상계좌·휴대폰·관리자 등록 등의 세부 방식

`ADMS`는 source가 아니라 레거시 관리자 입력 방식이므로 신규 원장의 `other` 또는 실제 근거 source로 이관한다.

## 5. 레거시 수정 계약

레거시 수정 dialog는 기존 `O_SEQ` 거래만 다음 필드로 수정한다.

| 입력 | 저장 필드 | MVP 분류 | MVP 계약 |
|---|---|---|---|
| 기부 방식 | `O_GATE` | 레거시 유지 | 허용 enum 검증 후 수정 |
| 기부 금액 | `O_PRICE`, `O_PAY` 동일 값 | 승인 Seed로 변경 | 총액·환불액·최종 실수령액을 분리하고 음수·역전 금지 |
| 진행 상태 | `O_STATUS` | 승인 Seed로 변경 | 명시적 상태 전이만 허용 |
| 내부 식별자 | `O_SEQ` | 레거시 유지 | 서버 path parameter, 양의 정수와 존재 여부 검증 |
| 마지막 수정 시각 | `EDT_DATE=NOW()` | 레거시 유지 | 유지 |
| 마지막 수정 IP | `EDT_IPADDR` | 레거시 유지 | 유지 |
| 마지막 수정 운영자 | `EDT_OPER` | 레거시 유지 | 인증된 `root`/`operator` 계정에서 서버가 기록 |

별도 변경 전후 감사 이력은 만들지 않는다. 원본 엑셀과 셀 값을 영구 감사 로그로 보존하지 않는다.

## 6. 레거시 엑셀 계약과 결함

### 6.1 업로드 파일 처리

| 레거시 동작 | MVP 분류 | MVP 계약 |
|---|---|---|
| 브라우저가 파일 존재만 확인 | 승인 Seed로 변경 | 확장자·MIME·ZIP 구조·최대 크기·시트 수를 서버에서 검증 |
| 확장자로 XLS/XLSX reader 선택 | 승인 Seed로 변경 | 실제 workbook signature와 reader 결과를 검증 |
| 업로드 파일을 `0744`로 저장 | MVP 제외 | 임시 파일은 최소 권한으로 저장하고 처리 종료 시 삭제 |
| `memory_limit=-1` | MVP 제외 | 제한된 메모리와 행 상한을 적용 |
| active sheet만 처리 | 승인 Seed로 변경 | 승인된 단일 업로드 시트 계약을 사용한다. 월별 67개 legacy sheet를 한 요청에서 암묵 처리하지 않는다. |
| 처리 배열 전체를 화면에 출력 | MVP 제외 | 개인정보를 응답·로그에 덤프하지 않고 행 번호·필드·오류 코드만 반환 |
| 사람이 출력 확인 후 코드 주석과 날짜를 수정해 재실행 | MVP 제외 | 전체 사전 검증 후 자동 단일 transaction 반영 |

### 6.2 레거시 열 매핑

| 열 | 의미 | 레거시 처리 | MVP 분류 |
|---|---|---|---|
| A | 비어 있거나 순번 | 사용하지 않음 | MVP 제외 |
| B | 이름 | 비어 있으면 행을 조용히 건너뜀 | 승인 Seed로 변경: 필수, 오류 시 파일 전체 거부 |
| C | 기수 | 회원 검색과 신규 회원 값 | 승인 Seed로 변경: 거래번호가 없으면 복합키 필수 |
| D | 학과 | 신규 회원 값. 기존 회원 검색에는 사용하지 않음 | 승인 Seed로 변경: 거래번호가 없으면 복합키 필수 |
| E | 연락처 | 하이픈 제거. 비어 있으면 행을 건너뜀 | 승인 Seed로 변경: 허용 형식 두 가지 외 전체 거부, 11자리 숫자 저장 |
| F | 해당 월 금액 | 쉼표 제거. 양수·숫자 검증 없음 | 승인 Seed로 변경: 양의 정수, 환불 필드와 최종 실수령액 검증 |

### 6.3 레거시 회원·거래 반영

- 기존 회원 검색 키: 이름 + 기수 + 연락처 + `USR_STATUS >= 'BBB'`; 학과는 검색하지 않는다.
- 일치 회원은 `USR_SEQ`별 배열에 저장되므로 같은 active sheet에서 동일 회원이 여러 행이면 마지막 행이 앞 행을 덮을 수 있다.
- 미일치 회원은 연락처 또는 생성 sequence를 배열 key로 사용한다.
- 주석을 수동으로 해제하면 신규 `WEO_MEMBER`를 `CCC`로 만들고 `WEO_ORDER`를 삽입한다.
- 거래일과 운영자 번호는 코드에 직접 입력한 고정값이다.
- DB transaction, rollback, idempotency, 중복 거래 검증이 없다.
- 저장 실패는 일부 행이 이미 반영된 뒤 발생할 수 있다.

위 동작은 데이터 근거로만 사용하며 신규 업로드 동작으로 보존하지 않는다.

## 7. 대표 workbook 구조 감사

개별 값은 출력하지 않고 구조만 집계했다.

| 항목 | 결과 |
|---|---:|
| 파일 크기 | 298,187 bytes |
| 시트 수 | 67 |
| 기간 | 2019년 12월~2025년 6월 |
| 데이터 행 | 4,217 |
| 이름 누락 | 7 |
| 기수 누락 | 107 |
| 학과 누락 | 615 |
| 연락처 누락 | 26 |
| 금액 누락 | 0 |
| 허용 하이픈 연락처 | 4,175 |
| 허용 compact 연락처 | 0 |
| 기타 연락처 형식 | 16 |
| 숫자 literal이 아닌 금액 셀 | 24 |
| 수식 셀 | 77 |
| 레거시 5필드 기준 시트 내 중복 | 1 |
| 6열 초과 시트 | 4 |

결론:

1. workbook은 개별 거래 원장이 아니라 **월별 후원 합산표**다.
2. transaction number와 개별 donation date가 없다. 시트 제목의 월은 거래일을 증명하지 않는다.
3. 승인 복합키의 기수·학과·연락처가 누락된 행이 있으므로 현 상태로는 전체 원자적 import를 통과할 수 없다.
4. 수식 셀은 cached value만 신뢰하지 말고 계산 결과가 유효한 숫자인지 parser에서 검증해야 한다.
5. 다중 시트 파일을 한 번에 반영하지 않고, 관리자가 승인 template로 행을 보완·이관한 뒤 업로드해야 한다.

## 8. 승인 MVP 업로드 계약

### 8.1 식별자와 우선순위

1. 거래번호가 있으면 `source + transactionNumber`를 우선 unique key로 사용한다.
2. 거래번호가 없으면 다음 값이 **모두 있어야 한다**.
   - donation date
   - 정규화 연락처
   - 이름
   - 기수
   - 학과
   - 금액
3. 복합키 필드가 하나라도 없으면 자동 반영하지 않고 원본을 수정해 재업로드한다.
4. 해피나눔 자동 연동과 엑셀 값이 충돌하면 해피나눔 값을 우선한다.
5. 같은 복합키가 한 파일에 두 번 나오면 중복 오류로 파일 전체를 거부한다.
6. 같은 사용자가 같은 날 같은 금액을 실제로 여러 번 기부한 경우 transaction number 없이 자동 구분하지 않는다.

### 8.2 파일 단위 원자성

처리 순서는 반드시 다음과 같다.

1. 인증·`root`/`operator` 권한 확인
2. 파일 형식·크기·시트·헤더 확인
3. 모든 행 parse
4. 모든 필드 정규화
5. 모든 행 validation과 파일 내부 중복 검사
6. 오류가 하나라도 있으면 DB write 0건으로 전체 거부
7. 전 행 성공 시 단일 DB transaction 시작
8. source 우선순위와 unique key로 증분 upsert
9. 한 행이라도 저장 실패 시 전체 rollback
10. commit 성공 후 임시 원본 삭제와 집계 cache 무효화

별도 최종 승인 단계는 두지 않는다.

### 8.3 연락처

- 입력 허용: `010-0000-0000`, `01000000000`
- 저장·비교: `01000000000` 형식의 11자리 숫자
- 빈 값, 다른 prefix, 자릿수 오류, 문자 포함은 행 오류이며 파일 전체를 거부한다.

### 8.4 거래 금액과 공개 합계

- `grossAmount >= 0`
- `refundedAmount >= 0`
- `refundedAmount <= grossAmount`
- `netReceivedAmount = grossAmount - refundedAmount`
- 완료 거래만 공개 합계 대상이다.
- 대기 거래, 취소 거래, 전액 환불 거래는 공개 합계에서 제외한다.
- 부분 환불 거래는 최종 실수령액만 합산한다.
- 공개 API는 donor row·이름·인원 배열을 반환하지 않는다.

## 9. 필드 분류표

| 개념/필드 | 분류 | 후속 작업 |
|---|---|---|
| `O_SEQ` 내부 식별자 | 레거시 유지 | `DONATION-01` |
| `USR_SEQ` 회원 연결 | 승인 Seed로 변경 | 탈퇴 시 법정 거래와 회원 개인정보 연결 분리 |
| 기부자 이름·기수·학과·연락처 | 레거시 유지(관리자·법정 최소 범위) | `DONATION-01`, `DATA-01`; 공개 금지 |
| `O_GATE` 정기/즉시/후원 | 레거시 유지 | enum 검증 |
| `O_PAY_TYPE` | 승인 Seed로 변경 | source와 payment method 분리 |
| `O_PRICE` | 승인 Seed로 변경 | gross amount 의미 고정 |
| `O_PAY` | 승인 Seed로 변경 | net received와의 호환 매핑 확정 |
| refund amount | 신규 필요 | 부분·전액 환불 차감 |
| source | 신규 필요 | happy_nanum/bank_transfer/other |
| external transaction number | 신규 필요 | source별 우선 unique key |
| donation date | 신규 필요 | 등록 시각과 분리 |
| `O_STATUS` | 승인 Seed로 변경 | 명시적 lifecycle 상태 |
| `O_PAYMENT` | 승인 Seed로 변경 | canonical 상태와 동기화·불일치 탐지 |
| `O_REGDATE`, `O_PAYDATE` | 레거시 유지 | 등록·결제 시각 |
| `O_MEMO` | 레거시 유지(최소화) | 민감정보 금지, 운영 메모 |
| `REG_OPER`, `REG_DATE` | 레거시 유지 | 최초 등록 운영자·시각 |
| `EDT_OPER`, `EDT_DATE`, `EDT_IPADDR` | 레거시 유지 | 마지막 수정 운영자·시각·IP |
| 변경 전후 별도 감사 이력 | MVP 제외 | 생성하지 않음 |
| 원본 엑셀 영구 보존 | MVP 제외 | commit/rollback 뒤 삭제 |
| 공개 기부자 명단 | MVP 제외 | API·DB projection·화면·테스트 없음 |
| legacy 신규 회원 자동 생성 | MVP 제외 | 기부 import가 회원 계정을 만들지 않음 |
| legacy hard-coded 거래일·운영자 | MVP 제외 | 요청·인증 context와 workbook 값 사용 |
| legacy 부분 반영 | MVP 제외 | 단일 transaction 전체 rollback |

미분류 필드: **0개**.

## 10. 후속 계약 결정

`CONTRACT-01`은 이 문서를 바탕으로 다음을 고정해야 한다.

- 관리자 transaction list/detail/create/update/import endpoint
- source·status·paymentMethod enum
- gross/refund/net amount response
- transaction number와 composite key의 정규화
- validation error의 `row`, `field`, `code`, `message` 구조
- 단일 transaction rollback 오류 응답
- 공개 summary에서 `displayAmount`만 노출하는 계약
- 법정 보존 종료와 회원 탈퇴 시 연결 해제 계약

`DB-01`은 MariaDB 10.1에서 JSON column, CTE, window function 없이 위 필드를 저장해야 한다.
