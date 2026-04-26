// SubscriptionLinkSection — Mypage entry card linking to the recurring-donation management page
import { Link } from 'react-router-dom';
import { ChevronRight, HeartHandshake } from 'lucide-react';

export function SubscriptionLinkSection() {
  return (
    <section className="px-4">
      <Link
        to="/me/subscription"
        className="flex items-center justify-between rounded-xl border border-border-subtle bg-white p-4 transition-colors duration-150 hover:bg-background"
      >
        <div className="flex items-center gap-3">
          <HeartHandshake size={20} className="text-primary" />
          <div>
            <p className="text-sm font-medium text-text-primary">정기후원 관리</p>
            <p className="text-xs text-text-tertiary">자동 결제 내역 확인 및 해지</p>
          </div>
        </div>
        <ChevronRight size={18} className="text-text-placeholder" />
      </Link>
    </section>
  );
}
