// CommentSection — Comment list display with auth-aware input and per-item delete
import { useAuth } from '../../hooks/useAuth';
import { useComments } from '../../hooks/useComments';
import { formatRelativeDate } from '../../utils/date';
import { CommentInput } from './CommentInput';
import { CommentDeleteButton } from './CommentDeleteButton';

interface CommentSectionProps {
  seq: number;
}

export function CommentSection({ seq }: CommentSectionProps) {
  const { user, isLoggedIn } = useAuth();
  const { data: comments, isLoading } = useComments(seq);

  return (
    <section className="mt-6 border-t border-border-subtle pt-5">
      <h3 className="mb-3 text-sm font-semibold text-text-primary">
        댓글 {comments?.length ? `(${comments.length})` : ''}
      </h3>

      {isLoggedIn ? (
        <CommentInput seq={seq} />
      ) : (
        <p className="mb-4 rounded-lg bg-background-secondary px-3 py-2.5 text-xs text-text-tertiary">
          댓글을 작성하려면 로그인이 필요합니다.
        </p>
      )}

      {isLoading && (
        <div className="space-y-2">
          <div className="h-12 rounded skeleton-shimmer" />
          <div className="h-12 rounded skeleton-shimmer" />
        </div>
      )}

      {!isLoading && comments?.length === 0 && (
        <p className="py-4 text-center text-xs text-text-tertiary">
          아직 댓글이 없습니다.
        </p>
      )}

      {comments && comments.length > 0 && (
        <ul className="space-y-3">
          {comments.map((c) => (
            <li key={c.bcSeq} className="flex items-start gap-2 text-sm">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-text-primary">{c.regName}</span>
                  <span className="text-xs text-text-tertiary">
                    {formatRelativeDate(c.regDate)}
                  </span>
                </div>
                <p className="mt-0.5 whitespace-pre-wrap break-words text-text-secondary">
                  {c.contents}
                </p>
              </div>
              {user && user.usrSeq === c.usrSeq && (
                <CommentDeleteButton seq={seq} bcSeq={c.bcSeq} />
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
