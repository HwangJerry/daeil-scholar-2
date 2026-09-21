// useNotificationTemplates — TanStack Query hook for the editable notification texts
import { useQuery } from '@tanstack/react-query';
import { fetchNotificationTemplates } from '../api/notificationTemplates.ts';

export const NOTIFICATION_TEMPLATES_QUERY_KEY = ['admin', 'notification-templates'] as const;

export function useNotificationTemplates() {
  return useQuery({
    queryKey: NOTIFICATION_TEMPLATES_QUERY_KEY,
    queryFn: fetchNotificationTemplates,
  });
}
