// RecentNotices — displays the 5 most recent notices on the admin dashboard
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api } from '../../api/client.ts';
import type { AdminNoticeListResponse } from '../../types/api.ts';

export function RecentNotices() {
  const { data } = useQuery({
    queryKey: ['admin', 'notices', 'recent'],
    queryFn: () => api.get<AdminNoticeListResponse>('/api/admin/feed?page=1&size=5'),
    staleTime: 60_000,
  });

  return (
    <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
      <h3 className="mb-4 font-semibold text-dark-slate">최근 공지</h3>
      {data?.items.length ? (
        <ul className="space-y-3">
          {data.items.map((n) => (
            <li key={n.seq} className="flex items-center justify-between">
              <Link
                to={`/notice/${n.seq}/edit`}
                className="truncate text-sm text-dark-slate hover:text-royal-indigo"
              >
                {n.subject}
              </Link>
              <span className="shrink-0 text-xs text-cool-gray">{n.regDate.slice(0, 10)}</span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-cool-gray">공지가 없습니다.</p>
      )}
    </div>
  );
}
