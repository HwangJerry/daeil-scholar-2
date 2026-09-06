// policyContent — Verified operator details and plain-language privacy policy content.
export const PRIVACY_CONTACT = {
  organization: '대일외국어고등학교 장학회',
  officer: '엄은숙',
  email: 'ghkdwp018@gmail.com',
} as const;

export const PRIVACY_UPDATED_AT = '2026-09-06';

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
    information: '회원과 연결된 기부 내역과 금액',
    purpose: '기부 현황과 본인의 기부 내역을 안내합니다. 외부 기부 페이지에서 입력하는 정보는 해당 운영자의 안내도 확인해 주세요.',
  },
  {
    title: '알림과 서비스 운영',
    information: '푸시 토큰, 앱·기기·언어 정보, 알림 설정, 가입·접속 기록, 방문 식별자, IP 주소·브라우저 정보의 해시값, 오류·성능 진단 정보',
    purpose: '알림 전달, 방문 통계, 오류 확인과 서비스 안정성 개선에 사용합니다.',
  },
] as const;

export const PRIVACY_EXTERNAL_SERVICES = [
  { name: 'Apple · 카카오', description: '소셜 로그인 과정에서 계정 인증과 동의한 정보의 전달에 이용합니다. iOS 알림은 Apple 푸시 서비스를 통해 전달합니다.' },
  { name: 'Sentry', description: '앱 오류와 실행·네트워크 성능을 확인하기 위해 진단 정보를 전송합니다.' },
  { name: '가비아', description: '웹사이트와 API 서버 운영을 위한 호스팅 환경을 이용합니다.' },
  { name: 'Google · jsDelivr', description: '웹 글꼴을 불러올 때 이용합니다. 글꼴 요청 과정에서 IP 주소와 브라우저 요청 정보가 해당 서비스에 전달될 수 있습니다. 이메일 문의에는 Google의 Gmail을 이용합니다.' },
  { name: '외부 기부 서비스', description: '기부를 진행하면 외부 기부 페이지로 이동합니다. 결제·기부 신청 정보의 처리 내용은 해당 페이지의 개인정보 안내를 확인해 주세요.' },
] as const;
