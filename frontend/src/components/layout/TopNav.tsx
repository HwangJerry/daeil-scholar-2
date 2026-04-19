// TopNav — Desktop top navigation with serif brand logo and warm editorial style
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
    <header className="hidden md:flex sticky top-0 z-40 w-full bg-surface/95 backdrop-blur-md shadow-xs border-b border-border-subtle">
      <div className="container mx-auto flex h-14 items-center justify-between px-6 max-w-[1080px]">
        <Link to="/" className="flex items-center gap-2.5">
          <img
            src="/logo.png"
            alt="대일외국어고등학교 장학회 로고"
            className="h-8 w-8 rounded-full object-cover flex-shrink-0"
          />
          <span className="text-base font-bold text-text-primary font-serif tracking-tight">
            대일외국어고등학교 장학회
          </span>
        </Link>

        <nav className="flex items-center gap-6">
          {NAV_ITEMS.map((item) => {
            const isActive =
              item.href === "/"
                ? location.pathname === "/"
                : location.pathname === item.href || location.pathname.startsWith(item.href + "/");
            return (
              <Link
                key={item.href}
                to={item.href}
                className={cn(
                  "relative text-[13px] font-medium transition-colors duration-150 pb-1",
                  isActive
                    ? "text-primary border-b-2 border-primary"
                    : "text-text-placeholder hover:text-text-primary"
                )}
              >
                {item.label}
                {item.hasBadge && unreadCount > 0 && (
                  <span className="absolute -top-1 -right-1.5 h-2 w-2 rounded-full bg-error" />
                )}
              </Link>
            );
          })}
        </nav>
      </div>
    </header>
  );
}
