// BankTransferConfirm — Shows bank account info after bank transfer order creation
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { CheckCircle, Copy, Check } from 'lucide-react';
import { Button } from '../ui/Button';
import { Card } from '../ui/Card';
import { BANK_NAME, BANK_ACCOUNT } from '../../constants/donation';
const COPY_FEEDBACK_MS = 2000;

interface BankTransferConfirmProps {
  orderSeq: number;
  amount: number;
}

export function BankTransferConfirm({ orderSeq, amount }: BankTransferConfirmProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(BANK_ACCOUNT);
    setCopied(true);
    setTimeout(() => setCopied(false), COPY_FEEDBACK_MS);
  };

  return (
    <div className="animate-fade-in-up">
      <Card padding="lg" className="space-y-6 text-center">
        <CheckCircle size={48} className="mx-auto text-emerald-500" />

        <div className="space-y-1">
          <h2 className="text-lg font-bold text-text-primary">주문이 접수되었습니다</h2>
          <p className="text-sm text-text-secondary">아래 계좌로 입금해 주세요.</p>
        </div>

        <div className="rounded-lg bg-background p-4 space-y-3">
          <div className="flex justify-between text-sm">
            <span className="text-text-tertiary">입금 은행</span>
            <span className="text-text-primary font-medium">{BANK_NAME}</span>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-text-tertiary">계좌번호</p>
              <p className="font-mono text-lg font-medium text-text-primary">{BANK_ACCOUNT}</p>
            </div>
            <button
              type="button"
              onClick={handleCopy}
              className="flex items-center gap-1 rounded-lg border border-border-subtle px-3 py-1.5 text-xs text-text-secondary hover:bg-surface transition-colors duration-150"
            >
              {copied ? <Check size={14} className="text-emerald-500" /> : <Copy size={14} />}
              {copied ? '복사됨' : '복사'}
            </button>
          </div>
          <div className="border-t border-border-subtle pt-3 flex justify-between text-sm">
            <span className="text-text-tertiary">입금 금액</span>
            <span className="text-primary font-bold text-base">{amount.toLocaleString()}원</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-text-tertiary">주문번호</span>
            <span className="text-text-primary font-medium">{orderSeq}</span>
          </div>
        </div>

        <p className="text-xs text-text-tertiary">
          입금 확인 후 기부 내역에 반영됩니다.
        </p>

        <Button className="w-full" asChild>
          <Link to="/me/donation">완료</Link>
        </Button>
      </Card>
    </div>
  );
}
