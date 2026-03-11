// notifications.ts — API client functions for notification system
import { api } from './client';

export interface NotificationItem {
  anSeq: number;
  usrSeq: number;
  anType: string;
  anTitle: string;
  anBody: string;
  anRefSeq: number | null;
  readYn: string;
  regDate: string;
}

export interface NotificationListResponse {
  items: NotificationItem[];
  totalCount: number;
  unreadCount: number;
  page: number;
  size: number;
  totalPages: number;
}

export interface BadgeResponse {
  unreadMessages: number;
  unreadNotifications: number;
}

export function getNotifications(page = 1, size = 20) {
  return api.get<NotificationListResponse>(
    `/api/notifications?page=${page}&size=${size}`,
  );
}

export function getBadges() {
  return api.get<BadgeResponse>('/api/badges');
}

export function markNotificationRead(seq: number) {
  return api.put<{ status: string }>(`/api/notifications/${seq}/read`);
}

export function markAllNotificationsRead() {
  return api.put<{ status: string }>('/api/notifications/read-all');
}
