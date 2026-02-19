// DonationPage — Composes donation summary, form, and bank account info
import { Copy } from 'lucide-react';
import { Button } from '../components/ui/Button';
import { DonationForm } from '../components/donation/DonationForm';
import { DonationSummaryCard } from '../components/donation/DonationSummaryCard';
import { BANK_NAME, BANK_ACCOUNT } from '../constants/donation';

export function DonationPage() {
  return (
    <div className="space-y-8 px-4 py-6 animate-fade-in-up">
      <div className="text-center space-y-2">
        <h1 className="text-xl font-bold text-text-primary font-serif">기부 안내</h1>
        <p className="text-text-tertiary">대일외고의 미래를 함께 만들어 주세요.</p>
      </div>

      <DonationSummaryCard />

      <DonationForm />

      <div className="space-y-4">
        <h3 className="font-bold text-lg text-text-primary">기부 계좌 안내</h3>
        <p className="text-xs text-text-tertiary">카드 결제 외 계좌이체를 원하시면 아래 계좌로 입금해 주세요.</p>
        <div className="flex items-center justify-between p-4 rounded-xl bg-surface border border-border shadow-card">
          <div>
            <p className="text-xs text-text-tertiary">{BANK_NAME}</p>
            <p className="font-mono text-lg font-medium text-text-primary">{BANK_ACCOUNT}</p>
          </div>
          <Button
            size="icon"
            variant="ghost"
            className="text-text-placeholder"
            onClick={() => navigator.clipboard.writeText(BANK_ACCOUNT)}
          >
            <Copy size={18} />
          </Button>
        </div>
      </div>
    </div>
  );
}
