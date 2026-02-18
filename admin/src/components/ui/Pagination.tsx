// Page navigation with numbered buttons and prev/next controls
import { cn } from "../../lib/utils.ts";

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

const MAX_VISIBLE = 5;

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null;

  const half = Math.floor(MAX_VISIBLE / 2);
  let start = Math.max(1, page - half);
  const end = Math.min(totalPages, start + MAX_VISIBLE - 1);
  if (end - start + 1 < MAX_VISIBLE) {
    start = Math.max(1, end - MAX_VISIBLE + 1);
  }

  const pages: number[] = [];
  for (let i = start; i <= end; i++) pages.push(i);

  return (
    <nav className="flex items-center justify-center gap-1 mt-6">
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="px-3 py-2 text-sm rounded-lg hover:bg-slate-100 disabled:opacity-40 disabled:pointer-events-none"
      >
        이전
      </button>
      {pages.map((p) => (
        <button
          key={p}
          onClick={() => onPageChange(p)}
          className={cn(
            "px-3 py-2 text-sm rounded-lg",
            p === page
              ? "bg-royal-indigo text-white"
              : "hover:bg-slate-100 text-cool-gray",
          )}
        >
          {p}
        </button>
      ))}
      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="px-3 py-2 text-sm rounded-lg hover:bg-slate-100 disabled:opacity-40 disabled:pointer-events-none"
      >
        다음
      </button>
    </nav>
  );
}
