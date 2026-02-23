// InlineFeedComments — Inline comment preview panel for feed cards (LinkedIn style)
import { Link } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { useComments } from '../../hooks/useComments';
import { formatRelativeDate } from '../../utils/date';
import { CommentInput } from '../post/CommentInput';

const MAX_PREVIEW_COMMENTS = 2;

interface InlineFeedCommentsProps {
  seq: number;
  commentCnt: number;
}

export function InlineFeedComments({ seq, commentCnt }: InlineFeedCommentsProps) {
  const { isLoggedIn } = useAuth();
  const { data: comments, isLoading } = useComments(seq);

  const previewComments = (comments ?? []).slice(0, MAX_PREVIEW_COMMENTS);
  const moreCount = Math.max(0, (comments?.length ?? commentCnt) - MAX_PREVIEW_COMMENTS);

  return (
    <div className="border-t border-border-subtle animate-fade-in-up">
      <div className="px-5 pb-4 pt-3">
        {isLoggedIn ? (
          <CommentInput seq={seq} autoFocus />
        ) : (
          <p className="mb-3 text-xs text-text-tertiary">
            로그인 후 댓글을 작성하세요.
          </p>
        )}

        {isLoading && (
          <div className="mt-2 space-y-2">
            <div className="h-10 rounded skeleton-shimmer" />
            <div className="h-10 rounded skeleton-shimmer" />
          </div>
        )}

        {previewComments.length > 0 && (
          <ul className="space-y-3">
            {previewComments.map((c) => (
              <li key={c.bcSeq} className="text-sm">
                <div className="flex items-center gap-1.5">
                  <span className="font-medium text-text-primary">{c.regName}</span>
                  <span className="text-xs text-text-placeholder">·</span>
                  <span className="text-xs text-text-tertiary">
                    {formatRelativeDate(c.regDate)}
                  </span>
                </div>
                <p className="mt-0.5 whitespace-pre-wrap break-words text-body-sm text-text-secondary">
                  {c.contents}
                </p>
              </li>
            ))}
          </ul>
        )}

        {moreCount > 0 && (
          <Link
            to={`/post/${seq}`}
            className="mt-3 inline-flex text-xs text-text-placeholder hover:text-primary transition-colors"
          >
            댓글 {moreCount}개 더 보기 →
          </Link>
        )}
      </div>
    </div>
  );
}
