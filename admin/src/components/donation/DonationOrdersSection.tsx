// DonationOrdersSection — paginated donation order table with search, status filter, and payment toggle
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import { Badge } from '../ui/Badge.tsx';
import { Button } from '../ui/Button.tsx';
import { Pagination } from '../ui/Pagination.tsx';
import { ErrorState } from '../ui/ErrorState.tsx';
import { useDonationOrders } from '../../hooks/useDonationOrders.ts';
import { formatAmount } from '../../lib/formatAmount.ts';

const PAY_TYPE_LABELS: Record<string, string> = {
  CARD: '카드',
  BANK: '계좌이체',
  VBANK: '가상계좌',
};

const GATE_LABELS: Record<string, string> = {
  S: '일시후원',
  P: '정기후원',
};

export function DonationOrdersSection() {
  const {
    data, isLoading, isError, refetch,
    pagination, filters,
    updateOrder, isUpdatingOrder,
  } = useDonationOrders();

  const items = data?.items ?? [];

  const handlePaymentToggle = (oSeq: number, currentPayment: string, amount: number) => {
    const nextPayment = currentPayment === 'Y' ? 'N' : 'Y';
    updateOrder({ seq: oSeq, payment: nextPayment, amount });
  };

  return (
    <div className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <h3 className="mb-4 font-semibold text-dark-slate">기부 주문 관리</h3>

      <div className="mb-4 flex flex-wrap gap-2">
        <Input
          aria-label="이름 검색"
          placeholder="회원명 검색..."
          value={filters.nameFilter}
          onChange={(e) => filters.handleNameChange(e.target.value)}
          className="max-w-xs"
        />
        <Select
          value={filters.statusFilter}
          onChange={(e) => filters.handleStatusChange(e.target.value)}
          className="w-32"
          aria-label="결제 상태"
        >
          <option value="">전체 상태</option>
          <option value="Y">결제완료</option>
          <option value="N">미결제</option>
        </Select>
        <Select
          value={filters.payTypeFilter}
          onChange={(e) => filters.handlePayTypeChange(e.target.value)}
          className="w-32"
          aria-label="결제 방식"
        >
          <option value="">전체 방식</option>
          <option value="CARD">카드</option>
          <option value="BANK">계좌이체</option>
          <option value="VBANK">가상계좌</option>
        </Select>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-3 py-2 font-medium w-16">번호</th>
              <th className="px-3 py-2 font-medium">회원명</th>
              <th className="px-3 py-2 font-medium text-right">금액</th>
              <th className="px-3 py-2 font-medium text-center">방식</th>
              <th className="px-3 py-2 font-medium text-center">유형</th>
              <th className="px-3 py-2 font-medium text-center">결제상태</th>
              <th className="px-3 py-2 font-medium">결제일</th>
              <th className="px-3 py-2 font-medium w-20 text-center">변경</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={8} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-cool-gray">로딩 중...</td>
              </tr>
            ) : items.length ? (
              items.map((order) => (
                <tr key={order.oSeq} className="border-b border-border-light">
                  <td className="px-3 py-2 text-cool-gray">{order.oSeq}</td>
                  <td className="px-3 py-2 text-dark-slate">{order.usrName}</td>
                  <td className="px-3 py-2 text-right text-dark-slate">₩{formatAmount(order.amount)}</td>
                  <td className="px-3 py-2 text-center">
                    <Badge variant="muted">{PAY_TYPE_LABELS[order.payType] ?? order.payType}</Badge>
                  </td>
                  <td className="px-3 py-2 text-center">
                    <Badge variant="default">{GATE_LABELS[order.gate] ?? order.gate}</Badge>
                  </td>
                  <td className="px-3 py-2 text-center">
                    <Badge variant={order.payment === 'Y' ? 'success' : 'warning'}>
                      {order.payment === 'Y' ? '완료' : '미결제'}
                    </Badge>
                  </td>
                  <td className="px-3 py-2 text-cool-gray">{order.payDate?.slice(0, 10) ?? '—'}</td>
                  <td className="px-3 py-2 text-center">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={isUpdatingOrder}
                      onClick={() => handlePaymentToggle(order.oSeq, order.payment, order.amount)}
                    >
                      {order.payment === 'Y' ? '취소' : '확인'}
                    </Button>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-cool-gray">주문 내역이 없습니다.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {data && (
        <Pagination
          page={pagination.page}
          totalPages={Math.ceil(data.total / pagination.pageSize)}
          onPageChange={pagination.setPage}
          pageSize={pagination.pageSize}
          onPageSizeChange={pagination.handlePageSizeChange}
        />
      )}
    </div>
  );
}
