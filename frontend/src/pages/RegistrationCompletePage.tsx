// RegistrationCompletePage — Confirmation for registration started in the mobile app.
import { Link } from 'react-router-dom';
import { PageMeta } from '../components/seo/PageMeta';
import { Card } from '../components/ui/Card';

export function RegistrationCompletePage() {
  return (
    <>
      <PageMeta title="가입 신청 완료" noIndex />
      <main className="mx-auto max-w-lg px-5 py-14">
        <Card padding="lg">
          <h1 className="text-xl font-semibold text-text-primary">가입 신청이 완료되었습니다.</h1>
          <p className="mt-4 text-sm leading-relaxed text-text-secondary">
            관리자 승인 후 대일외고 장학회 앱에서 로그인할 수 있습니다. 이 창을 닫고 앱으로 돌아가 주세요.
          </p>
          <Link to="/support" className="mt-6 inline-block text-sm text-primary underline underline-offset-4">
            가입 및 승인 문의
          </Link>
        </Card>
      </main>
    </>
  );
}
