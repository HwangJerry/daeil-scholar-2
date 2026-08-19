// DonationOrdersSection — canonical donation order search, table, and create/edit entry points
import { Pencil, Plus, RotateCcw } from 'lucide-react';
import { useState } from 'react';
import { useDonationOrderFilters } from '../../hooks/useDonationOrderFilters.ts';
import { useDonationOrdersList } from '../../hooks/useDonationOrders.ts';
import { usePagination } from '../../hooks/usePagination.ts';
import { formatAmount } from '../../lib/formatAmount.ts';
import type {
  DonationOrder,
  DonationOrderFilters,
  DonationSource,
  DonationStatus,
  DonationType,
} from '../../types/api.ts';
import { Badge } from '../ui/Badge.tsx';
import { Button } from '../ui/Button.tsx';
import { ErrorState } from '../ui/ErrorState.tsx';
import { Input } from '../ui/Input.tsx';
import { Pagination } from '../ui/Pagination.tsx';
import { Select } from '../ui/Select.tsx';
import { DonationOrderForm } from './DonationOrderForm.tsx';

const SOURCE_LABELS: Record<DonationSource, string> = {
  happy_nanum: '해피나눔',
  bank_transfer: '계좌이체',
  other: '기타',
};

const DONATION_TYPE_LABELS: Record<DonationType, string> = {
  recurring: '정기기부',
  one_time: '일시기부',
  sponsorship: '후원',
};

const STATUS_DETAILS: Record<DonationStatus, {
  label: string;
  variant: 'default' | 'success' | 'warning' | 'danger' | 'muted';
}> = {
  scheduled: { label: '예약', variant: 'default' },
  pending: { label: '대기', variant: 'warning' },
  completed: { label: '완료', variant: 'success' },
  partially_refunded: { label: '부분 환불', variant: 'warning' },
  cancelled: { label: '취소', variant: 'muted' },
  fully_refunded: { label: '전액 환불', variant: 'danger' },
};

const PAYMENT_METHOD_LABELS = {
  card: '카드',
  bank: '계좌이체',
  virtual_bank: '가상계좌',
  mobile: '휴대폰',
  admin: '관리자 등록',
  other: '기타',
};

function formatPhone(phone: string) {
  if (/^010[0-9]{8}$/.test(phone)) {
    return `${phone.slice(0, 3)}-${phone.slice(3, 7)}-${phone.slice(7)}`;
  }
  return phone;
}

