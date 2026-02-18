// Fetches the current user session on app startup
import { useEffect } from 'react';
import { useAuth } from '../../hooks/useAuth';

export function AuthInitializer({ children }: { children: React.ReactNode }) {
  const fetchUser = useAuth((s) => s.fetchUser);

  useEffect(() => {
    void fetchUser();
  }, [fetchUser]);

  return <>{children}</>;
}
