// DonationBanner — Donation progress widget for feed sidebar and mobile inline
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { TrendingUp } from 'lucide-react';
import { api } from '../../api/client';
import { formatAmount } from '../../utils/formatAmount';
import type { DonationSummary } from '../../types/api';

const STALE_TIME_MS = 5 * 60_000;

export function DonationBanner() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['donation', 'summary'],
    queryFn: () => api.get<DonationSummary>('/api/donation/summary'),
    staleTime: STALE_TIME_MS,
  });

  if (isLoading) {
    return <div className="h-48 rounded-[20px] skeleton-shimmer" />;
  }

  if (isError || !data) return null;

  const rate = Math.min(data.achievementRate, 100);
  const hasGoal = data.goalAmount > 0;

  return (
    <div className="rounded-[20px] bg-surface border border-border p-7 shadow-card">
      <p className="text-[10px] font-semibold text-text-placeholder tracking-widest uppercase mb-4">
        기부 캠페인
      </p>

      <div className="flex items-center gap-2 mb-1">
        <TrendingUp size={16} className="text-primary flex-shrink-0" />
        <span className="text-xs text-text-tertiary">누적 기부액</span>
      </div>
      <p className="text-2xl font-bold text-text-primary mb-1">
        {formatAmount(data.displayAmount)}원
      </p>
      <p className="text-sm text-text-tertiary mb-4">
        {data.donorCount}명 참여
        {hasGoal && <span> · 달성률 {rate.toFixed(0)}%</span>}
      </p>

      {hasGoal && (
        <div className="mb-4">
          <div className="h-1.5 overflow-hidden rounded-full bg-border">
            <div
              className="h-full rounded-full bg-gradient-to-r from-primary to-hero-via animate-fill-bar"
              style={{ width: `${rate}%` }}
            />
          </div>
          <p className="mt-1.5 text-right text-xs text-text-placeholder">
            목표 {formatAmount(data.goalAmount)}원
          </p>
        </div>
      )}

      <Link
        to="/donation"
        className="block w-full text-center rounded-xl border border-primary text-primary text-sm font-semibold py-2.5 transition-all duration-150 hover:bg-primary hover:text-white"
      >
        기부하기
      </Link>
    </div>
  );
}
