// DashboardStatsGrid — displays key metric cards for the admin dashboard
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Users, FileText, Heart, BarChart3, UserCheck, Activity, TrendingUp } from 'lucide-react';
import { api } from '../../api/client.ts';
import { StatsCard } from '../ui/StatsCard.tsx';
import { formatAmount } from '../../lib/formatAmount.ts';
import type { DashboardStats } from '../../types/api.ts';

export function DashboardStatsGrid() {
  const { data: stats } = useQuery({
    queryKey: ['admin', 'dashboard'],
    queryFn: () => api.get<DashboardStats>('/api/admin/dashboard'),
    staleTime: 60_000,
  });

  return (
    <section aria-labelledby="key-metrics-heading" className="space-y-4">
      <h3 id="key-metrics-heading" className="sr-only">
        주요 운영 지표
      </h3>

      <div className="relative overflow-hidden rounded-2xl bg-royal-indigo p-5 text-surface shadow-sm sm:p-6 lg:p-8">
        <div
          aria-hidden="true"
          className="absolute -right-16 -top-20 h-52 w-52 rounded-full border border-surface/15"
        />
        <div
          aria-hidden="true"
          className="absolute -bottom-24 right-10 h-44 w-44 rounded-full bg-primary-dark/30"
        />

        <div className="relative flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="mb-4 flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-surface/15">
                <Users className="h-5 w-5" aria-hidden="true" />
              </span>
              <p className="text-sm font-medium text-surface/80">총 회원</p>
            </div>
            <div className="flex items-baseline gap-2">
              <p className="text-4xl font-bold tracking-tight sm:text-5xl">
                {stats?.totalMembers.toLocaleString() ?? '—'}
              </p>
              {stats && <span className="text-base font-medium text-surface/70">명</span>}
            </div>
          </div>

          <div className="w-fit rounded-xl bg-surface/10 px-4 py-3 sm:text-right">
            <p className="text-xs text-surface/70">카카오 연동 회원</p>
            <p className="mt-1 text-lg font-semibold">
              {stats ? `${stats.kakaoLinkedMembers.toLocaleString()}명` : '—'}
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 2xl:grid-cols-6">
        <StatsCard
          label="오늘 DAU"
          value={stats?.dauToday.toLocaleString() ?? '—'}
          icon={<Activity className="h-5 w-5" />}
          subtext="오늘 고유 방문자"
        />
        <StatsCard
          label="MAU (30일)"
          value={stats?.mauCurrent.toLocaleString() ?? '—'}
          icon={<TrendingUp className="h-5 w-5" />}
          subtext="30일 고유 방문자"
        />
        <StatsCard
          label="공지 수"
          value={stats?.totalNotices.toLocaleString() ?? '—'}
          icon={<FileText className="h-5 w-5" />}
        />
        <StatsCard
          label="누적 기부액"
          value={stats ? `₩${formatAmount(stats.donation.displayAmount)}` : '—'}
          icon={<Heart className="h-5 w-5" />}
          subtext={stats ? `달성률 ${stats.donation.achievementRate.toFixed(1)}%` : undefined}
        />
        <StatsCard
          label="광고 CTR"
          value={stats ? `${stats.adStats.ctr.toFixed(1)}%` : '—'}
          icon={<BarChart3 className="h-5 w-5" />}
          subtext={stats ? `노출 ${stats.adStats.totalImpressions.toLocaleString()}` : undefined}
        />
        <Link to="/member/pending" className="block [&>div]:h-full">
          <StatsCard
            label="가입 신청"
            value={stats?.pendingApprovals.toLocaleString() ?? '—'}
            icon={<UserCheck className="h-5 w-5" />}
            subtext="승인 관리로 이동"
          />
        </Link>
      </div>
    </section>
  );
}
