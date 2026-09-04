// LandingPage — Full-bleed landing composition with motion, metadata, and landmarks
import { AppDownloadHero } from '../components/landing/AppDownloadHero';
import { FoundationOverviewSection } from '../components/landing/FoundationOverviewSection';
import { GreetingSection } from '../components/landing/GreetingSection';
import { HistorySection } from '../components/landing/HistorySection';
import { LandingFooter } from '../components/landing/LandingFooter';
import { LandingHeader } from '../components/landing/LandingHeader';
import { LatestNewsSection } from '../components/landing/LatestNewsSection';
import { OrganizationSection } from '../components/landing/OrganizationSection';
import { ScholarshipBusinessSection } from '../components/landing/ScholarshipBusinessSection';
import { ScrollReveal } from '../components/landing/ScrollReveal';
import { SupportSection } from '../components/landing/SupportSection';
import { VisionSection } from '../components/landing/VisionSection';
import { BannerAdSection } from '../components/feed/BannerAdSection';
import { PageMeta } from '../components/seo/PageMeta';
import { cn } from '../lib/utils';

export function LandingPage() {
  const focusMainContent = () => {
    document.getElementById('main-content')?.focus({ preventScroll: true });
  };

  return (
    <div className={cn('landing-page min-h-screen bg-background font-sans text-text-primary')}>
      <PageMeta
        title="대일의 오늘과 내일을 잇습니다"
        description="대일외국어고등학교 장학회가 전하는 최신 소식과 장학사업, 동문 네트워크를 만나보세요."
        canonicalPath="/"
      />
      <a
        href="#main-content"
        onClick={focusMainContent}
        className={cn(
          'fixed left-4 top-[calc(var(--landing-header-safe-area-top)+0.75rem)] z-[60] inline-flex min-h-11 -translate-y-[calc(100%+1rem+var(--landing-header-safe-area-top))] items-center rounded-md bg-surface px-4 py-2 text-sm font-semibold text-primary shadow-nav transition-transform focus:translate-y-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        )}
      >
        본문으로 건너뛰기
      </a>
      <LandingHeader />
      <main id="main-content" tabIndex={-1} className={cn('focus:outline-none')}>
        <ScrollReveal>
          <AppDownloadHero />
        </ScrollReveal>
        <ScrollReveal>
          <LatestNewsSection />
        </ScrollReveal>
        <div className={cn('mx-auto max-w-[1080px] px-5 sm:px-8 md:px-6')}>
          <ScrollReveal>
            <BannerAdSection />
          </ScrollReveal>
        </div>
        <ScrollReveal>
          <FoundationOverviewSection />
        </ScrollReveal>
        <ScrollReveal>
          <GreetingSection />
        </ScrollReveal>
        <ScrollReveal>
          <VisionSection />
        </ScrollReveal>
        <ScrollReveal>
          <HistorySection />
        </ScrollReveal>
        <ScrollReveal>
          <OrganizationSection />
        </ScrollReveal>
        <ScrollReveal>
          <ScholarshipBusinessSection />
        </ScrollReveal>
        <ScrollReveal>
          <SupportSection />
        </ScrollReveal>
      </main>
      <LandingFooter />
    </div>
  );
}
