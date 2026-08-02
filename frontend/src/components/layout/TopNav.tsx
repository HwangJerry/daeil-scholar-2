// TopNav — Desktop top navigation with serif brand logo and warm editorial style
import { Link, useLocation } from "react-router-dom";
import { cn } from "../../lib/utils";
import {
  getPublicNavigationSection,
  type PublicNavigationSection,
} from "../../constants/publicNavigation";

type NavItem = {
  label: string;
  href: string;
  section: PublicNavigationSection;
};

const NAV_ITEMS: NavItem[] = [
  { label: "소식", href: "/", section: "news" },
  { label: "장학회 소개", href: "/about", section: "about" },
];

export default function TopNav() {
  const location = useLocation();
  const activeSection = getPublicNavigationSection(location.pathname);

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
            const isActive = activeSection === item.section;
            const className = cn(
              "relative text-[13px] font-medium transition-colors duration-150 pb-1",
              isActive
                ? "text-primary border-b-2 border-primary"
                : "text-text-placeholder hover:text-text-primary"
            );
            return (
              <Link
                key={item.href}
                to={item.href}
                className={className}
                aria-current={isActive ? "page" : undefined}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>
      </div>
    </header>
  );
}
