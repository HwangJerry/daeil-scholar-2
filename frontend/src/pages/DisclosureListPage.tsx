// DisclosureListPage — 공익법인 의무공시 목록 (단순 표/리스트 + 첨부 강조)
import { Link } from 'react-router-dom';
import { Paperclip, Eye } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import { useDisclosureList } from '../hooks/useDisclosureList';
import type { DisclosureItem } from '../types/api';

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}.${m}.${day}`;
}

function DesktopRow({ item }: { item: DisclosureItem }) {
  return (
    <Link
      to={`/disclosure/${item.seq}`}
      className="grid grid-cols-[1fr_auto_auto_auto] items-center gap-4 px-4 py-3.5 text-sm border-b border-border-subtle last:border-b-0 hover:bg-surface-hover transition-colors"
    >
      <span className="font-medium text-text-primary truncate">{item.subject}</span>
      <span className="text-text-tertiary whitespace-nowrap">{formatDate(item.regDate)}</span>
      <span className="inline-flex items-center gap-1 text-text-tertiary whitespace-nowrap min-w-[3rem] justify-end">
        {item.attachmentCount > 0 && (
          <>
            <Paperclip className="h-3.5 w-3.5" />
            {item.attachmentCount}
          </>
        )}
      </span>
      <span className="inline-flex items-center gap-1 text-text-placeholder whitespace-nowrap">
        <Eye className="h-3.5 w-3.5" />
        {item.hit}
      </span>
    </Link>
  );
}

function MobileRow({ item }: { item: DisclosureItem }) {
  return (
    <Link
      to={`/disclosure/${item.seq}`}
      className="block px-4 py-3 border-b border-border-subtle last:border-b-0 hover:bg-surface-hover transition-colors"
    >
      <h3 className="font-medium text-text-primary text-body-md leading-snug line-clamp-2">{item.subject}</h3>
      <div className="mt-1.5 flex items-center gap-2 text-2xs text-text-tertiary">
        <span>{formatDate(item.regDate)}</span>
        {item.attachmentCount > 0 && (
          <>
            <span className="text-border-subtle">·</span>
            <span className="inline-flex items-center gap-0.5">
              <Paperclip className="h-3 w-3" />
              {item.attachmentCount}
            </span>
          </>
        )}
        <span className="text-border-subtle">·</span>
        <span className="inline-flex items-center gap-0.5">
          <Eye className="h-3 w-3" />
          {item.hit}
        </span>
      </div>
    </Link>
  );
}

export function DisclosureListPage() {
  const { items, hasMore, isFetching, loadMore } = useDisclosureList();

  return (
    <InfoPageShell
      title="의무공시"
      subtitle="공익법인 의무공시 자료를 공개합니다."
      canonicalPath="/disclosure"
    >
      <Card variant="default" padding="none">
        {items.length === 0 && !isFetching ? (
          <p className="px-4 py-12 text-center text-sm text-text-tertiary">
            등록된 공시 자료가 없습니다.
          </p>
        ) : (
          <>
            <div className="hidden md:block">
              <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 border-b border-border-subtle bg-surface-muted px-4 py-2.5 text-2xs font-medium text-text-tertiary">
                <span>제목</span>
                <span className="whitespace-nowrap">작성일</span>
                <span className="whitespace-nowrap min-w-[3rem] text-right">첨부</span>
                <span className="whitespace-nowrap">조회</span>
              </div>
              {items.map((item) => (
                <DesktopRow key={item.seq} item={item} />
              ))}
            </div>
            <div className="md:hidden">
              {items.map((item) => (
                <MobileRow key={item.seq} item={item} />
              ))}
            </div>
          </>
        )}
      </Card>

      {hasMore && (
        <div className="text-center">
          <button
            onClick={() => loadMore()}
            disabled={isFetching}
            className="rounded-lg border border-border-subtle bg-surface px-5 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors disabled:opacity-50"
          >
            {isFetching ? '불러오는 중...' : '더 보기'}
          </button>
        </div>
      )}
    </InfoPageShell>
  );
}
