// useAuthRedirect — Provides a callback that navigates to /login for auth error handlers
import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

export function useAuthRedirect() {
  const navigate = useNavigate();
  return useCallback(() => navigate('/login'), [navigate]);
}
