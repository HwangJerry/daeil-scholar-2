// Pagination — page navigation with numbered buttons, prev/next controls, and optional page size selector
import { cn } from "../../lib/utils.ts";

const PAGE_SIZE_OPTIONS = [10, 20, 50];

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  pageSize?: number;
  onPageSizeChange?: (size: number) => void;
}

const MAX_VISIBLE = 5;

export function Pagination({ page, totalPages, onPageChange, pageSize, onPageSizeChange }: PaginationProps) {
  if (totalPages <= 1 && !onPageSizeChange) return null;

  const half = Math.floor(MAX_VISIBLE / 2);
  let start = Math.max(1, page - half);
  const end = Math.min(totalPages, start + MAX_VISIBLE - 1);
  if (end - start + 1 < MAX_VISIBLE) {
    start = Math.max(1, end - MAX_VISIBLE + 1);
  }

  const pages: number[] = [];
  for (let i = start; i <= end; i++) pages.push(i);

  return (
    <nav aria-label="페이지 네비게이션" className="flex items-center justify-center gap-1 mt-6">
      {onPageSizeChange && pageSize != null && (
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          className="mr-4 h-9 rounded-xl border border-border bg-white px-2 text-sm text-cool-gray focus:outline-none focus:ring-2 focus:ring-royal-indigo"
          aria-label="페이지당 항목 수"
        >
          {PAGE_SIZE_OPTIONS.map((size) => (
            <option key={size} value={size}>{size}개씩</option>
          ))}
        </select>
      )}
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        aria-label="이전 페이지"
        className="px-3 py-2 text-sm rounded-lg hover:bg-border-light disabled:opacity-40 disabled:pointer-events-none"
      >
        이전
      </button>
      {pages.map((p) => (
        <button
          key={p}
          onClick={() => onPageChange(p)}
          aria-current={p === page ? 'page' : undefined}
          className={cn(
            "px-3 py-2 text-sm rounded-lg",
            p === page
              ? "bg-royal-indigo text-white"
              : "hover:bg-border-light text-cool-gray",
          )}
        >
          {p}
        </button>
      ))}
      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        aria-label="다음 페이지"
        className="px-3 py-2 text-sm rounded-lg hover:bg-border-light disabled:opacity-40 disabled:pointer-events-none"
      >
        다음
      </button>
    </nav>
  );
}
