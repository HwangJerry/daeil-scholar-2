// BottomNav — Mobile bottom navigation with message badge for unread count
import { Home, Users, Heart, MessageCircle, User } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { cn } from "../../lib/utils";
import { useUnreadMessages } from "../../hooks/useUnreadMessages";

export default function BottomNav() {
  const location = useLocation();
  const path = location.pathname;
  const { unreadCount } = useUnreadMessages();

  const navItems = [
    { label: "홈", icon: Home, href: "/" },
    { label: "동문찾기", icon: Users, href: "/alumni" },
    { label: "기부", icon: Heart, href: "/donation" },
    { label: "쪽지", icon: MessageCircle, href: "/messages", badge: unreadCount },
    { label: "MY", icon: User, href: "/me" },
  ];

  return (
    <nav className="fixed bottom-4 left-4 right-4 rounded-2xl bg-surface/95 backdrop-blur-xl shadow-nav pb-safe z-50 md:hidden">
      <div className="flex items-center justify-around h-14">
        {navItems.map((item) => {
          const isActive = path === item.href;
          const Icon = item.icon;

          return (
            <Link
              key={item.href}
              to={item.href}
              className={cn(
                "relative flex flex-col items-center justify-center w-full h-full gap-0.5 transition-colors duration-150",
                isActive ? "text-primary" : "text-text-placeholder hover:text-text-tertiary"
              )}
            >
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
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
