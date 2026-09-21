// notificationTemplates API — reads and writes the editable SMS and push texts
import { api } from './client.ts';
import type {
  NotificationTemplateView,
  SaveNotificationTemplateRequest,
} from '../types/notificationTemplate.ts';

const TEMPLATES_ENDPOINT = '/api/admin/notification-templates';

export function fetchNotificationTemplates() {
  return api.get<NotificationTemplateView[]>(TEMPLATES_ENDPOINT);
}

/** Resolves to the refreshed row, so the card picks up its new version. */
export function saveNotificationTemplate({
  key,
  title,
  body,
  expectedVersion,
}: SaveNotificationTemplateRequest) {
  return api.put<NotificationTemplateView>(`${TEMPLATES_ENDPOINT}/${encodeURIComponent(key)}`, {
    title,
    body,
    expectedVersion,
  });
}
