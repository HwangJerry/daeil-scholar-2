// NoticeListPage — paginated notice table with search, pin, delete, sort, and error handling
import { Link } from 'react-router-dom';
import { Plus, Pin, Trash2 } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { Pagination } from '../components/ui/Pagination.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { ConfirmDialog } from '../components/ui/ConfirmDialog.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { SortableHeader } from '../components/ui/SortableHeader.tsx';
import { useAdminNoticeList } from '../hooks/useAdminNoticeList.ts';
import { useNoticeListActions } from '../hooks/useNoticeListActions.ts';
import { useConfirmDialog } from '../hooks/useConfirmDialog.ts';
import { useTableSort } from '../hooks/useTableSort.ts';
import type { AdminNoticeListItem } from '../types/api.ts';

const SORT_ACCESSORS: Record<string, (item: AdminNoticeListItem) => string | number | null> = {
  subject: (n) => n.subject,
  regDate: (n) => n.regDate,
  hit: (n) => n.hit,
};

export function NoticeListPage() {
  const { data, isLoading, isError, refetch, page, pageSize, search, setPage, handleSearchChange, handlePageSizeChange } = useAdminNoticeList();
  const { togglePin, deleteNotice } = useNoticeListActions();
  const deleteDialog = useConfirmDialog<number>();
  const { sort, toggleSort, getSortedItems } = useTableSort();

  const handleDeleteConfirm = () => {
    const seq = deleteDialog.confirm();
    if (seq != null) deleteNotice(seq);
  };

  const items = data?.items ? getSortedItems(data.items, SORT_ACCESSORS) : [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-dark-slate">공지 관리</h2>
        <Link to="/notice/new">
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
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-light text-left text-cool-gray">
              <SortableHeader label="제목" column="subject" sort={sort} onToggle={toggleSort} className="px-4 py-3" />
              <SortableHeader label="작성일" column="regDate" sort={sort} onToggle={toggleSort} className="px-4 py-3 w-24" />
              <SortableHeader label="조회" column="hit" sort={sort} onToggle={toggleSort} className="px-4 py-3 w-16 text-center" />
              <th className="px-4 py-3 font-medium w-20 text-center">포맷</th>
              <th className="px-4 py-3 font-medium w-28 text-center">작업</th>
            </tr>
          </thead>
          <tbody aria-live="polite">
            {isError ? (
              <ErrorState colSpan={5} onRetry={() => void refetch()} />
            ) : isLoading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : items.length ? (
              items.map((n) => (
                <tr key={n.seq} className="border-b border-border-light hover:bg-background">
                  <td className="px-4 py-3">
                    <Link to={`/notice/${n.seq}/edit`} className="text-dark-slate hover:text-royal-indigo">
                      {n.openYn === 'N' && <Badge variant="muted" className="mr-1">삭제됨</Badge>}
                      {n.isPinned === 'Y' && <Pin className="mr-1 inline h-3.5 w-3.5 text-error-text" />}
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
                  <td className="px-4 py-3 text-center">
                    <div className="flex items-center justify-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => togglePin(n.seq)}
                        aria-label={n.isPinned === 'Y' ? '고정 해제' : '상단 고정'}
                      >
                        <Pin className={`h-4 w-4 ${n.isPinned === 'Y' ? 'text-error-text' : 'text-cool-gray'}`} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteDialog.open(n.seq)}
                        aria-label="공지 삭제"
                      >
                        <Trash2 className="h-4 w-4 text-cool-gray hover:text-error-text" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">공지가 없습니다.</td></tr>
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

      <ConfirmDialog
        open={deleteDialog.isOpen}
        onOpenChange={(open) => { if (!open) deleteDialog.close(); }}
        title="공지 삭제"
        description="정말 삭제하시겠습니까?"
        confirmLabel="삭제"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
      />
    </div>
  );
}
