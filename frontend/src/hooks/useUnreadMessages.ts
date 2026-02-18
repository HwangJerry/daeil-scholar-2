// useUnreadMessages — Polls unread message count every 60 seconds for authenticated users
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useAuth } from './useAuth';

interface UnreadCountResponse {
  count: number;
}

const POLL_INTERVAL_MS = 60_000;

export function useUnreadMessages(): { unreadCount: number } {
  const { user } = useAuth();
  const isAuthenticated = !!user;

  const { data } = useQuery({
    queryKey: ['messages', 'unread-count'],
    queryFn: () => api.get<UnreadCountResponse>('/api/messages/unread-count'),
    refetchInterval: POLL_INTERVAL_MS,
    enabled: isAuthenticated,
  });

  return {
    unreadCount: data?.count ?? 0,
  };
}
