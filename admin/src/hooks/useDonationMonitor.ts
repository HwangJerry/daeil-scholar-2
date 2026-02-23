// useDonationMonitor — view-model hook for DonationMonitorPage; composes queries and computes derived metrics
import { useDonationSummary } from './useDonationSummary.ts';
import { useDonationHistory } from './useDonationHistory.ts';

export interface DonationSnapshotRow {
  date: string;
  displayAmount: number;
  donorCount: number;
  goalAmount: number;
  achievementRate: number;
}

export function useDonationMonitor() {
  const summary = useDonationSummary();
  const historyQuery = useDonationHistory();

  const historyRows: DonationSnapshotRow[] = (historyQuery.data ?? []).map((s) => {
    const displayAmount = (s.dsTotal ?? 0) + (s.dsManualAdj ?? 0);
    const achievementRate = s.dsGoal > 0 ? Math.round((displayAmount / s.dsGoal) * 100) : 0;
    return {
      date: s.dsDate,
      displayAmount,
      donorCount: s.dsDonorCnt ?? 0,
      goalAmount: s.dsGoal ?? 0,
      achievementRate,
    };
  });

  return {
    summary,
    history: {
      isLoading: historyQuery.isLoading,
      isError: historyQuery.isError,
      refetch: historyQuery.refetch,
      rows: historyRows,
    },
  };
}
