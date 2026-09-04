// PublicSiteHeader — Shared responsive brand navigation for landing and editorial pages
import { useEffect, useRef, useState } from 'react';
import { Menu, X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';

export interface PublicSiteNavItem {
  id: string;
  label: string;
  href: string;
}

interface PublicSiteHeaderProps {
  activeSectionId?: string | null;
  items: readonly PublicSiteNavItem[];
  navigationLabel: string;
}

const MOBILE_MENU_ID = 'public-site-mobile-navigation';

interface PublicSiteNavigationProps extends PublicSiteHeaderProps {
  isMobile?: boolean;
  onNavigate?: () => void;
}

function PublicSiteNavigation({
  activeSectionId,
  isMobile = false,
  items,
  navigationLabel,
  onNavigate,
}: PublicSiteNavigationProps) {
  return (
    <nav
      id={isMobile ? MOBILE_MENU_ID : undefined}
      aria-label={isMobile ? `${navigationLabel} 모바일` : navigationLabel}
      className={cn(
        isMobile
          ? 'absolute inset-x-0 top-full border-b border-border-subtle bg-surface/95 px-4 py-2 shadow-nav backdrop-blur-sm md:hidden'
          : 'hidden items-center gap-2 md:flex',
      )}
    >
      {items.map((item) => {
        const isActive = activeSectionId === item.id;

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
      })}
    </nav>
  );
}

export function PublicSiteHeader({
  activeSectionId = null,
  items,
  navigationLabel,
}: PublicSiteHeaderProps) {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);

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

  return (
    <header className="landing-header fixed inset-x-0 top-0 z-50 border-b border-border-subtle bg-surface/90 pt-[var(--landing-header-safe-area-top)] shadow-xs backdrop-blur-sm">
      <div className="mx-auto flex h-[var(--landing-header-content-height)] max-w-[1080px] items-center justify-between px-4 md:px-6">
        <a
          href="/"
          aria-label="대일외국어고등학교 장학회 홈"
          className="flex min-h-11 min-w-0 items-center gap-2.5 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface"
        >
          <img
            src="/logo.png"
            alt=""
            className="h-7 w-7 shrink-0 rounded-full object-cover md:h-9 md:w-9"
          />
          <span className="truncate font-serif text-sm font-bold tracking-tight text-text-primary sm:text-base">
            대일외국어고등학교 장학회
          </span>
        </a>

        <PublicSiteNavigation
          activeSectionId={activeSectionId}
          items={items}
          navigationLabel={navigationLabel}
        />

        <Button
          ref={menuButtonRef}
          type="button"
          variant="ghost"
          size="icon"
          aria-label={isMenuOpen ? '메뉴 닫기' : '메뉴 열기'}
          aria-controls={MOBILE_MENU_ID}
          aria-expanded={isMenuOpen}
          onClick={() => setIsMenuOpen((isOpen) => !isOpen)}
          className="ml-2 size-11 shrink-0 md:hidden"
        >
          {isMenuOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}
        </Button>
      </div>

      {isMenuOpen && (
        <PublicSiteNavigation
          activeSectionId={activeSectionId}
          isMobile
          items={items}
          navigationLabel={navigationLabel}
          onNavigate={() => setIsMenuOpen(false)}
        />
      )}
    </header>
  );
}
