// DonationMonitorPage — organizes donation monitoring tools into focused tab panels
import {
  FileSpreadsheet,
  Heart,
  History,
  LayoutDashboard,
  ReceiptText,
  Target,
  TrendingUp,
  Users,
} from 'lucide-react';
import { useState } from 'react';
import { DonationConfigSection } from '../components/donation/DonationConfigSection.tsx';
import { DonationImportSection } from '../components/donation/DonationImportSection.tsx';
import { DonationOrdersSection } from '../components/donation/DonationOrdersSection.tsx';
import { Button } from '../components/ui/Button.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { StatsCard } from '../components/ui/StatsCard.tsx';
import {
  useDonationMonitor,
  type DonationSnapshotRow,
} from '../hooks/useDonationMonitor.ts';
import { formatAmount } from '../lib/formatAmount.ts';
import type { DonationSummary } from '../types/api.ts';

type DonationTab = 'overview' | 'orders' | 'import' | 'history';

const DONATION_TABS = [
  { id: 'overview', label: '현황·설정', icon: LayoutDashboard },
  { id: 'orders', label: '후원 내역', icon: ReceiptText },
  { id: 'import', label: '엑셀 임포트', icon: FileSpreadsheet },
  { id: 'history', label: '스냅샷 이력', icon: History },
] as const;

interface DonationSummaryPanelProps {
  summary?: DonationSummary;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}

function DonationSummaryPanel({
  summary,
  isLoading,
  isError,
  onRetry,
}: DonationSummaryPanelProps) {
  return (
    <div className="space-y-6">
      {isError ? (
        <div className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
          <ErrorState onRetry={onRetry} />
        </div>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4 lg:gap-4">
            <StatsCard
              label="총 기부금"
              value={isLoading ? '—' : `₩${formatAmount(summary?.displayAmount ?? 0)}`}
              icon={<Heart className="h-5 w-5" />}
            />
            <StatsCard
              label="기부자 수"
              value={isLoading ? '—' : `${(summary?.donorCount ?? 0).toLocaleString()}명`}
              icon={<Users className="h-5 w-5" />}
            />
            <StatsCard
              label="달성률"
              value={isLoading ? '—' : `${summary?.achievementRate.toFixed(1) ?? 0}%`}
              icon={<TrendingUp className="h-5 w-5" />}
            />
            <StatsCard
              label="목표금액"
              value={isLoading ? '—' : `₩${formatAmount(summary?.goalAmount ?? 0)}`}
              icon={<Target className="h-5 w-5" />}
            />
          </div>

          {!isLoading && summary && (
            <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
              <div className="mb-2 flex justify-between text-sm text-cool-gray">
                <span>달성률 진행 현황</span>
                <span>{summary.achievementRate.toFixed(1)}%</span>
              </div>
              <div className="h-3 w-full overflow-hidden rounded-full bg-border-light">
                <div
                  className="h-3 rounded-full bg-royal-indigo transition-all"
                  style={{ width: `${Math.min(summary.achievementRate, 100)}%` }}
                />
              </div>
              <p className="mt-2 text-xs text-cool-gray">
                스냅샷 기준일: {summary.snapshotDate?.slice(0, 10) ?? '—'}
              </p>
            </div>
          )}
        </>
      )}

      <DonationConfigSection />
    </div>
  );
}

interface SnapshotHistoryPanelProps {
  rows: DonationSnapshotRow[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}

function SnapshotHistoryPanel({ rows, isLoading, isError, onRetry }: SnapshotHistoryPanelProps) {
  return (
    <section className="rounded-2xl border border-border-light bg-white p-4 shadow-sm md:p-6">
      <div className="mb-4">
        <h3 className="font-semibold text-dark-slate">스냅샷 이력</h3>
        <p className="mt-1 text-sm text-cool-gray">최근 30일간의 기부 현황 변화를 확인합니다.</p>
      </div>
      {isError ? (
        <ErrorState onRetry={onRetry} />
      ) : isLoading ? (
        <p className="py-8 text-center text-sm text-cool-gray">로딩 중...</p>
      ) : rows.length ? (
        <div className="overflow-x-auto rounded-xl border border-border-light">
          <table className="w-full min-w-[640px] text-sm">
            <thead>
              <tr className="border-b border-border-light text-left text-cool-gray">
                <th className="px-3 py-3 font-medium">날짜</th>
                <th className="px-3 py-3 text-right font-medium">기부금</th>
                <th className="px-3 py-3 text-right font-medium">기부자수</th>
                <th className="px-3 py-3 text-right font-medium">목표금액</th>
                <th className="px-3 py-3 text-right font-medium">달성률</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.date} className="border-b border-border-light last:border-b-0">
                  <td className="px-3 py-3 text-dark-slate">{row.date}</td>
                  <td className="px-3 py-3 text-right font-medium text-dark-slate">
                    ₩{formatAmount(row.displayAmount)}
                  </td>
                  <td className="px-3 py-3 text-right text-cool-gray">
                    {row.donorCount.toLocaleString()}명
                  </td>
                  <td className="px-3 py-3 text-right text-cool-gray">
                    ₩{formatAmount(row.goalAmount)}
                  </td>
                  <td className="px-3 py-3 text-right text-cool-gray">
                    {row.achievementRate}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="py-8 text-center text-sm text-cool-gray">이력이 없습니다.</p>
      )}
    </section>
  );
}

export function DonationMonitorPage() {
  const { summary, history } = useDonationMonitor();
  const [activeTab, setActiveTab] = useState<DonationTab>('overview');

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-xl font-bold text-dark-slate">기부 관리</h2>
        <p className="mt-1 text-sm text-cool-gray">기부 현황과 운영 작업을 한곳에서 관리합니다.</p>
      </div>

      <nav
        role="tablist"
        aria-label="기부 관리 메뉴"
        className="flex gap-1 overflow-x-auto rounded-2xl border border-border-light bg-white p-1.5 shadow-sm"
      >
        {DONATION_TABS.map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <Button
              key={tab.id}
              id={`donation-tab-${tab.id}`}
              role="tab"
              type="button"
              variant={isActive ? 'secondary' : 'ghost'}
              aria-selected={isActive}
              aria-controls={`donation-panel-${tab.id}`}
              className="shrink-0 gap-2 rounded-xl px-3 md:flex-1"
              onClick={() => setActiveTab(tab.id)}
            >
              <Icon className="h-4 w-4" />
              {tab.label}
            </Button>
          );
        })}
      </nav>

      <section
        id="donation-panel-overview"
        role="tabpanel"
        aria-labelledby="donation-tab-overview"
        hidden={activeTab !== 'overview'}
      >
        <DonationSummaryPanel
          summary={summary.data}
          isLoading={summary.isLoading}
          isError={summary.isError}
          onRetry={() => void summary.refetch()}
        />
      </section>

      <section
        id="donation-panel-orders"
        role="tabpanel"
        aria-labelledby="donation-tab-orders"
        hidden={activeTab !== 'orders'}
      >
        <DonationOrdersSection />
      </section>

      <section
        id="donation-panel-import"
        role="tabpanel"
        aria-labelledby="donation-tab-import"
        hidden={activeTab !== 'import'}
      >
        <DonationImportSection />
      </section>

      <section
        id="donation-panel-history"
        role="tabpanel"
        aria-labelledby="donation-tab-history"
        hidden={activeTab !== 'history'}
      >
        <SnapshotHistoryPanel
          rows={history.rows}
          isLoading={history.isLoading}
          isError={history.isError}
          onRetry={() => void history.refetch()}
        />
      </section>
    </div>
  );
}
