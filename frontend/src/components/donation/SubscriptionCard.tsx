// SubscriptionCard — Displays subscription details with cancel action
import { CreditCard } from 'lucide-react';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { cn } from '../../lib/utils';

const PAY_TYPE_LABELS: Record<string, string> = {
  CARD: '신용카드(체크카드)',
  BANK: '계좌이체',
  HP: '휴대폰',
};

const STATUS_LABELS: Record<string, string> = {
  active: '이용중',
  cancelled: '해지됨',
  paused: '일시정지',
};

interface SubscriptionData {
  subSeq: number;
  amount: number;
  payType: string;
  status: string;
  startDate: string;
  nextBill: string;
}

interface SubscriptionCardProps {
  subscription: SubscriptionData;
  onCancel: () => void;
  isCancelling: boolean;
}

export function SubscriptionCard({ subscription, onCancel, isCancelling }: SubscriptionCardProps) {
  const payLabel = PAY_TYPE_LABELS[subscription.payType] ?? subscription.payType;
  const statusLabel = STATUS_LABELS[subscription.status] ?? subscription.status;
  const isActive = subscription.status === 'active';

  return (
    <Card padding="lg" className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <CreditCard size={18} className="text-primary" />
          <span className="font-bold text-text-primary">정기후원</span>
        </div>
        <span
          className={cn(
            'rounded-full px-3 py-0.5 text-xs font-medium',
            isActive
              ? 'bg-success-subtle text-success-text'
              : 'bg-border-subtle text-text-tertiary',
          )}
        >
          {statusLabel}
        </span>
      </div>

      <div className="rounded-lg bg-background p-4 space-y-3">
        <div className="flex justify-between text-sm">
          <span className="text-text-tertiary">월 후원금액</span>
          <span className="text-primary font-bold text-base">
            {subscription.amount.toLocaleString()}원
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-text-tertiary">결제 수단</span>
          <span className="text-text-primary font-medium">{payLabel}</span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-text-tertiary">시작일</span>
          <span className="text-text-primary font-medium">{subscription.startDate}</span>
        </div>
        {isActive && subscription.nextBill && (
          <div className="flex justify-between text-sm">
            <span className="text-text-tertiary">다음 결제일</span>
            <span className="text-text-primary font-medium">{subscription.nextBill}</span>
          </div>
        )}
      </div>

      {isActive && (
        <Button
          variant="outline"
          className="w-full text-error-text border-error-border hover:bg-error-subtle"
          disabled={isCancelling}
          onClick={onCancel}
        >
          {isCancelling ? '처리 중...' : '정기후원 해지'}
        </Button>
      )}
    </Card>
  );
}
