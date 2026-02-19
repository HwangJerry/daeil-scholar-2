// AdminHeader — top bar with hamburger menu, site title, user site link, and logout action
import { LogOut, ExternalLink, Menu } from 'lucide-react';
import { useAuth } from '../../hooks/useAuth.ts';
import { useMobileNav } from '../../hooks/useMobileNav.ts';
import { Button } from '../ui/Button.tsx';

export function AdminHeader() {
  const { user, logout } = useAuth();

  return (
    <header className="flex items-center justify-between border-b border-border bg-white px-6 py-3">
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          className="md:hidden"
          onClick={() => useMobileNav.getState().open()}
          aria-label="메뉴 열기"
        >
          <Menu className="h-5 w-5" />
        </Button>
        <h1 className="text-lg font-bold text-dark-slate">관리자 페이지</h1>
      </div>
      <div className="flex items-center gap-3">
        <a
          href="/"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1 text-sm text-cool-gray hover:text-royal-indigo"
        >
          사용자 사이트
          <ExternalLink className="h-3.5 w-3.5" />
        </a>
        {user && <span className="text-sm text-cool-gray">{user.usrName}</span>}
        <Button variant="ghost" size="sm" onClick={() => void logout()}>
          <LogOut className="mr-1 h-4 w-4" />
          로그아웃
        </Button>
      </div>
    </header>
  );
}
