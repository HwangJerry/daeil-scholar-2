// useBadges.ts — Polls unread message count every 60 seconds
import { useQuery } from '@tanstack/react-query';
import { getBadges } from '../api/badges';
import { useAuth } from './useAuth';

const POLL_INTERVAL_MS = 60_000;

export function useBadges() {
  const { user } = useAuth();
  const isAuthenticated = !!user;

  const { data } = useQuery({
    queryKey: ['badges'],
    queryFn: getBadges,
    refetchInterval: POLL_INTERVAL_MS,
    enabled: isAuthenticated,
  });

  return {
    unreadMessages: data?.unreadMessages ?? 0,
  };
}
