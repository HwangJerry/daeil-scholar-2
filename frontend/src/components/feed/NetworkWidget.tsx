// NetworkWidget — Alumni network summary widget for feed sidebar
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../../api/client';
import { Bone } from '../ui/Skeleton';
import type { AlumniWidgetResponse } from '../../types/api';

function NetworkWidgetSkeleton() {
  return (
    <div className="rounded-[20px] bg-surface border border-border p-7 shadow-card">
      <Bone className="h-2.5 w-20 mb-5" />
      <Bone className="h-6 w-full mb-2" />
      <Bone className="h-6 w-3/4 mb-8" />
      <Bone className="h-14 w-full rounded-2xl" />
    </div>
  );
}

export function NetworkWidget() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['alumni', 'widget'],
    queryFn: () => api.get<AlumniWidgetResponse>('/api/alumni/widget'),
    staleTime: 10 * 60_000,
  });

  if (isLoading) return <NetworkWidgetSkeleton />;
  if (isError) return null;

  const totalCount = data?.totalCount ?? 0;

  return (
    <div className="rounded-[20px] bg-surface border border-border p-7 shadow-card">
      <p className="text-[10px] font-semibold text-text-placeholder tracking-widest uppercase mb-5">
        동문 네트워크
      </p>

      <div className="mb-8">
        <span className="block text-2xl font-bold text-text-primary leading-none mb-1">
          {totalCount > 0 ? totalCount.toLocaleString() : '—'}
        </span>
        <span className="text-xs text-text-tertiary">명의 동문이 함께하고 있습니다</span>
      </div>

      <Link
        to="/alumni"
        className="flex items-center justify-center w-full rounded-2xl bg-background text-text-secondary text-sm font-medium py-4 transition-colors duration-150 hover:bg-border"
      >
        동문 찾기 →
      </Link>
    </div>
  );
}
