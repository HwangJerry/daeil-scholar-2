// Main feed page composing hero banner, donation progress, and infinite-scroll notice list
import { HeroSection } from '../components/feed/HeroSection';
import { DonationBanner } from '../components/feed/DonationBanner';
import { FeedList } from '../components/feed/FeedList';

export function FeedPage() {
  return (
    <div className="space-y-5 px-4 py-5">
      <HeroSection />
      <DonationBanner />
      <FeedList />
    </div>
  );
}
