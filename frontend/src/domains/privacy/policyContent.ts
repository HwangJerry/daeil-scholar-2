// policyContent — Verified operator details and plain-language privacy policy content.
export const PRIVACY_CONTACT = {
  organization: '대일외국어고등학교 장학회',
  officer: '엄은숙',
  requestHandler: '황제철',
  email: 'ghkdwp018@naver.com',
} as const;

export const PRIVACY_UPDATED_AT = '2026-09-07';

export const PRIVACY_SECTIONS = [
  { id: 'collection', title: '어떤 정보를 이용하나요?' },
  { id: 'sharing', title: '누구에게 공개되나요?' },
  { id: 'retention', title: '얼마나 보관하나요?' },
  { id: 'services', title: '어떤 외부 서비스를 이용하나요?' },
  { id: 'choices', title: '내 정보는 어떻게 관리하나요?' },
  { id: 'cookies', title: '쿠키와 기기 설정은 어떻게 쓰이나요?' },
  { id: 'protection', title: '정보를 어떻게 보호하나요?' },
  { id: 'contact', title: '개인정보 문의는 어디로 하나요?' },
] as const;

export const PRIVACY_COLLECTION = [
  {
    title: '회원가입과 동문 확인',
    information: '아이디, 비밀번호 인증정보, 이름, 전화번호, 이메일, 기수, 학과, 가입·승인 상태',
    purpose: '계정 인증, 동문 자격 확인, 가입 승인과 이용 문의에 사용합니다.',
  },
  {
    title: 'Apple·카카오 로그인',
    information: '소셜 계정 식별자, 제공에 동의한 이름·이메일, 인증 토큰',
    purpose: '선택한 소셜 계정으로 로그인하고 계정 연결을 관리합니다.',
  },
  {
    title: '동문 프로필',
    information: '사진, 소속, 직책, 업종, 소개, 태그, 근무지, 연락처 공개 설정 등 직접 입력한 정보',
    purpose: '동문 검색과 프로필 표시에 사용합니다. 선택 정보는 관련 화면에서 수정할 수 있습니다.',
  },
  {
    title: '쪽지와 신고',
    information: '송수신 계정, 메시지 내용, 전송·읽음 시각, 차단 내역, 신고 사유·설명, 신고 메시지 원문과 운영자 처리 기록',
    purpose: '동문 간 소통, 차단 처리, 부적절한 메시지 검토와 이용자 보호에 사용합니다.',
  },
  {
    title: '기부 내역',
    information: '기부자 이름·기수·학과·전화번호, 기부 금액·일자와 회원 연결 정보',
    purpose: '해피나눔에서 전달받은 기부 내역을 회원과 연결하여 기부 현황과 본인의 기부 내역을 안내합니다. 기부 페이지에서 입력하는 정보는 해피나눔의 안내도 확인해 주세요.',
  },
  {
    title: '알림과 서비스 운영',
    information: '푸시 토큰, 앱·기기·언어 정보, 알림 설정, 가입·접속 기록, 방문 식별자, IP 주소·브라우저 정보의 해시값, 오류·성능 진단 정보',
    purpose: '알림 전달, 방문 통계, 오류 확인과 서비스 안정성 개선에 사용합니다.',
  },
] as const;

export const PRIVACY_EXTERNAL_SERVICES = [
  { name: 'Apple · 카카오', description: '소셜 로그인 과정에서 계정 인증과 동의한 정보의 전달에 이용합니다. iOS 알림은 Apple 푸시 서비스를 통해 전달합니다.' },
  { name: 'Sentry', description: '앱 오류와 충돌 원인을 확인하기 위해 진단 정보를 HTTPS로 전송하며, 데이터는 미국에 저장됩니다. iOS 출시 버전은 회원 식별정보·요청 내용·화면 이동 기록을 제외하고 오류 위치와 앱 버전 등으로 수집을 제한합니다. 이전 버전과 Android에서는 실행·네트워크 성능 진단 정보가 수집될 수 있습니다.' },
  { name: '가비아', description: '웹사이트와 API 서버 운영을 위한 호스팅 환경을 이용합니다.' },
  { name: 'Google · jsDelivr', description: 'Pretendard 웹 글꼴을 jsDelivr에서 불러옵니다. 글꼴 요청 과정에서 IP 주소와 브라우저 요청 정보가 jsDelivr에 전달될 수 있습니다. 일반 앱 이용 문의에는 Google의 Gmail을 이용합니다.' },
  { name: '네이버', description: '개인정보 열람·정정·삭제·처리정지 요청을 네이버 메일로 접수합니다. 요청에 적어 보내신 연락처와 문의 내용이 메일 서비스에서 처리됩니다.' },
  { name: '해피나눔', description: '기부 신청을 위한 외부 서비스를 이용하며, 장학회는 해피나눔에서 기부 내역을 전달받아 현황과 회원별 내역을 관리합니다. 기부금 영수증은 해피나눔에서 디지털 방식으로 발급하며 원본도 해당 시스템에 보관합니다. 결제·기부 신청 정보의 처리 내용은 해피나눔의 개인정보 안내도 확인해 주세요.' },
] as const;
