// DonationBanner — Donation progress widget for feed sidebar and mobile inline
import { useQuery } from '@tanstack/react-query';
import { TrendingUp } from 'lucide-react';
import { api } from '../../api/client';
import { formatAmount } from '../../utils/formatAmount';
import { EXTERNAL_DONATION_URL } from '../../constants/donation';
import type { DonationSummary } from '../../types/api';

const STALE_TIME_MS = 5 * 60_000;

export function DonationBanner() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['donation', 'summary'],
    queryFn: () => api.get<DonationSummary>('/api/donation/summary'),
    staleTime: STALE_TIME_MS,
  });

  if (isLoading) {
    return (
      <section id="donation-summary" aria-label="누적 기부액" className="scroll-mt-20">
        <div className="h-48 rounded-[20px] skeleton-shimmer" />
      </section>
    );
  }

  if (isError || !data) {
    return (
      <section
        id="donation-summary"
        aria-label="누적 기부액"
        className="scroll-mt-20 rounded-[20px] bg-surface border border-border p-7 shadow-card"
      >
        <p className="text-sm text-text-tertiary">누적 기부액을 불러오지 못했습니다.</p>
      </section>
    );
  }

  return (
    <section
      id="donation-summary"
      aria-labelledby="donation-summary-heading"
      className="scroll-mt-20 rounded-[20px] bg-surface border border-border p-7 shadow-card"
    >
      <p className="text-[10px] font-semibold text-text-placeholder tracking-widest uppercase mb-4">
        함께 만드는 내일
      </p>

      <div className="flex items-center gap-2 mb-1">
        <TrendingUp size={16} className="text-primary flex-shrink-0" />
        <h2 id="donation-summary-heading" className="text-xs text-text-tertiary">
          누적 기부액
        </h2>
      </div>
      <p className="text-2xl font-bold text-text-primary mb-5">
        {formatAmount(data.displayAmount)}원
      </p>

      <a
        href={EXTERNAL_DONATION_URL}
        target="_blank"
        rel="noopener noreferrer"
        className="block w-full text-center rounded-xl border border-primary text-primary text-sm font-semibold py-2.5 transition-all duration-150 hover:bg-primary hover:text-white"
      >
        기부하기
      </a>
    </section>
  );
}
