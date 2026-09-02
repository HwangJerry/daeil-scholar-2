// LandingHeader — Fixed responsive navigation for landing-page anchor sections
import { useEffect, useRef, useState } from 'react';
import { Menu, X } from 'lucide-react';
import { Button } from '../ui/Button';
import { cn } from '../../lib/utils';
import { useActiveLandingSection } from './useActiveLandingSection';

type LandingNavItem = {
  id: string;
  label: string;
  href: `#${string}`;
};

const LANDING_NAV_ITEMS: readonly LandingNavItem[] = [
  { id: 'download', label: '모바일앱', href: '#download' },
  { id: 'news', label: '최근 소식', href: '#news' },
  { id: 'about', label: '장학회 소개', href: '#about' },
  { id: 'business', label: '장학사업', href: '#business' },
];

const ABOUT_SECTION_IDS = ['about', 'greeting', 'vision', 'history', 'organization'];
const LANDING_SECTION_IDS = [
  'download',
  'news',
  ...ABOUT_SECTION_IDS,
  'business',
];
const MOBILE_MENU_ID = 'landing-mobile-navigation';

interface LandingNavigationProps {
  activeSectionId: string | null;
  isMobile?: boolean;
  onNavigate?: () => void;
}

function LandingNavigation({
  activeSectionId,
  isMobile = false,
  onNavigate,
}: LandingNavigationProps) {
  return LANDING_NAV_ITEMS.map((item) => {
    const isAboutSectionActive =
      item.id === 'about' &&
      activeSectionId !== null &&
      ABOUT_SECTION_IDS.includes(activeSectionId);
    const isActive = isAboutSectionActive || activeSectionId === item.id;

    return (
      <a
        key={item.id}
        href={item.href}
        aria-current={isActive ? 'page' : undefined}
        onClick={onNavigate}
        className={cn(
          'rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface',
          isMobile
            ? 'flex min-h-11 items-center px-4 py-3 text-sm'
            : 'inline-flex min-h-11 items-center px-2 py-2 text-[13px]',
          isActive
            ? 'text-primary'
            : 'text-text-tertiary hover:bg-background hover:text-text-primary',
        )}
      >
        <span
          className={cn(
            'relative',
            isActive &&
              'after:absolute after:inset-x-0 after:-bottom-1.5 after:h-0.5 after:bg-primary after:content-[\'\']',
          )}
        >
          {item.label}
        </span>
      </a>
    );
  });
}

export function LandingHeader() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const activeSectionId = useActiveLandingSection(LANDING_SECTION_IDS);

  useEffect(() => {
    if (!isMenuOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;

      setIsMenuOpen(false);
      menuButtonRef.current?.focus();
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isMenuOpen]);

  const closeMobileMenu = () => setIsMenuOpen(false);

  return (
    <header
      className={cn(
        'landing-header fixed inset-x-0 top-0 z-50 border-b border-border-subtle bg-surface/90 shadow-xs backdrop-blur-sm',
      )}
    >
      <div
        className={cn(
          'mx-auto flex h-14 max-w-[1080px] items-center justify-between px-4 md:h-16 md:px-6',
        )}
      >
        <a
          href="/"
          aria-label="대일외국어고등학교 장학회 홈"
          className={cn(
            'flex min-h-11 min-w-0 items-center gap-2.5 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface',
          )}
        >
          <img
            src="/logo.png"
            alt=""
            className={cn('h-7 w-7 shrink-0 rounded-full object-cover md:h-9 md:w-9')}
          />
          <span
            className={cn(
              'truncate font-serif text-sm font-bold tracking-tight text-text-primary sm:text-base',
            )}
          >
            대일외국어고등학교 장학회
          </span>
        </a>

        <nav aria-label="랜딩 페이지" className={cn('hidden items-center gap-2 md:flex')}>
          <LandingNavigation activeSectionId={activeSectionId} />
        </nav>

        <Button
          ref={menuButtonRef}
          type="button"
          variant="ghost"
          size="icon"
          aria-label={isMenuOpen ? '메뉴 닫기' : '메뉴 열기'}
          aria-controls={MOBILE_MENU_ID}
          aria-expanded={isMenuOpen}
          onClick={() => setIsMenuOpen((isOpen) => !isOpen)}
          className={cn('ml-2 size-11 shrink-0 md:hidden')}
        >
          {isMenuOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}
        </Button>
      </div>

      {isMenuOpen && (
        <nav
          id={MOBILE_MENU_ID}
          aria-label="랜딩 페이지 모바일"
          className={cn(
            'absolute inset-x-0 top-full border-b border-border-subtle bg-surface/95 px-4 py-2 shadow-nav backdrop-blur-sm md:hidden',
          )}
        >
          <LandingNavigation
            activeSectionId={activeSectionId}
            isMobile
            onNavigate={closeMobileMenu}
          />
        </nav>
      )}
    </header>
  );
}
