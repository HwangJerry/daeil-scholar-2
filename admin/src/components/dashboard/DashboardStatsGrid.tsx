// DashboardStatsGrid — displays key metric cards for the admin dashboard
import { useQuery } from '@tanstack/react-query';
import { Users, FileText, Heart, BarChart3 } from 'lucide-react';
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
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatsCard
        label="총 회원"
        value={stats?.totalMembers.toLocaleString() ?? '—'}
        icon={<Users className="h-5 w-5" />}
        subtext={`카카오 연동: ${stats?.kakaoLinkedMembers.toLocaleString() ?? '—'}명`}
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
    </div>
  );
}
