// DashboardPage — composes stats grid, recent notices, and quick action links
import { Link } from 'react-router-dom';
import { Plus } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { DashboardStatsGrid } from '../components/dashboard/DashboardStatsGrid.tsx';
import { ActiveUsersChart } from '../components/dashboard/ActiveUsersChart.tsx';
import { RecentNotices } from '../components/dashboard/RecentNotices.tsx';

export function DashboardPage() {
  return (
    <div className="space-y-8">
      <header className="space-y-1">
        <h2 className="text-xl font-bold text-dark-slate">대시보드</h2>
        <p className="text-sm text-cool-gray">회원 현황과 주요 운영 지표를 한눈에 확인하세요.</p>
      </header>

      <DashboardStatsGrid />

      <section aria-labelledby="activity-heading" className="space-y-3">
        <div>
          <h3 id="activity-heading" className="font-semibold text-dark-slate">
            사용자 활동
          </h3>
          <p className="mt-1 text-sm text-cool-gray">최근 방문 흐름과 운영 현황입니다.</p>
        </div>
        <ActiveUsersChart />
      </section>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <RecentNotices />

        <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
          <h3 className="mb-4 font-semibold text-dark-slate">빠른 작업</h3>
          <div className="flex flex-wrap gap-3">
            <Link to="/notice/new">
              <Button size="sm">
                <Plus className="mr-1 h-4 w-4" />
                새 공지 작성
              </Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
