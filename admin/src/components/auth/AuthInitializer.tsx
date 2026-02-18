// Fetches the current user session on admin app startup
import { useEffect } from 'react';
import { useAuth } from '../../hooks/useAuth.ts';

export function AuthInitializer({ children }: { children: React.ReactNode }) {
  const fetchUser = useAuth((s) => s.fetchUser);

  useEffect(() => {
    void fetchUser();
  }, [fetchUser]);

  return <>{children}</>;
}
