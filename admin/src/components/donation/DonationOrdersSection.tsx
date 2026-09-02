// DonationOrdersSection — responsive donation order search, list, and edit entry points
import { CalendarDays, ChevronRight, CreditCard, Pencil, Plus, RotateCcw } from 'lucide-react';
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

interface DonationOrderFiltersPanelProps {
  filters: DonationOrderFilters;
  onFilterChange: <Key extends keyof DonationOrderFilters>(
    name: Key,
    value: DonationOrderFilters[Key],
  ) => void;
  onReset: () => void;
}

function DonationOrderFiltersPanel({
  filters,
  onFilterChange,
  onReset,
}: DonationOrderFiltersPanelProps) {
  return (
    <div className="mb-5 grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-7">
      <Input
        aria-label="기부자명 검색"
        placeholder="기부자명"
        value={filters.name}
        onChange={(event) => onFilterChange('name', event.target.value)}
      />
      <Input
        aria-label="전화번호 검색"
        placeholder="전화번호"
        value={filters.phone}
        onChange={(event) => onFilterChange('phone', event.target.value)}
      />
      <Input
        aria-label="거래번호 검색"
        placeholder="거래번호"
        value={filters.transactionNumber}
        onChange={(event) => onFilterChange('transactionNumber', event.target.value)}
      />
      <Select
        aria-label="유입 경로 검색"
        value={filters.source}
        onChange={(event) => onFilterChange('source', event.target.value as DonationSource | '')}
      >
        <option value="">전체 경로</option>
        <option value="happy_nanum">해피나눔</option>
        <option value="bank_transfer">계좌이체</option>
        <option value="other">기타</option>
      </Select>
      <Select
        aria-label="주문 상태 검색"
        value={filters.status}
        onChange={(event) => onFilterChange('status', event.target.value as DonationStatus | '')}
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
        onChange={(event) => onFilterChange('donationType', event.target.value as DonationType | '')}
      >
        <option value="">전체 유형</option>
        <option value="recurring">정기기부</option>
        <option value="one_time">일시기부</option>
        <option value="sponsorship">후원</option>
      </Select>
      <Button variant="outline" onClick={onReset}>
        <RotateCcw className="mr-2 h-4 w-4" />초기화
      </Button>
    </div>
  );
}

