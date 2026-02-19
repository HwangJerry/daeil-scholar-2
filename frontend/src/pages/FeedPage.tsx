// FeedPage — Main feed with 2-column desktop layout and infinite-scroll notice list
import { HeroSection } from '../components/feed/HeroSection';
import { DonationBanner } from '../components/feed/DonationBanner';
import { FeedList } from '../components/feed/FeedList';
import { NetworkWidget } from '../components/feed/NetworkWidget';

export function FeedPage() {
  return (
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
  );
}
