// BottomNav — Mobile bottom navigation with message badge
import { Home, Users, Info, Heart, MessageCircle, User } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { cn } from "../../lib/utils";
import { useUnreadMessages } from "../../hooks/useUnreadMessages";
import { EXTERNAL_DONATION_URL } from "../../constants/donation";

export default function BottomNav() {
  const location = useLocation();
  const path = location.pathname;
  const { unreadCount } = useUnreadMessages();

  const navItems = [
    { label: "홈", icon: Home, href: "/" },
    { label: "동문찾기", icon: Users, href: "/alumni" },
    { label: "소개", icon: Info, href: "/about" },
    { label: "기부", icon: Heart, href: EXTERNAL_DONATION_URL, external: true },
    { label: "쪽지", icon: MessageCircle, href: "/messages", badge: unreadCount },
    { label: "MY", icon: User, href: "/me" },
  ];

  return (
    <nav className="fixed bottom-4 left-4 right-4 rounded-2xl bg-surface/70 backdrop-blur-2xl border border-white/60 shadow-[0_8px_32px_rgb(26,26,46,0.14),inset_0_1px_0_rgb(255,255,255,0.8)] pb-safe z-50 md:hidden">
      <div className="flex items-center justify-around h-14">
        {navItems.map((item) => {
          const isActive = item.external
            ? false
            : item.href === '/' ? path === '/' : path === item.href || path.startsWith(item.href + '/');
          const Icon = item.icon;
          const className = cn(
            "relative flex flex-col items-center justify-center w-full h-full gap-0.5 transition-colors duration-150",
            isActive ? "text-primary" : "text-text-placeholder hover:text-text-tertiary"
          );
          const inner = (
            <>
              <div className="relative">
                <Icon size={22} strokeWidth={isActive ? 2.5 : 1.8} />
                {item.badge != null && item.badge > 0 && (
                  <span className="absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full bg-error" />
                )}
              </div>
              <span className="text-[11px] font-medium">{item.label}</span>
              {isActive && (
                <span className="absolute bottom-1.5 h-1 w-1 rounded-full bg-primary" />
              )}
            </>
          );
          if (item.external) {
            return (
              <a
                key={item.href}
                href={item.href}
                target="_blank"
                rel="noopener noreferrer"
                className={className}
              >
                {inner}
              </a>
            );
          }
          return (
            <Link key={item.href} to={item.href} className={className}>
              {inner}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
