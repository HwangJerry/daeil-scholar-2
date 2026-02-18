// DonationOrderListPage — searchable, paginated donation order table with inline editing
import { useState } from 'react';
import { Input } from '../components/ui/Input.tsx';
import { Select } from '../components/ui/Select.tsx';
import { Pagination } from '../components/ui/Pagination.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { useDonationOrders } from '../hooks/useDonationOrders.ts';

const PAY_TYPE_LABELS: Record<string, string> = {
  CARD: '카드',
};

const PAGE_SIZE = 20;

export function DonationOrderListPage() {
  const {
    data,
    isLoading,
    page,
    search,
    statusFilter,
    payTypeFilter,
    setPage,
    handleSearchChange,
    handleStatusChange,
    handlePayTypeChange,
    updateMutation,
  } = useDonationOrders();

  const [editingSeq, setEditingSeq] = useState<number | null>(null);
  const [editPayment, setEditPayment] = useState('');
  const [editAmount, setEditAmount] = useState('');

  const startEdit = (oSeq: number, payment: string, amount: number) => {
    setEditingSeq(oSeq);
    setEditPayment(payment);
    setEditAmount(String(amount));
  };

  const cancelEdit = () => {
    setEditingSeq(null);
    setEditPayment('');
    setEditAmount('');
  };

  const confirmEdit = () => {
    if (editingSeq === null) return;
    const parsedAmount = Number(editAmount);
    if (Number.isNaN(parsedAmount) || parsedAmount < 0) return;
    updateMutation.mutate(
      { seq: editingSeq, body: { payment: editPayment, amount: parsedAmount } },
      { onSuccess: cancelEdit },
    );
  };

  const formatAmount = (amount: number) =>
    `${amount.toLocaleString('ko-KR')}원`;

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-dark-slate">기부 내역 관리</h2>

      <div className="flex gap-2">
        <Input
          placeholder="기부자 검색..."
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          className="max-w-xs"
        />
        <Select
          value={statusFilter}
          onChange={(e) => handleStatusChange(e.target.value)}
          className="w-32"
        >
          <option value="">전체 상태</option>
          <option value="Y">결제완료</option>
          <option value="N">미결제</option>
        </Select>
        <Select
          value={payTypeFilter}
          onChange={(e) => handlePayTypeChange(e.target.value)}
          className="w-32"
        >
          <option value="">전체 방식</option>
          <option value="CARD">카드</option>
        </Select>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-100 bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-100 text-left text-cool-gray">
              <th className="px-4 py-3 font-medium w-20">주문번호</th>
              <th className="px-4 py-3 font-medium">기부자</th>
              <th className="px-4 py-3 font-medium w-28 text-right">금액</th>
              <th className="px-4 py-3 font-medium w-20 text-center">결제방식</th>
              <th className="px-4 py-3 font-medium w-20 text-center">결제상태</th>
              <th className="px-4 py-3 font-medium w-28">결제일</th>
              <th className="px-4 py-3 font-medium w-28">주문일</th>
              <th className="px-4 py-3 font-medium w-32 text-center">관리</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr>
                <td colSpan={8} className="px-4 py-8 text-center text-cool-gray">
                  로딩 중...
                </td>
              </tr>
            ) : data?.items.length ? (
              data.items.map((order) => {
                const isEditing = editingSeq === order.oSeq;

                return (
                  <tr key={order.oSeq} className="border-b border-slate-50 hover:bg-slate-50">
                    <td className="px-4 py-3 text-cool-gray">{order.oSeq}</td>
                    <td className="px-4 py-3 text-dark-slate">{order.usrName}</td>
                    <td className="px-4 py-3 text-right">
                      {isEditing ? (
                        <Input
                          type="number"
                          value={editAmount}
                          onChange={(e) => setEditAmount(e.target.value)}
                          className="w-28 text-right"
                          min={0}
                        />
                      ) : (
                        <span className="text-dark-slate">{formatAmount(order.amount)}</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-center text-cool-gray">
                      {PAY_TYPE_LABELS[order.payType] ?? order.payType}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {isEditing ? (
                        <Select
                          value={editPayment}
                          onChange={(e) => setEditPayment(e.target.value)}
                          className="w-24"
                        >
                          <option value="Y">결제완료</option>
                          <option value="N">미결제</option>
                        </Select>
                      ) : (
                        <Badge variant={order.payment === 'Y' ? 'success' : 'warning'}>
                          {order.payment === 'Y' ? '결제완료' : '미결제'}
                        </Badge>
                      )}
                    </td>
                    <td className="px-4 py-3 text-cool-gray">
                      {order.payDate?.slice(0, 10) || '—'}
                    </td>
                    <td className="px-4 py-3 text-cool-gray">
                      {order.regDate?.slice(0, 10) || '—'}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {isEditing ? (
                        <div className="flex items-center justify-center gap-1">
                          <button
                            onClick={confirmEdit}
                            disabled={updateMutation.isPending}
                            className="rounded-lg bg-royal-indigo px-3 py-1.5 text-xs font-medium text-white hover:bg-royal-indigo/90 disabled:opacity-50"
                          >
                            {updateMutation.isPending ? '저장중...' : '확인'}
                          </button>
                          <button
                            onClick={cancelEdit}
                            disabled={updateMutation.isPending}
                            className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-cool-gray hover:bg-slate-50 disabled:opacity-50"
                          >
                            취소
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => startEdit(order.oSeq, order.payment, order.amount)}
                          className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-cool-gray hover:bg-slate-50 hover:text-dark-slate"
                        >
                          수정
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })
            ) : (
              <tr>
                <td colSpan={8} className="px-4 py-8 text-center text-cool-gray">
                  기부 내역이 없습니다.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {data && data.total > PAGE_SIZE && (
        <Pagination
          page={page}
          totalPages={Math.ceil(data.total / PAGE_SIZE)}
          onPageChange={setPage}
        />
      )}
    </div>
  );
}
