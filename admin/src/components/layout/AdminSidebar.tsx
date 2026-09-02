// AdminSidebar — grouped desktop navigation for the admin application
import { NavLink } from 'react-router-dom';
import { cn } from '../../lib/utils.ts';
import { NAV_ITEMS } from './navItems.ts';

type NavItemRoute = (typeof NAV_ITEMS)[number]['to'];
type NavItem = (typeof NAV_ITEMS)[number];

interface NavigationGroup {
  id: string;
  label: string;
  routes: readonly NavItemRoute[];
}

interface SidebarGroupProps {
  group: NavigationGroup;
}

interface SidebarNavItemProps {
  item: NavItem;
}

interface SidebarSectionLabelProps {
  id: string;
  label: string;
}

interface SidebarNavItemsProps {
  items: readonly NavItem[];
}

const NAVIGATION_GROUPS = [
  {
    id: 'content',
    label: '콘텐츠',
    routes: ['/', '/notice', '/disclosure', '/banner-ad', '/history'],
  },
  {
    id: 'donation',
    label: '기부·후원',
    routes: ['/donation'],
  },
  {
    id: 'member',
    label: '회원',
    routes: ['/member', '/member/pending'],
  },
  {
    id: 'settings',
    label: '설정',
    routes: ['/job-categories', '/app-settings', '/app-monitoring'],
  },
] as const satisfies readonly NavigationGroup[];

function getNavigationItems(routes: readonly NavItemRoute[]) {
  return routes.flatMap((route) => {
    const item = NAV_ITEMS.find((navItem) => navItem.to === route);
    return item ? [item] : [];
  });
}

function SidebarBrand() {
  return (
    <div className="flex h-16 items-center border-b border-border-subtle px-6">
      <span className="text-lg font-bold text-royal-indigo">동문 관리</span>
    </div>
  );
}

function ActiveIndicator() {
  return (
    <span
      aria-hidden="true"
      className="absolute inset-y-2 left-0 w-1 rounded-r-full bg-royal-indigo"
    />
  );
}

function SidebarNavItem({ item }: SidebarNavItemProps) {
  const { to, icon: Icon, label, end } = item;

  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          'relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors',
          isActive
            ? 'bg-soft-sky font-semibold text-royal-indigo'
            : 'text-cool-gray hover:bg-background hover:text-dark-slate',
        )
      }
    >
      {({ isActive }) => (
        <>
          {isActive && <ActiveIndicator />}
          <Icon aria-hidden="true" className="h-4.5 w-4.5 shrink-0" />
          <span>{label}</span>
        </>
      )}
    </NavLink>
  );
}

function SidebarSectionLabel({ id, label }: SidebarSectionLabelProps) {
  return (
    <h2
      id={id}
      className="mb-2 px-3 text-caption font-semibold tracking-wider text-text-placeholder"
    >
      {label}
    </h2>
  );
}

function SidebarNavItems({ items }: SidebarNavItemsProps) {
  return (
    <div className="space-y-1">
      {items.map((item) => (
        <SidebarNavItem key={item.to} item={item} />
      ))}
    </div>
  );
}

function SidebarGroup({ group }: SidebarGroupProps) {
  const groupLabelId = `admin-nav-${group.id}`;
  const items = getNavigationItems(group.routes);

  return (
    <section aria-labelledby={groupLabelId}>
      <SidebarSectionLabel id={groupLabelId} label={group.label} />
      <SidebarNavItems items={items} />
    </section>
  );
}

function SidebarNavigation() {
  return (
    <nav aria-label="관리자 메뉴" className="space-y-6 px-3 py-5">
      {NAVIGATION_GROUPS.map((group) => (
        <SidebarGroup key={group.id} group={group} />
      ))}
    </nav>
  );
}

export function AdminSidebar() {
  return (
    <aside className="hidden w-60 shrink-0 border-r border-border bg-surface md:block">
      <SidebarBrand />
      <SidebarNavigation />
    </aside>
  );
}
