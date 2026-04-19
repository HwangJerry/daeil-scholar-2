// DonationPage — Composes donation summary and form
import { PageMeta } from '../components/seo/PageMeta';
import { DonationForm } from '../components/donation/DonationForm';
import { DonationSummaryCard } from '../components/donation/DonationSummaryCard';
import Footer from '../components/layout/Footer';

export function DonationPage() {
  return (
    <>
      <PageMeta title="후원하기" canonicalPath="/donation" />
      <div className="space-y-8 px-4 py-6 pb-20 animate-fade-in-up">
      <div className="text-center space-y-2">
        <h1 className="text-xl font-bold text-text-primary font-serif">기부 안내</h1>
        <p className="text-text-tertiary">대일외고의 미래를 함께 만들어 주세요.</p>
      </div>

      <DonationSummaryCard />

      <DonationForm />
      <Footer />
    </div>
    </>
  );
}
