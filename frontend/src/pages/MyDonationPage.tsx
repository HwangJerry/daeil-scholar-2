// MyDonationPage — user donation history list with sort and summary
import { Link } from 'react-router-dom';
import { ArrowLeft, ChevronLeft, ChevronRight, Heart } from 'lucide-react';
import { AuthGuard } from '../components/auth/AuthGuard';
import { buttonVariants } from '../components/ui/buttonVariants';
import { cn } from '../lib/utils';
import { useMyDonations } from '../hooks/useMyDonations';
import type { DonationSort } from '../hooks/useMyDonations';
import type { MyDonationItem } from '../types/donate';

// --- Constants ---

const PAY_TYPE_LABELS: Record<string, string> = {
  CARD: '카드',
  BANK: '계좌이체',
  HP: '휴대폰',
};

const SORT_OPTIONS: { value: DonationSort; label: string }[] = [
  { value: 'latest', label: '최신순' },
  { value: 'amount', label: '금액순' },
];

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
      <h1 className="text-xl font-bold text-text-primary">나의 기부내역</h1>
    </div>
  );
}

function SummaryCard({
  totalAmount,
  totalCount,
}: {
  totalAmount: number;
  totalCount: number;
}) {
  return (
    <div className="rounded-xl bg-gradient-to-br from-primary to-indigo-600 p-5 text-white shadow-xl">
      <p className="text-indigo-200 text-[13px] font-medium mb-1 tracking-wide">
        총 기부금액
      </p>
      <h2 className="text-2xl font-extrabold tracking-tight">
        {totalAmount.toLocaleString()}원
      </h2>
      <p className="text-indigo-200 text-sm mt-2">{totalCount}건</p>
    </div>
  );
}

function SortToggle({
  current,
  onChange,
}: {
  current: DonationSort;
  onChange: (sort: DonationSort) => void;
}) {
  return (
    <div className="flex gap-2">
      {SORT_OPTIONS.map(({ value, label }) => {
        const isActive = value === current;
        return (
          <button
            key={value}
            onClick={() => onChange(value)}
            className={`px-4 py-1.5 rounded-full text-sm font-medium transition-colors duration-150 ${
              isActive
                ? 'bg-primary text-white shadow-sm'
                : 'bg-background text-text-tertiary hover:text-text-secondary'
            }`}
          >
            {label}
          </button>
        );
      })}
    </div>
  );
}

function DonationCard({ item }: { item: MyDonationItem }) {
  const payLabel = PAY_TYPE_LABELS[item.payType] ?? item.payType;
  const dateStr = item.paidAt.slice(0, 10);

  return (
    <div className="rounded-xl bg-surface p-4 shadow-card border border-border-subtle flex items-center justify-between">
      <div className="space-y-1">
        <p className="text-lg font-bold text-text-primary">
          {item.amount.toLocaleString()}원
        </p>
        <div className="flex items-center gap-2 text-xs text-text-tertiary">
          <span>{payLabel}</span>
          <span className="text-border">|</span>
          <span>{dateStr}</span>
        </div>
      </div>
      <div className="flex items-center justify-center h-10 w-10 rounded-full bg-primary-light">
        <Heart size={18} className="text-primary" />
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 space-y-4">
      <Heart size={40} className="text-text-placeholder" />
      <p className="text-text-tertiary text-sm">
        아직 기부내역이 없습니다.
      </p>
      <Link
        to="/donation"
        className={cn(buttonVariants(), "no-underline")}
      >
        기부하기
      </Link>
    </div>
  );
}

function Pagination({
  page,
  totalCount,
  pageSize,
  onPageChange,
}: {
  page: number;
  totalCount: number;
  pageSize: number;
  onPageChange: (p: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const hasPrev = page > 1;
  const hasNext = page < totalPages;

  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-center gap-4 pt-4">
      <button
        disabled={!hasPrev}
        onClick={() => onPageChange(page - 1)}
        className="flex items-center gap-1 text-sm text-text-secondary disabled:text-text-placeholder disabled:cursor-not-allowed transition-colors duration-150"
      >
        <ChevronLeft size={16} />
        이전
      </button>
      <span className="text-sm text-text-tertiary">
        {page} / {totalPages}
      </span>
      <button
        disabled={!hasNext}
        onClick={() => onPageChange(page + 1)}
        className="flex items-center gap-1 text-sm text-text-secondary disabled:text-text-placeholder disabled:cursor-not-allowed transition-colors duration-150"
      >
        다음
        <ChevronRight size={16} />
      </button>
    </div>
  );
}

// --- Loading Skeleton ---

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      <div className="h-28 rounded-xl skeleton-shimmer" />
      <div className="h-8 w-40 rounded-full skeleton-shimmer" />
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="h-20 rounded-xl skeleton-shimmer" />
      ))}
    </div>
  );
}

// --- Page Content ---

function MyDonationContent() {
  const {
    data,
    isLoading,
    isError,
    page,
    sort,
    setPage,
    handleSortChange,
    pageSize,
  } = useMyDonations();

  const items = data?.items ?? [];
  const totalAmount = data?.totalAmount ?? 0;
  const totalCount = data?.totalCount ?? 0;

  return (
    <div className="px-4 py-6 space-y-5 animate-fade-in-up">
      <PageHeader />

      {isLoading ? (
        <LoadingSkeleton />
      ) : isError ? (
        <div className="rounded-xl bg-red-50 border border-red-200 p-6 text-center space-y-2">
          <p className="text-sm text-red-600">기부 내역을 불러올 수 없습니다.</p>
          <p className="text-xs text-text-tertiary">잠시 후 다시 시도해 주세요.</p>
        </div>
      ) : (
        <>
          <SummaryCard totalAmount={totalAmount} totalCount={totalCount} />

          <SortToggle current={sort} onChange={handleSortChange} />

          {items.length === 0 ? (
            <EmptyState />
          ) : (
            <div className="space-y-3">
              {items.map((item) => (
                <DonationCard key={item.orderSeq} item={item} />
              ))}
            </div>
          )}

          <Pagination
            page={page}
            totalCount={totalCount}
            pageSize={pageSize}
            onPageChange={setPage}
          />
        </>
      )}
    </div>
  );
}

// --- Exported Page ---

export function MyDonationPage() {
  return (
    <AuthGuard>
      <MyDonationContent />
    </AuthGuard>
  );
}
