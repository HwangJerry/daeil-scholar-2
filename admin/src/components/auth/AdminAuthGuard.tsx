// Admin auth guard — redirects non-admin users to login, shows spinner while loading
import { Navigate } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth.ts';

export function AdminAuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoggedIn, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-royal-indigo border-t-transparent" />
      </div>
    );
  }

  if (!isLoggedIn || !user) {
    return <Navigate to="/login" replace />;
  }

  if (user.usrStatus !== 'ZZZ') {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen gap-4">
        <p className="text-lg text-cool-gray">관리자 권한이 필요합니다.</p>
        <a href="/" className="text-royal-indigo hover:underline">사용자 사이트로 이동</a>
      </div>
    );
  }

  return <>{children}</>;
}
