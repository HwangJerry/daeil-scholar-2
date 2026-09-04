// EditorialLayout — Current public-site chrome for standalone editorial pages
import { Outlet } from 'react-router-dom';
import { LandingFooter } from '../landing/LandingFooter';
import { PublicSiteHeader, type PublicSiteNavItem } from './PublicSiteHeader';

const EDITORIAL_NAV_ITEMS: readonly PublicSiteNavItem[] = [
  { id: 'download', label: '다운로드', href: '/#download' },
  { id: 'news', label: '최근 소식', href: '/#news' },
  { id: 'about', label: '장학회 소개', href: '/#about' },
  { id: 'business', label: '장학사업', href: '/#business' },
];

export function EditorialLayout() {
  const focusMainContent = () => {
    document.getElementById('main-content')?.focus({ preventScroll: true });
  };

  return (
    <div className="min-h-screen bg-background font-sans text-text-primary">
      <a
        href="#main-content"
        onClick={focusMainContent}
        className="fixed left-4 top-[calc(var(--landing-header-safe-area-top)+0.75rem)] z-[60] inline-flex min-h-11 -translate-y-[calc(100%+1rem+var(--landing-header-safe-area-top))] items-center rounded-md bg-surface px-4 py-2 text-sm font-semibold text-primary shadow-nav transition-transform focus:translate-y-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      >
        본문으로 건너뛰기
      </a>
      <PublicSiteHeader items={EDITORIAL_NAV_ITEMS} navigationLabel="주요 메뉴" />
      <main
        id="main-content"
        tabIndex={-1}
        className="min-h-[60vh] pt-[var(--landing-header-height)] focus:outline-none"
      >
        <Outlet />
      </main>
      <LandingFooter />
    </div>
  );
}
