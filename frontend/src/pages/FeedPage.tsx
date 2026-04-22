// FeedPage — Main feed with 2-column desktop layout and infinite-scroll notice list
import { PageMeta } from '../components/seo/PageMeta';
import { HeroSection } from '../components/feed/HeroSection';
import { DonationBanner } from '../components/feed/DonationBanner';
import { FeedList } from '../components/feed/FeedList';
import { NetworkWidget } from '../components/feed/NetworkWidget';
import { ScrollToTopButton } from '../components/ui/ScrollToTopButton';

export function FeedPage() {
  return (
    <>
      <PageMeta canonicalPath="/" />
      <header className="sr-only">
        <h1>대일외고 장학회 | 대일외국어고등학교 장학재단</h1>
        <p>
          대일외고 장학회 공식 사이트입니다. 대일외국어고등학교(대일외고) 재학생 장학금
          지원과 동문 네트워크를 운영합니다.
        </p>
      </header>
      <div className="md:grid md:grid-cols-[1fr_360px] md:gap-6 px-4 py-5 md:px-0 md:py-0">
      {/* Left: Main content */}
      <div className="space-y-4 md:space-y-5 md:py-5">
        <HeroSection />
        {/* Mobile: donation banner before feed */}
        <div className="md:hidden">
          <DonationBanner />
        </div>
        <FeedList />
      </div>

      {/* Right: Sidebar (desktop only) */}
      <div className="hidden md:block md:py-5">
        <div className="sticky top-20 space-y-4">
          <DonationBanner />
          <NetworkWidget />
        </div>
      </div>
    </div>
    <ScrollToTopButton />
    </>
  );
}
