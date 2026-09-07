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
            최종 수정일 <time dateTime={PRIVACY_UPDATED_AT}>2026년 9월 7일</time>
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
              <h3 className="text-base font-semibold text-primary">계정 삭제는 자동으로 진행하며, 필요한 경우 담당자가 처리합니다</h3>
              <p className="mt-3 text-sm leading-7 text-primary">
                앱에서 ‘회원 탈퇴 및 계정 삭제 요청’을 접수하면 계정 이용이 중지됩니다. 자동 삭제를 진행하며 오류나 별도 확인이 필요한 경우 담당자가 처리합니다. 통상 접수일로부터 3일 이내 삭제를 목표로 하며, 10일 이내 처리 결과와 법정 보존 내역을 안내합니다. 처리가 지연되면 담당자가 사유와 예정일을 안내합니다. 앱 로그인 화면의 ‘계정 삭제 처리 현황’에서도 결과를 확인할 수 있습니다.
              </p>
            </Card>
            <p>회원정보·프로필·사진·명함·본인이 작성한 게시글·댓글·첨부파일·좋아요·쪽지와 계정에 연결된 인증정보는 이용 목적이 끝나거나 삭제 요청이 접수되면 지체 없이 삭제합니다. 탈퇴 계정이 주고받은 쪽지는 상대방 화면에서도 삭제합니다. Apple·카카오 연결에 대한 권한 철회도 함께 처리합니다. 법령에 따라 남겨야 하는 자료는 필요한 항목만 별도로 보관하며 다른 서비스 이용에 쓰지 않습니다.</p>
            <dl className="divide-y divide-border">
              <div className="py-4"><dt className="font-semibold text-text-primary">기부금 영수증과 회계 자료</dt><dd className="mt-2 text-sm leading-7">소득세법 제160조의3 또는 법인세법 제112조의2에 따른 보관 의무가 적용되는 기부자별 발급명세는 발급일부터 5년간 보관합니다. 해피나눔에서 디지털로 발급한 영수증 원본은 해피나눔 시스템에 보관합니다. 디지털 발급만으로 홈택스 전자기부금영수증의 법정 예외를 적용하지 않습니다. 상속세 및 증여세법 제51조가 적용되는 공익법인등의 장부와 중요한 증명서류는 해당 사업연도 종료일인 12월 31일부터 10년간 보관합니다. 앱 회원 연결·전화번호·기수·학과 등 보존에 불필요한 정보는 삭제하고, 필요한 기부자명·기부일·금액·거래 증빙은 접근을 제한하여 암호화 분리 보관한 뒤 보존 기간이 끝나면 삭제합니다. 개인을 식별할 수 없는 전체 모금 집계는 유지합니다. 이 기간을 모든 기부 내역이나 회원정보에 일괄 적용하지 않으며, 장학회에 적용되는 보존 의무와 자료의 성격에 따라 항목별로 관리합니다.</dd></div>
              <div className="py-4"><dt className="font-semibold text-text-primary">신고 자료</dt><dd className="mt-2 text-sm leading-7">신고 처리와 이용자 보호에 필요한 동안 보관하며, 처리 완료 후 최대 90일 이내 삭제합니다. 그 전에 보관 목적이 끝나거나 유효한 삭제 요청이 있으면 더 일찍 삭제합니다. 법령상 보존 의무가 있는 증거는 근거와 종료일을 정하여 별도 보관합니다.</dd></div>
              <div className="py-4"><dt className="font-semibold text-text-primary">방문 기록과 삭제 요청 확인 자료</dt><dd className="mt-2 text-sm leading-7">방문 상세 기록은 90일 후 정리하고 개인을 식별할 수 없는 집계만 유지합니다. 외부 삭제 작업에 필요한 식별정보는 별도 암호화하여 임시 보관하고, 외부 처리 확인 시 지체 없이 삭제합니다. 임시 작업 정보는 생성 후 10일이 지나면 사용을 중단하고 정기 정리 작업으로 삭제하며, 미완료 요청은 담당자가 확인합니다. 삭제 완료 시 접수증에서 회원 번호 연결을 제거하며, 처리 결과 조회를 위해 완료 후 30일간 확인 자료를 보관한 뒤 삭제합니다.</dd></div>
            </dl>
            <p>Sentry는 무료 요금제에서 새로 수집하는 오류·성능 기록을 30일간 보관합니다. 무료 요금제 전환 전 신규 계정 체험 기간에 수집한 오류 기록은 최대 90일, 성능 기록은 30일간 보관될 수 있습니다. 보관 기간은 수집 당시 기준이 적용되며 요금제 전환으로 기존 기록의 기간이 단축되지는 않습니다. Sentry의 운영 데이터 백업은 자료 종류에 따라 백업 생성 후 30일 또는 90일에 삭제됩니다. 장학회 서버 백업의 보관 기간은 별도로 확인 중입니다. 계정 삭제 작업에는 백업·업로드 파일·외부 서비스의 해당 개인 데이터 확인도 포함됩니다. 삭제가 확인되지 않은 요청은 완료로 처리하지 않습니다.</p>
            <Button asChild variant="outline"><Link to="/account-deletion">계정 삭제 처리 현황 확인</Link></Button>
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
            <p>Sentry의 진단 데이터 저장 국가는 미국이며, 앱 실행 중 오류 등 진단 정보가 발생하면 암호화된 통신으로 전달됩니다. 그 밖의 외부 서비스에 대한 계약상 처리 범위와 보관 기간의 상세 내용은 확인 중입니다. 관련 문의는 개인정보 보호 담당자에게 보내주세요.</p>
          </PolicySection>

          <PolicySection id="choices">
            <p>프로필 화면에서 정보를 수정하고 전화번호·이메일 공개 여부를 변경할 수 있습니다. 기기의 설정에서는 앱 알림 권한을 변경할 수 있습니다.</p>
            <p>개인정보 열람, 정정, 삭제, 처리정지에 관한 요청은 본인 또는 적법한 대리인이 아래 이메일로 보낼 수 있습니다. 요청 내용을 확인하는 데 필요한 본인 확인이나 대리권 확인이 이루어질 수 있습니다.</p>
            <p>앱의 ‘마이페이지 → 계정 설정 → 회원 탈퇴 및 계정 삭제 요청’에서 별도 이메일 발송 없이 요청할 수 있습니다. 동문 인증 대기 화면에서도 삭제 요청이 가능합니다. 삭제·보존 내역에 이의가 있으면 아래 개인정보 요청 접수 이메일로 연락해 주세요.</p>
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
                <div><dt className="text-text-secondary">개인정보 요청 처리 담당자</dt><dd className="font-semibold text-text-primary">{PRIVACY_CONTACT.requestHandler}</dd></div>
                <div>
                  <dt className="text-text-secondary">개인정보 요청 접수 이메일</dt>
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
