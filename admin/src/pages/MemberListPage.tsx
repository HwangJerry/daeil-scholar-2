// MemberListPage — searchable, paginated member table with status filter
import { Link } from 'react-router-dom';
import { Input } from '../components/ui/Input.tsx';
import { Select } from '../components/ui/Select.tsx';
import { Pagination } from '../components/ui/Pagination.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { useMemberList } from '../hooks/useMemberList.ts';

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

export function MemberListPage() {
  const { data, isLoading, page, search, statusFilter, setPage, handleSearchChange, handleStatusChange } = useMemberList();

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-dark-slate">회원 관리</h2>

      <div className="flex gap-2">
        <Input
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

      <div className="overflow-x-auto rounded-2xl border border-slate-100 bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-100 text-left text-cool-gray">
              <th className="px-4 py-3 font-medium">이름</th>
              <th className="px-4 py-3 font-medium w-16">기수</th>
              <th className="px-4 py-3 font-medium w-20 text-center">상태</th>
              <th className="px-4 py-3 font-medium w-36">연락처</th>
              <th className="px-4 py-3 font-medium w-28">최근 접속</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : data?.items.length ? (
              data.items.map((m) => (
                <tr key={m.usrSeq} className="border-b border-slate-50 hover:bg-slate-50">
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

      {data && data.total > 20 && (
        <Pagination page={page} totalPages={Math.ceil(data.total / 20)} onPageChange={setPage} />
      )}
    </div>
  );
}
