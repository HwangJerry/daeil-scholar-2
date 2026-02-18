// Account settings section with donation history link and sign-out action
import { Link } from 'react-router-dom';
import { Heart, LogOut } from 'lucide-react';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import { buttonVariants } from '../ui/buttonVariants';
import { cn } from '../../lib/utils';

export function AccountActions() {
  const { logout } = useAuth();

  const handleLogout = async () => {
    await logout();
  };

  return (
    <div className="px-4 space-y-4">
      <div className="rounded-xl bg-surface p-4 shadow-card border border-border-subtle space-y-3">
        <h3 className="font-semibold text-text-primary">활동</h3>
        <Link
          to="/me/donation"
          className={cn(
            buttonVariants({ variant: "ghost" }),
            "w-full justify-start text-text-secondary hover:text-primary hover:bg-primary-light no-underline"
          )}
        >
          <Heart className="mr-2 h-4 w-4" />
          나의 기부내역
        </Link>
      </div>

      <div className="rounded-xl bg-surface p-4 shadow-card border border-border-subtle space-y-3">
        <h3 className="font-semibold text-text-primary">계정</h3>
        <Button
          variant="ghost"
          className="w-full justify-start text-error hover:text-red-600 hover:bg-error-light"
          onClick={handleLogout}
        >
          <LogOut className="mr-2 h-4 w-4" />
          로그아웃
        </Button>
      </div>
    </div>
  );
}
