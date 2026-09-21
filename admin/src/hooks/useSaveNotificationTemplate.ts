// useSaveNotificationTemplate — mutation hook that swaps in the refreshed template row
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { saveNotificationTemplate } from '../api/notificationTemplates.ts';
import type { NotificationTemplateView } from '../types/notificationTemplate.ts';
import { NOTIFICATION_TEMPLATES_QUERY_KEY } from './useNotificationTemplates.ts';
import { useToast } from './useToast.ts';

export function useSaveNotificationTemplate(displayName: string) {
  const queryClient = useQueryClient();
  const addToast = useToast((state) => state.addToast);

  return useMutation({
    mutationFn: saveNotificationTemplate,
    onSuccess: (saved) => {
      // The response is the refreshed row, so the list can be patched in place
      // instead of refetched: a second read would also reset the other cards.
      queryClient.setQueryData<NotificationTemplateView[]>(
        NOTIFICATION_TEMPLATES_QUERY_KEY,
        (templates) =>
          templates?.map((template) => (template.key === saved.key ? saved : template)),
      );
      addToast({ variant: 'success', title: `「${displayName}」 문구가 저장되었습니다.` });
    },
    // Failures are reported inside the card, beside the retry and reload
    // actions, so a toast would only duplicate them somewhere less useful.
  });
}
