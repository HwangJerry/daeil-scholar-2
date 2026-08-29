// FoundationOverviewSection — Editorial landing overview with cumulative donation context
import { useQuery } from '@tanstack/react-query';
import { ArrowUpRight, HandHeart, TrendingUp } from 'lucide-react';
import { api } from '../../api/client';
import { EXTERNAL_DONATION_URL } from '../../constants/donation';
import { cn } from '../../lib/utils';
import type { DonationSummary } from '../../types/api';
import { formatAmount } from '../../utils/formatAmount';
import { Bone } from '../ui/Skeleton';
import { Button } from '../ui/Button';

const DONATION_SUMMARY_STALE_TIME_MS = 5 * 60_000;

function DonationAmountLoading() {
  return (
    <div role="status" aria-label="누적 기부액 불러오는 중" className={cn('space-y-3')}>
      <Bone className={cn('h-4 w-24')} />
      <Bone className={cn('h-10 w-52 max-w-full')} />
      <Bone className={cn('h-3 w-44 max-w-full')} />
    </div>
  );
}

function DonationAmountError() {
  return (
    <div role="alert" className={cn('flex gap-3 text-text-tertiary')}>
      <HandHeart aria-hidden="true" className={cn('mt-0.5 size-5 shrink-0 text-primary')} />
      <div>
        <p className={cn('font-serif text-base font-semibold text-text-primary')}>
          기부 현황을 불러오지 못했습니다.
        </p>
        <p className={cn('mt-1 text-body-sm leading-relaxed')}>잠시 후 다시 확인해 주세요.</p>
      </div>
    </div>
  );
}

function DonationAmount({ amount }: { amount: number }) {
  return (
    <div>
      <div className={cn('flex items-center gap-2 text-text-tertiary')}>
        <TrendingUp aria-hidden="true" className={cn('size-4 text-primary')} />
        <p className={cn('text-xs font-semibold uppercase tracking-[0.18em]')}>누적 기부액</p>
      </div>
      <p className={cn('mt-3 font-serif text-4xl font-bold tracking-tight text-text-primary sm:text-5xl')}>
        {formatAmount(amount)}원
      </p>
      <p className={cn('mt-3 text-body-sm leading-relaxed text-text-tertiary')}>
        동문들의 마음이 모여 후배들의 더 큰 가능성으로 이어집니다.
      </p>
    </div>
  );
}

export function FoundationOverviewSection() {
  const { data, isError, isLoading } = useQuery({
    queryKey: ['donation', 'summary'],
    queryFn: () => api.get<DonationSummary>('/api/donation/summary'),
    staleTime: DONATION_SUMMARY_STALE_TIME_MS,
  });

  return (
    <section
      id="about"
      aria-labelledby="foundation-overview-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] border-t border-border-subtle bg-surface px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div
        className={cn(
          'mx-auto grid max-w-[1080px] gap-12 lg:grid-cols-[minmax(0,1.2fr)_minmax(300px,0.8fr)] lg:items-center lg:gap-20',
        )}
      >
        <div className={cn('max-w-2xl')}>
          <p className={cn('text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder')}>
            About the Foundation
          </p>
          <h2
            id="foundation-overview-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            동문의 마음으로,
            <br />
            후배의 내일을 엽니다.
          </h2>
          <p className={cn('mt-6 max-w-xl text-sm leading-7 text-text-secondary sm:text-base')}>
            대일외국어고등학교 장학회는 동문의 뜻을 모아 안정적인 장학 기반을 만들고,
            후배들의 꿈과 성장을 꾸준히 지원하기 위해 설립되었습니다.
          </p>
          <Button asChild variant="outline" className={cn('mt-8 min-h-11')}>
            <a href={EXTERNAL_DONATION_URL} target="_blank" rel="noopener noreferrer">
              기부하기
              <ArrowUpRight aria-hidden="true" className={cn('size-4')} />
            </a>
          </Button>
        </div>

        <aside
          aria-label="장학회 누적 기부 현황"
          className={cn(
            'border-y border-border py-8 sm:py-10 lg:border-l lg:border-y-0 lg:py-12 lg:pl-12',
          )}
        >
          {isLoading && <DonationAmountLoading />}
          {isError && <DonationAmountError />}
          {data && <DonationAmount amount={data.displayAmount} />}
        </aside>
      </div>
    </section>
  );
}
