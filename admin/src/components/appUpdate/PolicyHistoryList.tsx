// PolicyHistoryList — collapsible change log for one platform's update policy
import { useState } from 'react';
import { useAppUpdatePolicyHistory } from '../../hooks/useAppUpdatePolicyHistory.ts';
import type { AppPlatform, AppUpdatePolicy } from '../../types/appUpdate.ts';
import { formatOperator, formatPolicyDateTime } from './formatPolicyDateTime.ts';

function summarizePolicy(raw: string) {
  try {
    const policy = JSON.parse(raw) as AppUpdatePolicy;
    const force = policy.forceEnabled ? `강제 ${policy.minBuild}` : '강제 꺼짐';
    const recommend = policy.recommendEnabled ? `권장 ${policy.recommendedBuild}` : '권장 꺼짐';
    return `${force} · ${recommend} · 최소 OS ${policy.minOsVersion}`;
  } catch {
    return raw;
  }
}

interface PolicyHistoryListProps {
  platform: AppPlatform;
}

export function PolicyHistoryList({ platform }: PolicyHistoryListProps) {
  const [isOpen, setIsOpen] = useState(false);
  const historyQuery = useAppUpdatePolicyHistory(platform, isOpen);
  const entries = historyQuery.data?.entries ?? [];

  return (
    <details
      className="mt-4 border-t border-border-subtle pt-4"
      onToggle={(event) => setIsOpen(event.currentTarget.open)}
    >
      <summary className="cursor-pointer text-sm font-medium text-dark-slate">변경 이력</summary>

      {historyQuery.isLoading && <p className="mt-3 text-sm text-cool-gray">불러오는 중입니다.</p>}
      {historyQuery.isError && (
        <p className="mt-3 text-sm text-error-text">변경 이력을 불러오지 못했습니다.</p>
      )}
      {historyQuery.isSuccess && entries.length === 0 && (
        <p className="mt-3 text-sm text-cool-gray">아직 변경 이력이 없습니다.</p>
      )}

      <ol className="mt-3 space-y-3">
        {entries.map((entry) => (
          <li key={entry.seq} className="rounded-xl bg-background p-3 text-xs text-cool-gray">
            <p className="font-medium text-dark-slate">
              {formatPolicyDateTime(entry.changedAt)} · {formatOperator(entry.changedBy)}
            </p>
            <p className="mt-1.5">이전: {summarizePolicy(entry.before)}</p>
            <p>변경: {summarizePolicy(entry.after)}</p>
          </li>
        ))}
      </ol>
    </details>
  );
}
