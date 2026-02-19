// MobileNavDrawer — Radix Dialog-based slide-out navigation for mobile viewports
import { useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { NavLink, useLocation } from 'react-router-dom';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils.ts';
import { NAV_ITEMS } from './navItems.ts';
import { useMobileNav } from '../../hooks/useMobileNav.ts';

export function MobileNavDrawer() {
  const { isOpen, close } = useMobileNav();
  const location = useLocation();

  useEffect(() => {
    close();
  }, [location.pathname, close]);

  return (
    <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) close(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40 md:hidden data-[state=open]:animate-in data-[state=open]:fade-in data-[state=closed]:animate-out data-[state=closed]:fade-out" />
        <Dialog.Content className="fixed inset-y-0 left-0 z-50 w-64 bg-white shadow-xl md:hidden data-[state=open]:animate-in data-[state=open]:slide-in-from-left data-[state=closed]:animate-out data-[state=closed]:slide-out-to-left">
          <div className="flex h-14 items-center justify-between border-b border-border px-5">
            <span className="text-lg font-bold text-royal-indigo">동문 관리</span>
            <Dialog.Close aria-label="메뉴 닫기" className="rounded-lg p-1 hover:bg-border-light">
              <X className="h-5 w-5 text-cool-gray" />
            </Dialog.Close>
          </div>
          <nav className="flex flex-col gap-1 px-3 py-2">
            {NAV_ITEMS.map(({ to, icon: Icon, label, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-soft-sky text-royal-indigo'
                      : 'text-cool-gray hover:bg-background hover:text-dark-slate',
                  )
                }
              >
                <Icon className="h-4.5 w-4.5" />
                {label}
              </NavLink>
            ))}
          </nav>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
