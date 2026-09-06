// AppSupportPage — Public DFLH app support contacts and issue-reporting guidance
import { ArrowUpRight, Mail, MessageCircle, ShieldCheck } from 'lucide-react';
import { PageMeta } from '../components/seo/PageMeta';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';

const SUPPORT_EMAIL = 'ghkdwp018@gmail.com';
const KAKAO_SUPPORT_URL = 'https://open.kakao.com/o/sWwTIiMi';
const EMAIL_SUPPORT_URL = `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent('[DFLH] 앱 이용 문의')}`;

const ISSUE_REPORT_DETAILS = [
  { title: '기기와 운영체제', description: '사용 중인 기기 모델과 iOS 또는 Android 버전을 알려주세요.' },
  { title: '앱 버전', description: '현재 설치된 DFLH 앱의 버전을 알려주세요.' },
  { title: '문제가 발생한 상황', description: '어떤 화면에서 어떤 동작을 했을 때 문제가 생겼는지 설명해 주세요.' },
  { title: '오류 화면 캡처 · 선택', description: '화면을 첨부하시면 도움이 됩니다. 개인정보는 가린 뒤 보내주세요.' },
] as const;

export function AppSupportPage() {
  return (
    <>
      <PageMeta
        title="앱 이용 문의"
        description="DFLH 앱의 회원가입, 로그인, 동문 인증, 쪽지, 기부 현황에 관한 도움이 필요하시면 카카오톡 또는 이메일로 문의해 주세요."
        canonicalPath="/support"
      />

      <header className="border-b border-border-subtle bg-surface px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <div className="mx-auto max-w-[1080px]">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-text-secondary">
            DFLH App Support
          </p>
          <h1 className="mt-4 font-serif text-4xl font-bold tracking-tight text-text-primary sm:text-5xl md:text-6xl">
            앱 이용 문의
          </h1>
          <p className="mt-6 max-w-2xl text-base leading-8 text-text-secondary">
            대일의 인연을 이어가는 데 어려움이 없도록.
            <br />
            대일외고 장학회 앱 이용 중 궁금하거나 불편한 점을 알려주세요.
          </p>
        </div>
      </header>

      <section aria-labelledby="support-contact-heading" className="px-5 py-12 sm:px-8 md:px-6 md:py-16">
        <div className="mx-auto max-w-[1080px]">
          <div className="mb-7 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <h2 id="support-contact-heading" className="font-serif text-2xl font-semibold tracking-tight sm:text-3xl">
              편한 방법으로 연락해 주세요.
            </h2>
          </div>

          <div className="grid gap-5 md:grid-cols-2">
            <Card className="flex flex-col border-primary bg-primary p-7 text-surface shadow-none sm:p-9">
              <MessageCircle aria-hidden="true" className="size-7 text-primary-muted" />
              <h3 className="mt-6 font-serif text-2xl font-semibold text-surface">카카오톡 문의</h3>
              <p className="mt-3 text-sm leading-7 text-primary-muted">
                카카오톡 오픈채팅으로 궁금한 점을 남겨주세요.
                <br />
                앱 이용 중 겪으신 상황을 함께 알려주시면 도움이 됩니다.
              </p>
              <div className="mt-auto pt-8">
                <Button asChild size="lg" className="w-full bg-kakao text-kakao-text shadow-none hover:bg-kakao/90 hover:shadow-none focus-visible:ring-primary-muted sm:w-auto">
                  <a href={KAKAO_SUPPORT_URL} target="_blank" rel="noopener noreferrer" aria-label="카카오톡으로 문의하기 (새 창)">
                    카카오톡으로 문의하기
                    <ArrowUpRight aria-hidden="true" className="size-4" />
                  </a>
                </Button>
                <p className="mt-3 text-xs leading-6 text-primary-muted">카카오톡 이용을 위해 앱 실행 또는 로그인이 필요할 수 있습니다.</p>
              </div>
            </Card>

            <Card className="flex flex-col border-border p-7 shadow-none sm:p-9">
              <Mail aria-hidden="true" className="size-7 text-text-secondary" />
              <h3 className="mt-6 font-serif text-2xl font-semibold">이메일 문의</h3>
              <p className="mt-3 text-sm leading-7 text-text-secondary">
                카카오톡 이용이 어려우시거나 자세한 설명이 필요하다면
                이메일로 문의해 주세요.
              </p>
              <a href={EMAIL_SUPPORT_URL} className="mt-4 inline-flex min-h-11 items-center self-start break-all rounded-sm text-base font-semibold text-primary underline decoration-border-hover underline-offset-4 hover:decoration-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary">
                {SUPPORT_EMAIL}
              </a>
              <div className="mt-auto pt-6">
                <Button asChild variant="outline" size="lg" className="w-full sm:w-auto">
                  <a href={EMAIL_SUPPORT_URL}>
                    이메일로 문의하기
                    <ArrowUpRight aria-hidden="true" className="size-4" />
                  </a>
                </Button>
                <p className="mt-3 text-xs leading-6 text-text-secondary">메일 앱이 열리지 않으면 위 주소를 복사해 보내주세요.</p>
              </div>
            </Card>
          </div>
        </div>
      </section>

      <section aria-labelledby="support-details-heading" className="px-5 pb-14 sm:px-8 md:px-6 md:pb-20">
        <div className="mx-auto grid max-w-[1080px] gap-8 border-t border-border pt-10 md:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] md:gap-16 md:pt-12">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-text-secondary">Before You Send</p>
            <h2 id="support-details-heading" className="mt-3 font-serif text-2xl font-semibold leading-snug tracking-tight sm:text-3xl">
              오류를 문의하실 때<br />함께 알려주세요.
            </h2>
            <p className="mt-4 max-w-sm text-sm leading-7 text-text-secondary">
              아래 내용을 함께 보내주시면 문제를 확인하는 데 도움이 됩니다.
              일반적인 이용 문의는 궁금한 내용만 남겨주셔도 괜찮습니다.
            </p>
          </div>

          <div>
            <dl className="divide-y divide-border">
              {ISSUE_REPORT_DETAILS.map((detail) => (
                <div key={detail.title} className="py-5 first:pt-0">
                  <dt className="text-base font-semibold text-text-primary">{detail.title}</dt>
                  <dd className="mt-2 text-sm leading-7 text-text-secondary">{detail.description}</dd>
                </div>
              ))}
            </dl>
            <p className="mt-5 flex items-start gap-3 rounded-md bg-primary-light px-4 py-4 text-sm leading-6 text-primary">
              <ShieldCheck aria-hidden="true" className="mt-0.5 size-5 shrink-0" />
              비밀번호나 인증번호는 보내지 마세요.
            </p>
          </div>
        </div>
      </section>
    </>
  );
}
