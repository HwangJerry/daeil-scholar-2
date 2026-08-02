// FeedPage — Main feed with 2-column desktop layout and infinite-scroll notice list
import { PageMeta } from '../components/seo/PageMeta';
import { HeroSection } from '../components/feed/HeroSection';
import { DonationBanner } from '../components/feed/DonationBanner';
import { FeedList } from '../components/feed/FeedList';
import { ScrollToTopButton } from '../components/ui/ScrollToTopButton';

export function FeedPage() {
  return (
    <>
      <PageMeta canonicalPath="/" />
      <header className="sr-only">
        <h1>대일외고 장학회</h1>
        <p>
          대일외고 장학회 공식 사이트입니다. 대일외국어고등학교(대일외고) 재학생 장학금
          지원 소식과 장학회 소개, 누적 기부액을 제공합니다.
        </p>
      </header>
      <div className="grid gap-4 px-4 py-5 md:grid-cols-[1fr_360px] md:gap-5 md:px-0">
        <div className="md:col-start-1 md:row-start-1">
          <HeroSection />
        </div>

        <aside className="md:col-start-2 md:row-start-1 md:row-span-2">
          <div className="md:sticky md:top-20">
          <DonationBanner />
          </div>
        </aside>

        <div className="md:col-start-1 md:row-start-2">
          <FeedList />
        </div>
      </div>
      <ScrollToTopButton />
    </>
  );
}
