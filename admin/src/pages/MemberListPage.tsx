// MemberListPage — searchable, paginated member table with sort, error handling, and a11y
import { Link } from 'react-router-dom';
import { Input } from '../components/ui/Input.tsx';
import { Select } from '../components/ui/Select.tsx';
import { Pagination } from '../components/ui/Pagination.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { SortableHeader } from '../components/ui/SortableHeader.tsx';
import { useMemberList } from '../hooks/useMemberList.ts';
import { useTableSort } from '../hooks/useTableSort.ts';
import type { AdminMemberListItem } from '../types/api.ts';

const STATUS_LABELS: Record<string, string> = {
  AAA: '탈퇴',
  ABA: '휴면',
  ACA: '정지',
  BAA: '승인거절',
  BBB: '승인대기',
  CCC: '승인회원',
  ZZZ: '운영자',
};

const STATUS_VARIANT: Record<string, 'success' | 'default' | 'warning' | 'danger' | 'muted'> = {
  AAA: 'danger',
  ABA: 'warning',
  ACA: 'danger',
  BAA: 'danger',
  BBB: 'warning',
  CCC: 'success',
  ZZZ: 'default',
};

const SORT_ACCESSORS: Record<string, (item: AdminMemberListItem) => string | number | null> = {
  usrName: (m) => m.usrName,
  usrFn: (m) => m.usrFn,
  visitDate: (m) => m.visitDate,
};

export function MemberListPage() {
  const { data, isLoading, isError, refetch, page, pageSize, search, statusFilter, setPage, handleSearchChange, handleStatusChange, handlePageSizeChange } = useMemberList();
  const { sort, toggleSort, getSortedItems } = useTableSort();

  const items = data?.items ? getSortedItems(data.items, SORT_ACCESSORS) : [];

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-dark-slate">회원 관리</h2>

      <div className="flex gap-2">
        <Input
          aria-label="이름 검색"
          placeholder="이름 검색..."
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
          <option value="AAA">탈퇴</option>
          <option value="ABA">휴면</option>
          <option value="ACA">정지</option>
          <option value="BAA">승인거절</option>
          <option value="BBB">승인대기</option>
          <option value="CCC">승인회원</option>
          <option value="ZZZ">운영자</option>
        </Select>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <SortableHeader label="이름" column="usrName" sort={sort} onToggle={toggleSort} className="px-4 py-3" />
              <SortableHeader label="기수" column="usrFn" sort={sort} onToggle={toggleSort} className="px-4 py-3 w-16" />
              <th className="px-4 py-3 font-medium w-20 text-center">상태</th>
              <th className="px-4 py-3 font-medium w-36">연락처</th>
              <SortableHeader label="최근 접속" column="visitDate" sort={sort} onToggle={toggleSort} className="px-4 py-3 w-28" />
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={5} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : items.length ? (
              items.map((m) => (
                <tr key={m.usrSeq} className="border-b border-border-light hover:bg-background">
                  <td className="px-4 py-3">
                    <Link to={`/member/${m.usrSeq}`} className="text-dark-slate hover:text-royal-indigo">
                      {m.usrName}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-cool-gray">{m.usrFn ?? '—'}</td>
                  <td className="px-4 py-3 text-center">
                    <Badge variant={STATUS_VARIANT[m.usrStatus] ?? 'muted'}>
                      {STATUS_LABELS[m.usrStatus] ?? m.usrStatus}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-cool-gray">{m.usrPhone ?? '—'}</td>
                  <td className="px-4 py-3 text-cool-gray">{m.visitDate?.slice(0, 10) ?? '—'}</td>
                </tr>
              ))
            ) : (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">회원이 없습니다.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {data && (
        <Pagination
          page={page}
          totalPages={Math.ceil(data.total / pageSize)}
          onPageChange={setPage}
          pageSize={pageSize}
          onPageSizeChange={handlePageSizeChange}
        />
      )}
    </div>
  );
}
