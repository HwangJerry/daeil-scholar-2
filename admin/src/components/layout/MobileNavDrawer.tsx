// MobileNavDrawer — mobile bottom navigation with a Radix-powered more menu sheet
import { useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { matchPath, NavLink, useLocation } from 'react-router-dom';
import { Ellipsis, X } from 'lucide-react';
import { cn } from '../../lib/utils.ts';
import { Button } from '../ui/Button.tsx';
import { NAV_ITEMS } from './navItems.ts';
import { useMobileNav } from '../../hooks/useMobileNav.ts';

type NavItem = (typeof NAV_ITEMS)[number];

interface MoreMenuSheetProps {
  close: () => void;
}

interface MobileBottomNavigationProps {
  hasActiveMoreItem: boolean;
  isMoreMenuOpen: boolean;
  toggleMoreMenu: () => void;
}

interface MobileNavItemProps {
  item: NavItem;
}

const PRIMARY_MOBILE_ROUTES = new Set(['/', '/donation', '/member']);
const PRIMARY_MOBILE_ITEMS = NAV_ITEMS.filter(({ to }) => PRIMARY_MOBILE_ROUTES.has(to));
const MORE_MENU_ITEMS = NAV_ITEMS.filter(({ to }) => !PRIMARY_MOBILE_ROUTES.has(to));

function MoreMenuItem({ item, close }: MobileNavItemProps & MoreMenuSheetProps) {
  const { to, icon: Icon, label, end } = item;

  return (
    <NavLink
      to={to}
      end={end}
      onClick={close}
      className={({ isActive }) =>
        cn(
          'flex min-h-20 flex-col items-start justify-between rounded-xl border p-3 text-sm font-medium transition-colors',
          isActive
            ? 'border-royal-indigo bg-soft-sky font-semibold text-royal-indigo'
            : 'border-border bg-surface text-cool-gray hover:bg-background hover:text-dark-slate',
        )
      }
    >
      <Icon aria-hidden="true" className="h-5 w-5" />
      <span>{label}</span>
    </NavLink>
  );
}

function MoreMenuSheet({ close }: MoreMenuSheetProps) {
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-40 bg-dark-slate/40 md:hidden" />
      <Dialog.Content
        id="mobile-more-menu"
        className="fixed inset-x-0 bottom-0 z-50 max-h-[calc(100dvh-2rem)] overflow-y-auto rounded-t-2xl border-t border-border bg-surface pb-[calc(env(safe-area-inset-bottom)+1rem)] shadow-xl md:hidden"
      >
        <Dialog.Description className="sr-only">
          대시보드, 기부 관리, 회원 관리 외 관리자 메뉴입니다.
        </Dialog.Description>
        <div className="sticky top-0 border-b border-border-subtle bg-surface px-5 pb-3 pt-2">
          <div aria-hidden="true" className="mx-auto mb-2 h-1 w-10 rounded-full bg-border" />
          <div className="flex items-center justify-between">
            <Dialog.Title className="text-lg font-bold text-dark-slate">
              더보기
            </Dialog.Title>
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon" aria-label="더보기 메뉴 닫기">
                <X aria-hidden="true" className="h-5 w-5 text-cool-gray" />
              </Button>
            </Dialog.Close>
          </div>
        </div>
        <nav aria-label="추가 관리자 메뉴" className="grid grid-cols-2 gap-2 p-4">
          {MORE_MENU_ITEMS.map((item) => (
            <MoreMenuItem key={item.to} item={item} close={close} />
          ))}
        </nav>
      </Dialog.Content>
    </Dialog.Portal>
  );
}

function PrimaryMobileNavItem({ item }: MobileNavItemProps) {
  const { to, icon: Icon, label, end } = item;
  const mobileLabel = label.replace(/\s/g, '');

  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          'flex min-w-0 flex-col items-center justify-center gap-1 text-caption font-medium transition-colors',
          isActive ? 'font-semibold text-royal-indigo' : 'text-cool-gray',
        )
      }
    >
      <Icon aria-hidden="true" className="h-5 w-5" />
      <span className="truncate">{mobileLabel}</span>
    </NavLink>
  );
}

function MobileBottomNavigation({
  hasActiveMoreItem,
  isMoreMenuOpen,
  toggleMoreMenu,
}: MobileBottomNavigationProps) {
  const isMoreMenuHighlighted = isMoreMenuOpen || hasActiveMoreItem;

  return (
    <nav
      aria-label="모바일 관리자 메뉴"
      className="fixed inset-x-0 bottom-0 z-30 border-t border-border bg-surface pb-[env(safe-area-inset-bottom)] md:hidden"
    >
      <div className="grid h-16 grid-cols-4">
        {PRIMARY_MOBILE_ITEMS.map((item) => (
          <PrimaryMobileNavItem key={item.to} item={item} />
        ))}
        <button
          type="button"
          aria-controls="mobile-more-menu"
          aria-expanded={isMoreMenuOpen}
          aria-haspopup="dialog"
          onClick={toggleMoreMenu}
          className={cn(
            'flex min-w-0 flex-col items-center justify-center gap-1 text-caption font-medium transition-colors',
            isMoreMenuHighlighted
              ? 'font-semibold text-royal-indigo'
              : 'text-cool-gray',
          )}
        >
          <Ellipsis aria-hidden="true" className="h-5 w-5" />
          <span>더보기</span>
        </button>
      </div>
    </nav>
  );
}

export function MobileNavDrawer() {
  const { isOpen, close, setOpen, toggle } = useMobileNav();
  const location = useLocation();
  const hasActiveMoreItem = MORE_MENU_ITEMS.some(({ to, end }) =>
    Boolean(matchPath({ path: to, end }, location.pathname)),
  );

  useEffect(() => {
    close();
  }, [location.pathname, close]);

  return (
    <Dialog.Root open={isOpen} onOpenChange={setOpen}>
      <MoreMenuSheet close={close} />
      <MobileBottomNavigation
        hasActiveMoreItem={hasActiveMoreItem}
        isMoreMenuOpen={isOpen}
        toggleMoreMenu={toggle}
      />
    </Dialog.Root>
  );
}
