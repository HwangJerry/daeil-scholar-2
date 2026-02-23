// AdCommentList — Fetches and renders the ad comment list with per-item delete
import { useAuth } from '../../hooks/useAuth';
import { useAdComments } from '../../hooks/useAdComments';
import { formatRelativeDate } from '../../utils/date';
import { AdCommentDeleteButton } from './AdCommentDeleteButton';

interface AdCommentListProps {
  maSeq: number;
}

export function AdCommentList({ maSeq }: AdCommentListProps) {
  const { user } = useAuth();
  const { data: comments, isLoading } = useAdComments(maSeq);

  return (
    <>
      <h3 className="mb-3 text-sm font-semibold text-text-primary">
        댓글 {comments?.length ? `(${comments.length})` : ''}
      </h3>

      {isLoading && (
        <div className="space-y-2">
          <div className="h-12 rounded skeleton-shimmer" />
          <div className="h-12 rounded skeleton-shimmer" />
        </div>
      )}

      {!isLoading && !comments?.length && (
        <p className="py-4 text-center text-xs text-text-tertiary">아직 댓글이 없습니다.</p>
      )}

      {comments && comments.length > 0 && (
        <ul className="space-y-3">
          {comments.map((c) => (
            <li key={c.acSeq} className="flex items-start gap-2 text-sm">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-text-primary">{c.nickname}</span>
                  <span className="text-xs text-text-tertiary">{formatRelativeDate(c.regDate)}</span>
                </div>
                <p className="mt-0.5 whitespace-pre-wrap break-words text-text-secondary">
                  {c.contents}
                </p>
              </div>
              {user && user.usrSeq === c.usrSeq && (
                <AdCommentDeleteButton maSeq={maSeq} acSeq={c.acSeq} />
              )}
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
