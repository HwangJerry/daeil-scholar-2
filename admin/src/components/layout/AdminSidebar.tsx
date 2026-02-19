// AdminSidebar — left navigation with menu items linked to admin routes
import { NavLink } from 'react-router-dom';
import { cn } from '../../lib/utils.ts';
import { NAV_ITEMS } from './navItems.ts';

export function AdminSidebar() {
  return (
    <aside className="hidden w-56 shrink-0 border-r border-border bg-white md:block">
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
                  : 'text-cool-gray hover:bg-background hover:text-dark-slate',
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
