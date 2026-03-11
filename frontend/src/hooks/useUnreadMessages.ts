// useUnreadMessages.ts — Backward-compatible re-export from useBadges
import { useBadges } from './useBadges';

export function useUnreadMessages() {
  const { unreadMessages } = useBadges();
  return { unreadCount: unreadMessages };
}
