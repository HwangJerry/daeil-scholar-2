// useFeedCategories — shared query for every feed category (incl. hidden) in admin order
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type { AdminFeedCategory } from '../types/api.ts';

export const FEED_CATEGORIES_QUERY_KEY = ['admin', 'feed-categories'] as const;

/** Most categories the backend allows; the add button disables at this count. */
export const FEED_CATEGORY_MAX_COUNT = 6;

export function useFeedCategories() {
  return useQuery<AdminFeedCategory[]>({
    queryKey: FEED_CATEGORIES_QUERY_KEY,
    queryFn: () => api.get<AdminFeedCategory[]>('/api/admin/feed-categories'),
  });
}
