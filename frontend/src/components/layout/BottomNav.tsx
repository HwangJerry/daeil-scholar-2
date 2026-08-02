// BottomNav — Mobile navigation for the public MVP information areas
import { Home, Info } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { cn } from "../../lib/utils";
import { getPublicNavigationSection } from "../../constants/publicNavigation";

export default function BottomNav() {
  const location = useLocation();
  const activeSection = getPublicNavigationSection(location.pathname);

  const navItems = [
    { label: "소식", icon: Home, href: "/", section: "news" },
    { label: "장학회 소개", icon: Info, href: "/about", section: "about" },
  ];

  return (
    <nav className="fixed bottom-4 left-4 right-4 rounded-2xl bg-surface/70 backdrop-blur-2xl border border-white/60 shadow-[0_8px_32px_rgb(26,26,46,0.14),inset_0_1px_0_rgb(255,255,255,0.8)] pb-safe z-50 md:hidden">
      <div className="flex items-center justify-around h-14">
        {navItems.map((item) => {
          const isActive = activeSection === item.section;
          const Icon = item.icon;
          const className = cn(
            "relative flex flex-col items-center justify-center w-full h-full gap-0.5 transition-colors duration-150",
            isActive ? "text-primary" : "text-text-placeholder hover:text-text-tertiary"
          );
          const inner = (
            <>
              <div className="relative">
                <Icon size={22} strokeWidth={isActive ? 2.5 : 1.8} />
              </div>
              <span className="text-[11px] font-medium">{item.label}</span>
              {isActive && (
                <span className="absolute bottom-1.5 h-1 w-1 rounded-full bg-primary" />
              )}
            </>
          );
          return (
            <Link
              key={item.href}
              to={item.href}
              className={className}
              aria-current={isActive ? "page" : undefined}
            >
              {inner}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
