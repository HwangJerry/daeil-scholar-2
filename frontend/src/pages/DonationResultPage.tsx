// DonationResultPage — displays success/failed state after EasyPay PG return
import { useSearchParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle, XCircle } from 'lucide-react';
import { api } from '../api/client';
import { Button } from '../components/ui/Button';
import type { OrderDetail } from '../types/donate';

// --- Constants ---

const FAILURE_REASONS: Record<string, string> = {
  pg_error: 'PG 결제 오류가 발생했습니다.',
  server_error: '서버 처리 중 오류가 발생했습니다.',
  bad_request: '잘못된 요청입니다.',
};

const CONTACT_NUMBER = '02-543-3558';

// --- Sub-components ---

function SuccessResult({ orderSeq }: { orderSeq: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['donation', 'order', orderSeq],
    queryFn: () => api.get<OrderDetail>(`/api/donation/orders/${orderSeq}`),
    staleTime: Infinity,
  });

  return (
    <div className="flex min-h-[50vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm text-center space-y-6 px-4">
        <CheckCircle size={56} className="mx-auto text-success" />

        <div className="space-y-2">
          <h1 className="text-xl font-bold text-text-primary">
            기부해 주셔서 감사합니다
          </h1>
          {isLoading ? (
            <div className="h-12 rounded-lg skeleton-shimmer" />
          ) : isError ? (
            <p className="text-sm text-text-tertiary">주문 정보를 불러올 수 없습니다.</p>
          ) : data ? (
            <div className="rounded-lg bg-background p-4 space-y-1">
              <p className="text-sm text-text-tertiary">기부 금액</p>
              <p className="text-2xl font-bold text-primary">
                {data.amount.toLocaleString()}원
              </p>
              <p className="text-xs text-text-placeholder">
                {new Date(data.paidAt).toLocaleDateString('ko-KR', {
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric',
                })}
              </p>
            </div>
          ) : null}
        </div>

        <p className="text-xs text-text-tertiary leading-relaxed">
          기부금은 연말정산 시 소득공제 혜택을 받을 수 있습니다.
        </p>

        <div className="flex gap-3">
          <Button variant="outline" className="flex-1" asChild>
            <Link to="/">홈으로</Link>
          </Button>
          <Button className="flex-1" asChild>
            <Link to="/donation">기부 현황 보기</Link>
          </Button>
        </div>
      </div>
    </div>
  );
}

function FailedResult({ reason }: { reason: string }) {
  const reasonMessage = FAILURE_REASONS[reason] ?? '알 수 없는 오류가 발생했습니다.';

  return (
    <div className="flex min-h-[50vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm text-center space-y-6 px-4">
        <XCircle size={56} className="mx-auto text-error-text" />

        <div className="space-y-2">
          <h1 className="text-xl font-bold text-text-primary">
            결제에 실패했습니다
          </h1>
          <p className="text-sm text-text-secondary">{reasonMessage}</p>
        </div>

        <p className="text-xs text-text-tertiary">
          문의: {CONTACT_NUMBER}
        </p>

        <Button className="w-full" asChild>
          <Link to="/donation">다시 시도</Link>
        </Button>
      </div>
    </div>
  );
}

// --- Main Component ---

export function DonationResultPage() {
  const [searchParams] = useSearchParams();
  const status = searchParams.get('status');
  const orderSeq = searchParams.get('order') ?? '';
  const reason = searchParams.get('reason') ?? '';

  if (status === 'success' && orderSeq) {
    return <SuccessResult orderSeq={orderSeq} />;
  }

  return <FailedResult reason={reason} />;
}
