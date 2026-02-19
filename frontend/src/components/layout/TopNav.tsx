// TopNav — Desktop top navigation header with message badge for unread count
import { Link, useLocation } from "react-router-dom";
import { cn } from "../../lib/utils";
import { useUnreadMessages } from "../../hooks/useUnreadMessages";

const NAV_ITEMS = [
  { label: "뉴스피드", href: "/" },
  { label: "동문찾기", href: "/alumni" },
  { label: "기부", href: "/donation" },
  { label: "쪽지", href: "/messages", hasBadge: true },
  { label: "마이페이지", href: "/me" },
];

export default function TopNav() {
  const location = useLocation();
  const { unreadCount } = useUnreadMessages();

  return (
    <header className="hidden md:flex sticky top-0 z-40 w-full bg-surface/80 backdrop-blur-md shadow-xs">
      <div className="container mx-auto flex h-14 items-center justify-between px-4">
        <Link to="/" className="flex items-center gap-2">
          <img src="/logo.png" alt="대일외고 장학회 로고" className="h-8" />
          <span className="text-lg font-bold text-text-primary">
            대일외고 장학회
          </span>
        </Link>

        <nav className="flex items-center gap-6">
          {NAV_ITEMS.map((item) => {
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.href}
                to={item.href}
                className={cn(
                  "relative text-[13px] font-medium transition-colors duration-150",
                  isActive
                    ? "text-primary font-semibold"
                    : "text-text-tertiary hover:text-primary"
                )}
              >
                {item.label}
                {item.hasBadge && unreadCount > 0 && (
                  <span className="absolute -top-2 -right-4 min-w-[16px] h-4 flex items-center justify-center rounded-full bg-error text-white text-[10px] font-bold px-1">
                    {unreadCount > 99 ? "99+" : unreadCount}
                  </span>
                )}
              </Link>
            );
          })}
        </nav>
      </div>
    </header>
  );
}
