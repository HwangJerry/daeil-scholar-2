// PostNavigation — 이전/다음글 네비게이션
import { Link } from 'react-router-dom';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { usePostSiblings } from '../../hooks/usePostSiblings';

interface PostNavigationProps {
  currentSeq: number;
}

export function PostNavigation({ currentSeq }: PostNavigationProps) {
  const { data: siblings } = usePostSiblings(currentSeq);

  return (
    <nav className="border-t border-border-subtle">
      <div className="divide-y divide-border-subtle">
        {siblings?.prev && (
          <Link
            to={`/post/${siblings.prev.seq}`}
            className="flex items-center gap-2 px-5 py-3 text-sm text-text-secondary hover:bg-background-secondary transition-colors"
          >
            <ChevronLeft className="h-4 w-4 shrink-0 text-text-placeholder" />
            <span className="shrink-0 text-text-placeholder">이전글</span>
            <span className="truncate">{siblings.prev.subject}</span>
          </Link>
        )}
        {siblings?.next && (
          <Link
            to={`/post/${siblings.next.seq}`}
            className="flex items-center gap-2 px-5 py-3 text-sm text-text-secondary hover:bg-background-secondary transition-colors"
          >
            <ChevronRight className="h-4 w-4 shrink-0 text-text-placeholder" />
            <span className="shrink-0 text-text-placeholder">다음글</span>
            <span className="truncate">{siblings.next.subject}</span>
          </Link>
        )}
      </div>
    </nav>
  );
}