export function DonationOrdersSection() {
  const pagination = usePagination();
  const { filters, setFilter, resetFilters } = useDonationOrderFilters(pagination.resetPage);
  const query = useDonationOrdersList(filters, pagination.page, pagination.pageSize);
  const [isCreating, setIsCreating] = useState(false);
  const [editingOrder, setEditingOrder] = useState<DonationOrder | null>(null);
  const items = query.data?.items ?? [];
  const totalPages = Math.ceil((query.data?.total ?? 0) / pagination.pageSize);

  const handleFilterChange = <Key extends keyof DonationOrderFilters>(
    name: Key,
    value: DonationOrderFilters[Key],
  ) => setFilter(name, value);

  return (
    <section className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="font-semibold text-dark-slate">기부 주문 관리</h3>
          <p className="mt-1 text-sm text-cool-gray">
            단건 기부를 등록하고 캐노니컬 주문 정보를 조회·수정합니다.
          </p>
        </div>
        <Button onClick={() => setIsCreating(true)}>
          <Plus className="mr-2 h-4 w-4" />새 기부 등록
        </Button>
      </div>

      <div className="mb-5 grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-7">
        <Input
          aria-label="기부자명 검색"
          placeholder="기부자명"
          value={filters.name}
          onChange={(event) => handleFilterChange('name', event.target.value)}
        />
        <Input
          aria-label="전화번호 검색"
          placeholder="전화번호"
          value={filters.phone}
          onChange={(event) => handleFilterChange('phone', event.target.value)}
        />
        <Input
          aria-label="거래번호 검색"
          placeholder="거래번호"
          value={filters.transactionNumber}
          onChange={(event) => handleFilterChange('transactionNumber', event.target.value)}
        />
        <Select
          aria-label="유입 경로 검색"
          value={filters.source}
          onChange={(event) => handleFilterChange('source', event.target.value as DonationSource | '')}
        >
          <option value="">전체 경로</option>
          <option value="happy_nanum">해피나눔</option>
          <option value="bank_transfer">계좌이체</option>
          <option value="other">기타</option>
        </Select>
        <Select
          aria-label="주문 상태 검색"
          value={filters.status}
          onChange={(event) => handleFilterChange('status', event.target.value as DonationStatus | '')}
        >
          <option value="">전체 상태</option>
          <option value="scheduled">예약</option>
          <option value="pending">대기</option>
          <option value="completed">완료</option>
          <option value="partially_refunded">부분 환불</option>
          <option value="cancelled">취소</option>
          <option value="fully_refunded">전액 환불</option>
        </Select>
        <Select
          aria-label="기부유형 검색"
          value={filters.donationType}
          onChange={(event) => handleFilterChange('donationType', event.target.value as DonationType | '')}
        >
          <option value="">전체 유형</option>
          <option value="recurring">정기기부</option>
          <option value="one_time">일시기부</option>
          <option value="sponsorship">후원</option>
        </Select>
        <Button variant="outline" onClick={resetFilters}>
          <RotateCcw className="mr-2 h-4 w-4" />초기화
        </Button>
      </div>

      <div className="overflow-x-auto rounded-xl border border-border-light">
        <table className="w-full min-w-[1480px] text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <th className="px-3 py-3 font-medium">기부일자</th>
              <th className="px-3 py-3 font-medium">기부자명</th>
              <th className="px-3 py-3 font-medium">기수 / 학과</th>
              <th className="px-3 py-3 font-medium">전화</th>
              <th className="px-3 py-3 text-center font-medium">유입 경로</th>
              <th className="px-3 py-3 text-center font-medium">기부유형</th>
              <th className="px-3 py-3 text-right font-medium">총액 / 환불 / 실수령</th>
              <th className="px-3 py-3 text-center font-medium">상태</th>
              <th className="px-3 py-3 text-center font-medium">결제방식</th>
              <th className="px-3 py-3 text-center font-medium">계정 연결</th>
              <th className="w-20 px-3 py-3 text-center font-medium">작업</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {query.isError ? (
              <ErrorState colSpan={11} onRetry={() => void query.refetch()} />
            ) : query.isLoading ? (
              <tr><td colSpan={11} className="px-3 py-10 text-center text-cool-gray">로딩 중...</td></tr>
            ) : items.length === 0 ? (
              <tr><td colSpan={11} className="px-3 py-10 text-center text-cool-gray">조건에 맞는 주문이 없습니다.</td></tr>
            ) : items.map((order) => {
              const status = STATUS_DETAILS[order.status];
              return (
                <tr key={order.orderSeq} className="border-b border-border-light last:border-b-0 hover:bg-background">
                  <td className="px-3 py-3 text-dark-slate">{order.donationDate}</td>
                  <td className="px-3 py-3 font-medium text-dark-slate">{order.donor.name}</td>
                  <td className="px-3 py-3 text-cool-gray">{order.donor.cohort} / {order.donor.department}</td>
                  <td className="px-3 py-3 text-cool-gray">{formatPhone(order.donor.phone)}</td>
                  <td className="px-3 py-3 text-center"><Badge variant="muted">{SOURCE_LABELS[order.source]}</Badge></td>
                  <td className="px-3 py-3 text-center"><Badge>{DONATION_TYPE_LABELS[order.donationType]}</Badge></td>
                  <td className="px-3 py-3 text-right tabular-nums text-dark-slate">
                    <div>₩{formatAmount(order.grossAmount)}</div>
                    <div className="text-xs text-cool-gray">-₩{formatAmount(order.refundedAmount)} / ₩{formatAmount(order.netReceivedAmount)}</div>
                  </td>
                  <td className="px-3 py-3 text-center"><Badge variant={status.variant}>{status.label}</Badge></td>
                  <td className="px-3 py-3 text-center text-cool-gray">{PAYMENT_METHOD_LABELS[order.paymentMethod]}</td>
                  <td className="px-3 py-3 text-center">
                    <Badge variant={order.accountUsrSeq == null ? 'muted' : 'success'}>
                      {order.accountUsrSeq == null ? '미연결' : `#${order.accountUsrSeq}`}
                    </Badge>
                  </td>
                  <td className="px-3 py-3 text-center">
                    <Button variant="ghost" size="sm" aria-label={`${order.donor.name} 주문 수정`} onClick={() => setEditingOrder(order)}>
                      <Pencil className="mr-1 h-4 w-4" />수정
                    </Button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <Pagination
        page={pagination.page}
        totalPages={totalPages}
        onPageChange={pagination.setPage}
        pageSize={pagination.pageSize}
        onPageSizeChange={pagination.handlePageSizeChange}
      />

      <DonationOrderForm open={isCreating} onOpenChange={setIsCreating} />
      <DonationOrderForm
        open={editingOrder !== null}
        order={editingOrder ?? undefined}
        onOpenChange={(open) => { if (!open) setEditingOrder(null); }}
      />
    </section>
  );
}