function DonationOrdersTable({
  orders,
  onOpen,
}: {
  orders: DonationOrder[];
  onOpen: (order: DonationOrder) => void;
}) {
  return (
    <div className="hidden overflow-x-auto rounded-xl border border-border-light md:block">
      <table className="w-full min-w-[840px] table-fixed text-sm">
        <thead>
          <tr className="border-b border-border-light text-left text-cool-gray">
            <th className="w-[24%] px-4 py-3 font-medium">기부자</th>
            <th className="w-[18%] px-4 py-3 text-right font-medium">금액</th>
            <th className="w-[14%] px-4 py-3 text-center font-medium">구분</th>
            <th className="w-[11%] px-4 py-3 text-center font-medium">상태</th>
            <th className="w-[13%] px-4 py-3 font-medium">일시</th>
            <th className="w-[12%] px-4 py-3 text-center font-medium">결제수단</th>
            <th className="w-[8%] px-4 py-3 text-center font-medium">상세</th>
          </tr>
        </thead>
        <tbody aria-live="polite">
          {orders.map((order) => {
            const status = STATUS_DETAILS[order.status];
            return (
              <tr
                key={order.orderSeq}
                className="border-b border-border-light last:border-b-0 hover:bg-background"
              >
                <td className="px-4 py-3">
                  <p className="font-medium text-dark-slate">{order.donor.name}</p>
                  <p className="mt-0.5 truncate text-xs text-cool-gray">
                    {order.donor.cohort} / {order.donor.department}
                  </p>
                  <p className="mt-0.5 text-xs text-cool-gray">{formatPhone(order.donor.phone)}</p>
                </td>
                <td className="px-4 py-3 text-right tabular-nums">
                  <p className="font-semibold text-dark-slate">₩{formatAmount(order.grossAmount)}</p>
                  <p className="mt-0.5 text-xs text-cool-gray">
                    실수령 ₩{formatAmount(order.netReceivedAmount)}
                  </p>
                  {order.refundedAmount > 0 && (
                    <p className="mt-0.5 text-xs text-error-text">
                      환불 ₩{formatAmount(order.refundedAmount)}
                    </p>
                  )}
                </td>
                <td className="px-4 py-3 text-center">
                  <Badge>{DONATION_TYPE_LABELS[order.donationType]}</Badge>
                  <p className="mt-1 text-xs text-cool-gray">{SOURCE_LABELS[order.source]}</p>
                </td>
                <td className="px-4 py-3 text-center">
                  <Badge variant={status.variant}>{status.label}</Badge>
                </td>
                <td className="px-4 py-3 text-dark-slate">{order.donationDate}</td>
                <td className="px-4 py-3 text-center text-cool-gray">
                  {PAYMENT_METHOD_LABELS[order.paymentMethod]}
                </td>
                <td className="px-4 py-3 text-center">
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`${order.donor.name} 주문 상세`}
                    onClick={() => onOpen(order)}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function DonationOrdersMobileList({
  orders,
  onOpen,
}: {
  orders: DonationOrder[];
  onOpen: (order: DonationOrder) => void;
}) {
  return (
    <div className="space-y-3 md:hidden" aria-live="polite">
      {orders.map((order) => {
        const status = STATUS_DETAILS[order.status];
        return (
          <button
            key={order.orderSeq}
            type="button"
            aria-label={`${order.donor.name} 기부 주문 상세 열기`}
            className="w-full rounded-2xl border border-border-light bg-white p-4 text-left shadow-sm transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-royal-indigo focus-visible:ring-offset-2"
            onClick={() => onOpen(order)}
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate font-semibold text-dark-slate">{order.donor.name}</p>
                <p className="mt-1 truncate text-xs text-cool-gray">
                  {order.donor.cohort} / {order.donor.department}
                </p>
              </div>
              <div className="flex shrink-0 items-start gap-2">
                <p className="text-base font-bold tabular-nums text-dark-slate">
                  ₩{formatAmount(order.grossAmount)}
                </p>
                <ChevronRight className="mt-0.5 h-5 w-5 text-cool-gray" />
              </div>
            </div>

            <div className="mt-3 flex flex-wrap gap-2">
              <Badge variant={status.variant}>{status.label}</Badge>
              <Badge>{DONATION_TYPE_LABELS[order.donationType]}</Badge>
            </div>

            <div className="mt-4 grid grid-cols-2 gap-3 border-t border-border-light pt-3 text-xs text-cool-gray">
              <span className="flex items-center gap-1.5">
                <CalendarDays className="h-3.5 w-3.5" />
                {order.donationDate}
              </span>
              <span className="flex items-center justify-end gap-1.5 text-right">
                <CreditCard className="h-3.5 w-3.5" />
                {PAYMENT_METHOD_LABELS[order.paymentMethod]}
              </span>
            </div>
          </button>
        );
      })}
    </div>
  );
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
    <section className="rounded-2xl border border-border-light bg-white p-4 shadow-sm md:p-6">
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-semibold text-dark-slate">후원 내역</h3>
            {!query.isLoading && !query.isError && (
              <Badge variant="muted">총 {(query.data?.total ?? 0).toLocaleString()}건</Badge>
            )}
          </div>
          <p className="mt-1 text-sm text-cool-gray">
            단건 기부를 등록하고 주문 정보를 조회·수정합니다.
          </p>
        </div>
        <Button className="w-full sm:w-auto" onClick={() => setIsCreating(true)}>
          <Plus className="mr-2 h-4 w-4" />새 기부 등록
        </Button>
      </div>

      <DonationOrderFiltersPanel
        filters={filters}
        onFilterChange={handleFilterChange}
        onReset={resetFilters}
      />

      {query.isError ? (
        <div className="rounded-xl border border-border-light">
          <ErrorState onRetry={() => void query.refetch()} />
        </div>
      ) : query.isLoading ? (
        <div className="rounded-xl border border-border-light px-3 py-10 text-center text-sm text-cool-gray">
          로딩 중...
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-xl border border-border-light px-3 py-10 text-center text-sm text-cool-gray">
          조건에 맞는 주문이 없습니다.
        </div>
      ) : (
        <>
          <DonationOrdersTable orders={items} onOpen={setEditingOrder} />
          <DonationOrdersMobileList orders={items} onOpen={setEditingOrder} />
        </>
      )}

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
