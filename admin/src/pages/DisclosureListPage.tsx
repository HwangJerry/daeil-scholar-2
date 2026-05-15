// DisclosureListPage — paginated public-disclosure table with search and sort
import { Link } from 'react-router-dom';
import { Plus } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { Pagination } from '../components/ui/Pagination.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { SortableHeader } from '../components/ui/SortableHeader.tsx';
import { useAdminDisclosureList } from '../hooks/useAdminDisclosureList.ts';
import { useTableSort } from '../hooks/useTableSort.ts';
import type { AdminDisclosureListItem } from '../types/api.ts';

const SORT_ACCESSORS: Record<string, (item: AdminDisclosureListItem) => string | number | null> = {
  subject: (n) => n.subject,
  regDate: (n) => n.regDate,
  hit: (n) => n.hit,
};

export function DisclosureListPage() {
  const { data, isLoading, isError, refetch, page, pageSize, search, setPage, handleSearchChange, handlePageSizeChange } = useAdminDisclosureList();
  const { sort, toggleSort, getSortedItems } = useTableSort();

  const items = data?.items ? getSortedItems(data.items, SORT_ACCESSORS) : [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-dark-slate">의무공시 관리</h2>
        <Link to="/disclosure/new">
          <Button size="sm">
            <Plus className="mr-1 h-4 w-4" />
            새 글 작성
          </Button>
        </Link>
      </div>

      <div className="flex gap-2">
        <Input
          aria-label="제목 검색"
          placeholder="제목 검색..."
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          className="max-w-xs"
        />
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border-light bg-white shadow-sm">
        <table className="w-full table-fixed text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <SortableHeader label="제목" column="subject" sort={sort} onToggle={toggleSort} className="px-4 py-3" />
              <SortableHeader label="작성일" column="regDate" sort={sort} onToggle={toggleSort} className="px-4 py-3 w-28" />
              <SortableHeader label="조회" column="hit" sort={sort} onToggle={toggleSort} className="px-4 py-3 w-20 text-center" />
              <th className="px-4 py-3 font-medium w-20 text-center">포맷</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={4} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : items.length ? (
              items.map((n) => (
                <tr key={n.seq} className="border-b border-border-light hover:bg-background">
                  <td className="px-4 py-3 min-w-0">
                    <Link to={`/disclosure/${n.seq}/edit`} className="block truncate text-dark-slate hover:text-royal-indigo">
                      {n.openYn === 'N' && <Badge variant="muted" className="mr-1">삭제됨</Badge>}
                      {n.subject}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-cool-gray">{n.regDate.slice(0, 10)}</td>
                  <td className="px-4 py-3 text-center text-cool-gray">{n.hit}</td>
                  <td className="px-4 py-3 text-center">
                    <Badge variant={n.contentFormat === 'MARKDOWN' ? 'default' : 'muted'}>
                      {n.contentFormat === 'MARKDOWN' ? 'MD' : 'HTML'}
                    </Badge>
                  </td>
                </tr>
              ))
            ) : (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-cool-gray">공시 자료가 없습니다.</td></tr>
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
