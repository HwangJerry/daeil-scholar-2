// AdCommentDeleteButton — Delete button for own ad comments with inline error feedback
import { Trash2 } from 'lucide-react';
import { useDeleteAdComment } from '../../hooks/useAdComments';

interface AdCommentDeleteButtonProps {
  maSeq: number;
  acSeq: number;
}

export function AdCommentDeleteButton({ maSeq, acSeq }: AdCommentDeleteButtonProps) {
  const { mutate, isPending, isError } = useDeleteAdComment(maSeq);

  return (
    <div className="shrink-0">
      <button
        type="button"
        onClick={() => mutate(acSeq)}
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
