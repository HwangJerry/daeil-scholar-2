// PrivacyPolicyPage — Public, accessible privacy information in the site's editorial style.
import type { ReactNode } from 'react';
import { ArrowUpRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { PageMeta } from '../components/seo/PageMeta';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import {
  PRIVACY_COLLECTION,
  PRIVACY_CONTACT,
  PRIVACY_EXTERNAL_SERVICES,
  PRIVACY_SECTIONS,
  PRIVACY_UPDATED_AT,
} from '../domains/privacy/policyContent';
import { cn } from '../lib/utils';
import { usePolicyAnchor } from '../domains/privacy/usePolicyAnchor';

type SectionId = (typeof PRIVACY_SECTIONS)[number]['id'];

function PolicySection({ id, children }: { id: SectionId; children: ReactNode }) {
  const index = PRIVACY_SECTIONS.findIndex((section) => section.id === id);
  const section = PRIVACY_SECTIONS[index];

  return (
    <section
      id={id}
      tabIndex={-1}
      aria-labelledby={`${id}-heading`}
      className="scroll-mt-[calc(var(--landing-header-height)+2rem)] border-t border-border py-9 first:border-t-0 first:pt-0 focus:outline-none md:py-12"
    >
      <p className="text-xs font-semibold tracking-widest text-text-secondary">{String(index + 1).padStart(2, '0')}</p>
      <h2 id={`${id}-heading`} className="mt-3 font-serif text-2xl font-semibold leading-snug tracking-tight sm:text-3xl">
        {section.title}
      </h2>
      <div className="mt-5 space-y-4 text-base leading-8 text-text-secondary">{children}</div>
    </section>
  );
}

export function PrivacyPolicyPage() {
  usePolicyAnchor();

  return (
    <>
      <PageMeta
        title="개인정보처리방침"
        description="대일외국어고등학교 장학회 DFLH의 개인정보 이용 목적, 보관 방식, 공개 범위와 개인정보 문의 방법을 안내합니다."
        canonicalPath="/privacy"
      />
      <header className="border-b border-border-subtle bg-surface px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <div className="mx-auto max-w-[1080px]">
          <p className="text-xs font-semibold uppercase tracking-[0.24em] text-text-secondary">DFLH Privacy Policy</p>
          <h1 className="mt-4 break-keep font-serif text-4xl font-bold leading-tight tracking-tight text-text-primary sm:text-5xl md:text-6xl">
            개인정보처리방침
          </h1>
          <p className="mt-6 max-w-2xl text-base leading-8 text-text-secondary">
            대일의 인연을 이어가는 공간에서, 내 정보가 어떻게 쓰이는지 알 수 있도록.{' '}
            <br className="hidden sm:block" />
            {PRIVACY_CONTACT.organization}의 웹사이트와 DFLH 앱에서 개인정보를 이용하고 관리하는 방법을 안내합니다.
          </p>
          <p className="mt-6 text-sm text-text-secondary">
            최종 수정일 <time dateTime={PRIVACY_UPDATED_AT}>2026년 9월 6일</time>
          </p>
        </div>
      </header>

      <div className="mx-auto grid max-w-[1080px] gap-10 px-5 py-12 sm:px-8 md:px-6 lg:grid-cols-[15rem_minmax(0,1fr)] lg:gap-16 lg:py-16">
        <aside>
          <nav aria-label="개인정보처리방침 목차" className="lg:sticky lg:top-[calc(var(--landing-header-height)+2rem)]">
            <p className="mb-4 text-sm font-semibold text-text-primary">필요한 내용을 찾아보세요.</p>
            <ol className="grid gap-1 sm:grid-cols-2 lg:grid-cols-1">
              {PRIVACY_SECTIONS.map((section, index) => (
                <li key={section.id}>
                  <a
                    href={`#${section.id}`}
                    className={cn(
                      'flex min-h-11 items-start gap-3 rounded-sm py-2 text-sm leading-6 text-text-secondary',
                      'hover:text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                    )}
                  >
                    <span aria-hidden="true" className="shrink-0 font-semibold">{String(index + 1).padStart(2, '0')}</span>
                    {section.title}
                  </a>
                </li>
              ))}
            </ol>
          </nav>
        </aside>

        <article className="min-w-0" aria-label="개인정보 처리 내용">
          <PolicySection id="collection">
            <p>회원가입, 프로필 입력, 쪽지 전송과 서비스 이용 과정에서 아래 정보를 처리합니다. 이용하지 않는 선택 기능의 정보는 직접 입력하지 않으셔도 됩니다.</p>
            <dl className="divide-y divide-border">
              {PRIVACY_COLLECTION.map((item) => (
                <div key={item.title} className="py-5 first:pt-2">
                  <dt className="font-semibold text-text-primary">{item.title}</dt>
                  <dd className="mt-2 space-y-2 text-sm leading-7">
                    <p>{item.information}</p>
                    <p>{item.purpose}</p>
                  </dd>
                </div>
              ))}
            </dl>
          </PolicySection>

          <PolicySection id="sharing">
            <p>동문 검색과 프로필 화면에서는 이름, 기수, 학과와 입력한 프로필 정보를 다른 승인 동문이 볼 수 있습니다. 전화번호와 이메일은 각각의 공개 설정에 따라 표시됩니다.</p>
            <p>쪽지는 송수신자가 확인하며, 신고된 메시지의 원문·신고 사유·설명은 권한 있는 운영자가 검토합니다. 신고자 정보는 신고 대상자에게 공개하지 않습니다.</p>
            <p>장학회가 게시하는 소식과 기부 관련 공개 콘텐츠는 로그인하지 않은 방문자도 볼 수 있습니다. 게시 내용에 관한 개인정보 문의는 아래 담당자에게 남겨주세요.</p>
          </PolicySection>

          <PolicySection id="retention">
            <Card className="border-border bg-primary-light p-6 shadow-none">
              <h3 className="text-base font-semibold text-primary">회원 탈퇴 시 기록 보관 안내</h3>
              <p className="mt-3 text-sm leading-7 text-primary">
                현재 회원 탈퇴는 계정 이용을 중지하는 방식입니다. 탈퇴 후 로그인이 차단되지만, 회원정보·프로필·쪽지·기부 내역·신고 기록은 탈퇴만으로 삭제되지 않습니다. 이 기록들의 보관 종료 시점은 별도로 정해져 있지 않습니다.
              </p>
            </Card>
            <p>소셜 계정 연결도 탈퇴 시 자동으로 해제되지 않습니다. 개인정보 삭제나 처리정지를 원하시면 개인정보 보호 담당자에게 요청할 수 있습니다. 요청에 따른 삭제 범위와 처리 일정은 담당자에게 확인해 주세요.</p>
            <p>탈퇴에 따른 기록 보관과 별도로, 방문 상세 기록은 90일이 지난 자료를 정리하며 집계 통계는 유지합니다. 만료된 인증정보와 차단으로 전달되지 않은 메시지는 각각의 정리 절차에 따라 삭제됩니다.</p>
            <p>서버 백업과 외부 진단 서비스의 보관 기간은 현재 확인 중입니다. 확인되는 내용은 이 페이지에 반영하겠습니다.</p>
          </PolicySection>

          <PolicySection id="services">
            <p>로그인, 알림, 서버 운영과 오류 확인을 위해 아래 서비스를 이용합니다.</p>
            <dl className="divide-y divide-border">
              {PRIVACY_EXTERNAL_SERVICES.map((service) => (
                <div key={service.name} className="py-4 first:pt-0">
                  <dt className="font-semibold text-text-primary">{service.name}</dt>
                  <dd className="mt-2 text-sm leading-7">{service.description}</dd>
                </div>
              ))}
            </dl>
            <p>외부 서비스의 처리위탁·제3자 제공 구분, 국외 이전 대상과 국가, 이전 시기·방법 및 보관 기간의 상세 내용은 확인 중입니다. 관련 문의는 개인정보 보호 담당자에게 보내주세요.</p>
          </PolicySection>

          <PolicySection id="choices">
            <p>프로필 화면에서 정보를 수정하고 전화번호·이메일 공개 여부를 변경할 수 있습니다. 기기의 설정에서는 앱 알림 권한을 변경할 수 있습니다.</p>
            <p>개인정보 열람, 정정, 삭제, 처리정지에 관한 요청은 본인 또는 적법한 대리인이 아래 이메일로 보낼 수 있습니다. 요청 내용을 확인하는 데 필요한 본인 확인이나 대리권 확인이 이루어질 수 있습니다.</p>
            <p>앱에서 서비스 탈퇴를 원하시면 ‘마이페이지 → 계정 설정 → 회원 탈퇴’를 이용해 주세요. 탈퇴 후 정보가 보관되는 방식은 위 ‘얼마나 보관하나요?’ 항목에서 확인할 수 있습니다.</p>
          </PolicySection>

          <PolicySection id="cookies">
            <p>웹사이트는 로그인 상태 유지와 방문 통계에 쿠키를 사용합니다. 방문 식별 쿠키의 유효기간은 발급일로부터 1년이며, 브라우저의 세션 저장소에는 방문 중복 집계 방지와 임시 화면 상태를 저장합니다.</p>
            <p>브라우저 설정에서 쿠키를 차단하거나 저장된 사이트 데이터를 지울 수 있습니다. 쿠키를 차단하면 로그인 등 일부 기능의 이용이 제한될 수 있습니다.</p>
            <p>앱은 로그인 상태와 이용 설정을 기기에 저장하며, 알림을 허용한 기기에는 푸시 토큰을 이용해 알림을 보냅니다.</p>
          </PolicySection>

          <PolicySection id="protection">
            <p>계정 인증과 권한 확인을 통해 회원 기능 및 관리자 기능의 접근을 구분합니다. 신고 처리 기록과 메시지 증거는 권한 있는 관리자만 확인할 수 있도록 관리합니다.</p>
            <p>개인정보가 포함된 화면을 문의에 첨부할 때는 문의와 관계없는 정보를 가려주세요. 비밀번호나 인증번호는 이메일로 보내지 마세요.</p>
          </PolicySection>

          <PolicySection id="contact">
            <p>내 정보에 관한 궁금한 점이나 요청이 있다면 연락해 주세요.</p>
            <Card className="border-border p-6 shadow-none sm:p-8">
              <dl className="space-y-4 text-sm leading-7">
                <div><dt className="text-text-secondary">운영 단체</dt><dd className="font-semibold text-text-primary">{PRIVACY_CONTACT.organization}</dd></div>
                <div><dt className="text-text-secondary">개인정보 보호 담당자</dt><dd className="font-semibold text-text-primary">{PRIVACY_CONTACT.officer}</dd></div>
                <div>
                  <dt className="text-text-secondary">문의 이메일</dt>
                  <dd><a className="inline-flex min-h-11 items-center break-all rounded-sm font-semibold text-primary underline underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary" href={`mailto:${PRIVACY_CONTACT.email}`}>{PRIVACY_CONTACT.email}</a></dd>
                </div>
              </dl>
              <Button asChild variant="outline" className="mt-6 w-full sm:w-auto">
                <Link to="/support">앱 이용 문의 안내<ArrowUpRight aria-hidden="true" className="size-4" /></Link>
              </Button>
            </Card>
            <p>개인정보 처리 내용이 변경되면 이 페이지의 내용과 최종 수정일을 함께 갱신하여 안내하겠습니다.</p>
          </PolicySection>
        </article>
      </div>
    </>
  );
}
