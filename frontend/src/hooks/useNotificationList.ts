// useNotificationList.ts — Query hook for paginated notification list
import { useQuery } from '@tanstack/react-query';
import { getNotifications } from '../api/notifications';

export function useNotificationList(page = 1, size = 20) {
  return useQuery({
    queryKey: ['notifications', page, size],
    queryFn: () => getNotifications(page, size),
  });
}
