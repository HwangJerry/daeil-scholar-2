// AdminSidebar — left navigation with menu items linked to admin routes
import { NavLink } from 'react-router-dom';
import { LayoutDashboard, FileText, Megaphone, Heart, List, Users } from 'lucide-react';
import { cn } from '../../lib/utils.ts';

const NAV_ITEMS = [
  { to: '/', icon: LayoutDashboard, label: '대시보드', end: true },
  { to: '/notice', icon: FileText, label: '공지 관리', end: false },
  { to: '/ad', icon: Megaphone, label: '광고 관리', end: false },
  { to: '/donation', icon: Heart, label: '기부 설정', end: true },
  { to: '/donation/orders', icon: List, label: '기부 내역', end: false },
  { to: '/member', icon: Users, label: '회원 관리', end: false },
] as const;

export function AdminSidebar() {
  return (
    <aside className="hidden w-56 shrink-0 border-r border-slate-200 bg-white md:block">
      <div className="flex h-14 items-center px-5">
        <span className="text-lg font-bold text-royal-indigo">동문 관리</span>
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
                  : 'text-cool-gray hover:bg-slate-50 hover:text-dark-slate',
              )
            }
          >
            <Icon className="h-4.5 w-4.5" />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
