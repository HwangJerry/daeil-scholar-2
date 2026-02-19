// NetworkWidget — Alumni network summary widget for feed sidebar
import { Link } from 'react-router-dom';
import { Users, ArrowRight } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { cn } from '../../lib/utils';
import { api } from '../../api/client';
import type { AlumniSearchResponse } from '../../types/api';

const AVATAR_COLORS = [
  'bg-cat-notice-bg text-cat-notice-text',
  'bg-cat-event-bg text-cat-event-text',
  'bg-cat-scholarship-bg text-cat-scholarship-text',
  'bg-cat-career-bg text-cat-career-text',
  'bg-primary-light text-primary',
];

export function NetworkWidget() {
  const { data } = useQuery({
    queryKey: ['alumni', 'preview'],
    queryFn: () => api.get<AlumniSearchResponse>('/api/alumni?page=1&size=5'),
    staleTime: 10 * 60_000,
  });

  const previewItems = data?.items?.slice(0, 5) ?? [];
  const totalCount = data?.totalCount ?? 0;
  const weeklyCount = data?.weeklyCount;

  return (
    <div className="rounded-[20px] bg-surface border border-border p-7 shadow-card">
      <div className="flex items-center gap-2 mb-4">
        <Users size={15} className="text-text-placeholder" />
        <p className="text-[10px] font-semibold text-text-placeholder tracking-widest uppercase">
          동문 네트워크
        </p>
      </div>

      {/* Avatar stack */}
      <div className="flex items-center mb-4">
        <div className="flex -space-x-2">
          {previewItems.map((item, idx) => (
            <div
              key={item.fmSeq}
              className={cn(
                'h-8 w-8 rounded-full ring-2 ring-surface flex items-center justify-center text-xs font-bold',
                AVATAR_COLORS[idx % AVATAR_COLORS.length]
              )}
            >
              {item.fmName.charAt(0)}
            </div>
          ))}
        </div>
        {totalCount > 5 && (
          <span className="ml-3 text-sm text-text-tertiary font-medium">
            +{totalCount.toLocaleString()}명
          </span>
        )}
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-3 mb-5">
        <div className="rounded-xl bg-background p-3">
          <p className="text-[10px] text-text-placeholder mb-1">전체 동문</p>
          <p className="text-lg font-bold text-text-primary">
            {totalCount > 0 ? totalCount.toLocaleString() : '—'}
          </p>
        </div>
        <div className="rounded-xl bg-background p-3">
          <p className="text-[10px] text-text-placeholder mb-1">이번 주 가입</p>
          <p className="text-lg font-bold text-text-primary">
            {weeklyCount !== undefined ? weeklyCount.toLocaleString() : '—'}
          </p>
        </div>
      </div>

      <Link
        to="/alumni"
        className="flex items-center justify-center gap-1.5 w-full rounded-xl border border-border text-text-secondary text-sm font-medium py-2.5 transition-all duration-150 hover:border-border-hover hover:text-text-primary"
      >
        동문 찾기
        <ArrowRight size={14} />
      </Link>
    </div>
  );
}
