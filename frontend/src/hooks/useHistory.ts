// useHistory — Shared public history query for foundation timeline views
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';

export interface HistoryItem {
  heSeq: number;
  eventDate: string;
  text: string;
  sortOrder: number;
}

export interface HistoryYearGroup {
  year: number;
  items: HistoryItem[];
}

const HISTORY_STALE_TIME_MS = 5 * 60_000;

export function useHistory() {
  return useQuery({
    queryKey: ['history'],
    queryFn: () => api.get<HistoryYearGroup[]>('/api/history'),
    staleTime: HISTORY_STALE_TIME_MS,
  });
}
