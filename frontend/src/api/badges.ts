// badges.ts — API client for unified badge count polling
import { api } from './client';

export interface BadgeResponse {
  unreadMessages: number;
}

export function getBadges() {
  return api.get<BadgeResponse>('/api/badges');
}
