// CommentDeleteButton — Soft-delete button for own comments with inline error feedback
import { Trash2 } from 'lucide-react';
import { useDeleteComment } from '../../hooks/useComments';

interface CommentDeleteButtonProps {
  seq: number;
  bcSeq: number;
}

export function CommentDeleteButton({ seq, bcSeq }: CommentDeleteButtonProps) {
  const { mutate, isPending, isError } = useDeleteComment(seq);

  return (
    <div className="shrink-0">
      <button
        type="button"
        onClick={() => mutate(bcSeq)}
        disabled={isPending}
        className={`p-1 transition-colors ${isError ? 'text-error-text' : 'text-text-tertiary hover:text-error-text'}`}
        aria-label="댓글 삭제"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
      {isError && (
        <p className="text-xs text-error-text">삭제할 수 없습니다</p>
      )}
    </div>
  );
}
