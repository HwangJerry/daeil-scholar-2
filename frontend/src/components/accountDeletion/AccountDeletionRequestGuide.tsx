// AccountDeletionRequestGuide — How to request account deletion without the app, and what is deleted or kept.
import { Card } from '../ui/Card';
import { PRIVACY_CONTACT } from '../../domains/privacy/policyContent';

const APP_NAME = '대일외고 장학회 앱';
const MAIL_SUBJECT = `[${APP_NAME}] 계정 삭제 요청`;
const MAIL_BODY = [
  '이름:',
  '기수·학과:',
  '로그인 방식(아이디·카카오·Apple):',
  '아이디 또는 가입 이메일:',
  '',
  '비밀번호나 인증번호는 적지 마세요.',
].join('\n');
const MAIL_HREF = `mailto:${PRIVACY_CONTACT.email}?subject=${encodeURIComponent(MAIL_SUBJECT)}&body=${encodeURIComponent(MAIL_BODY)}`;

const DELETED_ITEMS = [
  '회원정보와 프로필·사진·명함',
  '작성한 게시글 내용·댓글·첨부파일·좋아요. 다른 회원의 댓글이 달린 게시글은 내용을 지우고 자리만 남깁니다.',
  '주고받은 쪽지와 차단 내역. 상대방 화면에서도 삭제합니다.',
  '로그인·인증 정보, 알림 설정과 푸시 토큰',
  'Apple·카카오 계정 연결. 해당 서비스에 연결 해제를 요청합니다.',
];

const KEPT_ITEMS = [
  { item: '기부금 영수증 발급명세', period: '발급일부터 5년. 기부자명·기부일·금액·거래 증빙만 암호화해 분리 보관합니다.' },
  { item: '기부 관련 장부와 중요 증빙', period: '사업연도 종료일부터 10년. 해당 법령이 적용되는 경우에만 보관합니다.' },
  { item: '처리 완료된 쪽지 신고 자료', period: '처리 완료 후 최대 90일' },
  { item: '삭제 처리 현황 확인 자료', period: '삭제 완료 후 30일' },
  { item: '백업 복원 시 다시 삭제하기 위한 회원 번호', period: '삭제 완료 후 35일' },
  { item: '삭제 전에 만든 서버 백업', period: '백업 생성 후 28일' },
];

export function AccountDeletionRequestGuide() {
  return (
    <Card className="space-y-6 border-border p-6 shadow-none" aria-labelledby="deletion-request-guide">
      <div className="space-y-3">
        <h2 id="deletion-request-guide" className="text-xl font-semibold text-text-primary">앱 없이 계정 삭제 요청하기</h2>
        <p className="leading-7 text-text-secondary">
          {APP_NAME}(운영: {PRIVACY_CONTACT.organization})의 계정은 앱을 설치하지 않아도 이메일로 삭제를 요청할 수 있습니다.
          앱이 있다면 ‘내정보 → 계정 설정 → 계정 삭제 안내 보기’에서 바로 요청할 수 있습니다.
        </p>
      </div>
      <ol className="list-decimal space-y-3 pl-5 leading-7 text-text-primary">
        <li>
          가입할 때 쓴 이메일로{' '}
          <a className="break-all text-primary underline underline-offset-4" href={MAIL_HREF}>{PRIVACY_CONTACT.email}</a>
          에 메일을 보내 주세요. 제목은 ‘계정 삭제 요청’으로 적어 주세요.
        </li>
        <li>본문에 이름, 기수·학과, 로그인 방식(아이디·카카오·Apple), 아이디 또는 가입 이메일을 적어 주세요. 비밀번호나 인증번호는 보내지 마세요.</li>
        <li>담당자({PRIVACY_CONTACT.requestHandler})가 본인 확인 후 요청을 접수하고, 처리 현황 확인번호와 취소 인증번호를 회신합니다. 본인 확인을 위해 추가 정보를 요청할 수 있습니다.</li>
        <li>접수되면 계정 이용이 즉시 중지되고, 7일 뒤 자동으로 삭제합니다. 삭제가 시작되기 전까지는 취소할 수 있으며, 결과는 아래에서 확인번호로 조회할 수 있습니다.</li>
      </ol>
      <div className="space-y-2">
        <h3 className="font-semibold text-text-primary">삭제하는 정보</h3>
        <ul className="list-disc space-y-1 pl-5 text-sm leading-7 text-text-secondary">
          {DELETED_ITEMS.map((item) => <li key={item}>{item}</li>)}
        </ul>
      </div>
      <div className="space-y-2">
        <h3 className="font-semibold text-text-primary">보관하는 정보와 기간</h3>
        <dl className="divide-y divide-border text-sm leading-7">
          {KEPT_ITEMS.map(({ item, period }) => (
            <div key={item} className="grid gap-1 py-2 sm:grid-cols-[minmax(0,14rem)_1fr] sm:gap-4">
              <dt className="font-medium text-text-primary">{item}</dt>
              <dd className="text-text-secondary">{period}</dd>
            </div>
          ))}
        </dl>
        <p className="text-sm leading-7 text-text-secondary">보관하는 정보는 다른 목적에 쓰지 않으며 기간이 끝나면 삭제합니다.</p>
      </div>
    </Card>
  );
}
