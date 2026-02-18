// NoticeListPage — paginated notice table with search, pin toggle, and delete actions
import { Link } from 'react-router-dom';
import { Plus, Pin, Trash2 } from 'lucide-react';
import { Button } from '../components/ui/Button.tsx';
import { Input } from '../components/ui/Input.tsx';
import { Pagination } from '../components/ui/Pagination.tsx';
import { Badge } from '../components/ui/Badge.tsx';
import { useAdminNoticeList } from '../hooks/useAdminNoticeList.ts';
import { useNoticeListActions } from '../hooks/useNoticeListActions.ts';

export function NoticeListPage() {
  const { data, isLoading, page, search, setPage, handleSearchChange } = useAdminNoticeList();
  const { togglePin, deleteNotice } = useNoticeListActions();

  const handleDelete = (seq: number) => {
    if (window.confirm('정말 삭제하시겠습니까?')) {
      deleteNotice(seq);
    }
  };

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
          placeholder="제목 검색..."
          value={search}
          onChange={(e) => handleSearchChange(e.target.value)}
          className="max-w-xs"
        />
      </div>

      <div className="overflow-x-auto rounded-2xl border border-slate-100 bg-white shadow-sm">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-100 text-left text-cool-gray">
              <th className="px-4 py-3 font-medium">제목</th>
              <th className="px-4 py-3 font-medium w-24">작성일</th>
              <th className="px-4 py-3 font-medium w-16 text-center">조회</th>
              <th className="px-4 py-3 font-medium w-20 text-center">포맷</th>
              <th className="px-4 py-3 font-medium w-28 text-center">작업</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-cool-gray">로딩 중...</td></tr>
            ) : data?.items.length ? (
              data.items.map((n) => (
                <tr key={n.seq} className="border-b border-slate-50 hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <Link to={`/notice/${n.seq}/edit`} className="text-dark-slate hover:text-royal-indigo">
                      {n.isPinned === 'Y' && <Pin className="mr-1 inline h-3.5 w-3.5 text-red-500" />}
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
                        title={n.isPinned === 'Y' ? '고정 해제' : '상단 고정'}
                      >
                        <Pin className={`h-4 w-4 ${n.isPinned === 'Y' ? 'text-red-500' : 'text-cool-gray'}`} />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => handleDelete(n.seq)} title="삭제">
                        <Trash2 className="h-4 w-4 text-cool-gray hover:text-red-500" />
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

      {data && data.total > 20 && (
        <Pagination page={page} totalPages={Math.ceil(data.total / 20)} onPageChange={setPage} />
      )}
    </div>
  );
}
