// DashboardPage — composes stats grid, recent notices, and quick action links
import { Link } from 'react-router-dom';
import { Plus, Megaphone } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { DashboardStatsGrid } from '../components/dashboard/DashboardStatsGrid.tsx';
import { RecentNotices } from '../components/dashboard/RecentNotices.tsx';

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold text-dark-slate">대시보드</h2>

      <DashboardStatsGrid />

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
            <Link to="/ad">
              <Button variant="secondary" size="sm">
                <Megaphone className="mr-1 h-4 w-4" />
                광고 관리
              </Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
