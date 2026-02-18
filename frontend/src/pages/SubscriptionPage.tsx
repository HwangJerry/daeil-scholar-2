// SubscriptionPage — Shows current subscription status with cancel option
import { Link } from 'react-router-dom';
import { ArrowLeft, CalendarClock, AlertTriangle } from 'lucide-react';
import { AuthGuard } from '../components/auth/AuthGuard';
import { buttonVariants } from '../components/ui/buttonVariants';
import { SubscriptionCard } from '../components/donation/SubscriptionCard';
import { cn } from '../lib/utils';
import { useSubscription, useCancelSubscription } from '../hooks/useSubscription';

// --- Sub-components ---

function PageHeader() {
  return (
    <div className="flex items-center gap-3">
      <Link
        to="/me"
        className="flex items-center justify-center h-9 w-9 rounded-full hover:bg-background transition-colors duration-150"
      >
        <ArrowLeft size={20} className="text-text-secondary" />
      </Link>
      <h1 className="text-xl font-bold text-text-primary">정기후원 관리</h1>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      <div className="h-10 w-48 rounded-lg skeleton-shimmer" />
      <div className="h-48 rounded-xl skeleton-shimmer" />
    </div>
  );
}

function ErrorState() {
  return (
    <div className="rounded-xl bg-red-50 border border-red-200 p-6 text-center space-y-2">
      <p className="text-sm text-red-600">정기후원 정보를 불러올 수 없습니다.</p>
      <p className="text-xs text-text-tertiary">잠시 후 다시 시도해 주세요.</p>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 space-y-4">
      <CalendarClock size={40} className="text-text-placeholder" />
      <p className="text-text-tertiary text-sm">등록된 정기후원이 없습니다.</p>
      <Link to="/donation" className={cn(buttonVariants(), 'no-underline')}>
        정기후원 등록하기
      </Link>
    </div>
  );
}

// --- Page Content ---

function SubscriptionContent() {
  const { data, isLoading, isError } = useSubscription();
  const cancelMutation = useCancelSubscription();

  const handleCancel = () => {
    if (!data) return;
    const confirmed = window.confirm(
      '정기후원을 해지하시겠습니까?\n해지 후에는 다시 등록해야 합니다.',
    );
    if (confirmed) {
      cancelMutation.mutate(data.subSeq);
    }
  };

  return (
    <div className="px-4 py-6 space-y-5 animate-fade-in-up">
      <PageHeader />

      {isLoading ? (
        <LoadingSkeleton />
      ) : isError ? (
        <ErrorState />
      ) : !data ? (
        <EmptyState />
      ) : (
        <SubscriptionCard
          subscription={data}
          onCancel={handleCancel}
          isCancelling={cancelMutation.isPending}
        />
      )}

      {cancelMutation.isError && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 border border-red-200 p-3">
          <AlertTriangle size={16} className="text-red-500 shrink-0" />
          <p className="text-xs text-red-600">
            {cancelMutation.error?.message ?? '해지 처리에 실패했습니다. 잠시 후 다시 시도해 주세요.'}
          </p>
        </div>
      )}
    </div>
  );
}

// --- Exported Page ---

export function SubscriptionPage() {
  return (
    <AuthGuard>
      <SubscriptionContent />
    </AuthGuard>
  );
}
