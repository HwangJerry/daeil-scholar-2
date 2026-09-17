// AppUpdatePolicyPage — per-platform app update policy management
import { RefreshCw, Smartphone } from 'lucide-react';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { PlatformPolicyCard } from '../components/appUpdate/PlatformPolicyCard.tsx';
import { useAppUpdatePolicies } from '../hooks/useAppUpdatePolicies.ts';

export function AppUpdatePolicyPage() {
  const policiesQuery = useAppUpdatePolicies();
  const policies = policiesQuery.data?.policies ?? [];

  return (
    <div className="space-y-6">
      <header>
        <div className="flex items-center gap-2">
          <Smartphone aria-hidden="true" className="h-5 w-5 text-royal-indigo" />
          <h2 className="text-xl font-bold text-dark-slate">앱 업데이트</h2>
        </div>
        <p className="mt-2 text-sm text-cool-gray">
          iOS와 Android의 강제·권장 업데이트를 각각 관리합니다. 최소 빌드는 해당 스토어에 100% 배포가
          끝난 뒤에 올려 주세요.
        </p>
      </header>

      {policiesQuery.isLoading ? (
        <div className="rounded-2xl border border-border-light bg-surface p-6 shadow-sm">
          <div className="flex items-center gap-2 text-sm text-cool-gray">
            <RefreshCw aria-hidden="true" className="h-4 w-4" />
            업데이트 정책을 불러오는 중입니다.
          </div>
        </div>
      ) : policiesQuery.isError ? (
        <div className="rounded-2xl border border-border-light bg-surface shadow-sm">
          <ErrorState
            message="업데이트 정책을 불러오는 데 실패했습니다."
            onRetry={() => void policiesQuery.refetch()}
          />
        </div>
      ) : (
        <section aria-label="플랫폼별 업데이트 정책" className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {policies.map((record) => (
            <PlatformPolicyCard
              key={`${record.platform}:${record.updatedAt}`}
              record={record}
            />
          ))}
        </section>
      )}
    </div>
  );
}
