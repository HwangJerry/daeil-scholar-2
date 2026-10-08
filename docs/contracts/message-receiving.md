# 쪽지 수신과 차단 목록 관리

2026-10-08: 프로필 공개 스위치는 제거한다. 승인된 동문의 기본 프로필은 동문 네트워크에 계속 표시하고, 연락처 공개 설정은 그대로 사용한다.

## 쪽지 수신

- GET /api/profile/message-preferences → {"messageAllowed": true|false}
- PUT /api/profile/message-preferences ← 같은 본문. messageAllowed는 필수 boolean이고, 알 수 없는 필드와 복수 JSON 본문은 거부한다.
- 현재 인증된 계정만 읽고 변경할 수 있다. WEO_MEMBER.USR_MESSAGE_ALLOWED에 저장한다. 기존 계정은 Y가 기본이다.
- DB migration 081을 서버 배포 전에 적용해야 한다. 마이그레이션 승인용 새 manifest SHA-256은 `314a876133ed5e1da99b678bef25eff217cf4445509aeab14ca3a5618fde3ea4`이며 운영 환경의 승인 값은 배포 시 별도로 갱신해야 한다. 알림 설정의 messageEnabled와 독립적인 값이다.
- 동문 검색 카드 및 상세 응답의 messageAllowed=false이면 앱/웹의 쪽지 보내기 동작을 숨긴다.
- GET /api/alumni?messageRecipientsOnly=true는 수신 허용자만 대상으로 검색한다. 필터는 count와 page 쿼리 모두에 적용한다. 일반 동문 검색은 이 조건을 적용하지 않는다.
- 기존 대화 응답에 recipientMessageAllowed를 제공한다. false이면 이전 쪽지는 계속 읽을 수 있지만 입력창 대신 ‘상대방이 쪽지 수신을 꺼두었어요.’를 표시한다.
- 신규 전송은 서비스 검사와 INSERT 시점의 DB 검사 모두 수행한다. 거부는 HTTP 403 / RECIPIENT_RECEIVING_DISABLED다. 메시지를 저장하지 않으며, 알림·실시간 수신 이벤트도 발생시키지 않는다.
- 수신을 끈 계정도 수신을 켠 다른 계정에게 발신할 수 있다. 다시 켜면 이후의 수신을 허용하며 거부된 메시지를 소급 전달하지 않는다.
- 이전에 수락한 clientMessageId 재시도는 원래 결과를 반환한다. 수신 거부 후 중복 메시지를 생성하지 않는다.
- 설정 화면은 로딩 성공 전과 저장 중에 스위치를 비활성화한다. 저장 실패는 마지막 저장 값으로 되돌리고 차분한 안내를 표시한다.
- 상대방 설정은 검색/상세/대화를 다시 불러올 때 갱신된다. 이미 열린 화면의 오래된 상태에서도 실제 전송은 서버가 거부한다.

## 차단한 동문

- 내정보의 ‘차단한 동문’ 메뉴는 기존 GET /api/blocks를 사용한다.
- items는 인증된 사용자가 차단한 방향만 반환한다. userSeq / blockedByMe / updatedAt 외에 name / photoUrl / cohort / department를 목록 표시용으로 제공한다.
- 탈퇴한 상대는 ‘탈퇴한 회원’으로 표시하며 프로필 사진·학적 정보는 제공하지 않는다. 차단 해제는 허용한다.
- 왼쪽 밀기 또는 상단 편집 → 빨간 원형 −로 휴지통을 드러낸다. 휴지통의 접근성 이름은 ‘[이름] 차단 해제’다.
- 휴지통은 DELETE /api/blocks/{userSeq}로 차단을 해제한다. 회원 삭제나 대화 삭제가 아니다.
- 서버 성공 뒤에만 목록에서 제거한다. 실패하면 항목을 유지하며 안내한다. 중복 요청은 저장 중 비활성화한다.
- 차단 당시 숨겨진 쪽지는 차단을 해제해도 다시 노출하지 않는다. 전체 쪽지 수신이 꺼져 있다면 개인 차단을 풀어도 새 쪽지를 받을 수 없다.

## 검증

백엔드 단위 테스트와 MariaDB 10.1.38 통합 테스트, 두 앱의 설정/차단 관리 모델 테스트, iOS UI 및 Android Compose UI 테스트로 확인한다. 운영 계정과 실제 쪽지는 테스트에 사용하지 않는다.
