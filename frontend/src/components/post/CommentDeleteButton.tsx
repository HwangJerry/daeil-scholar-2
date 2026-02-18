// CommentDeleteButton — Soft-delete button for own comments
import { Trash2 } from 'lucide-react';
import { useDeleteComment } from '../../hooks/useComments';

interface CommentDeleteButtonProps {
  seq: number;
  bcSeq: number;
}

export function CommentDeleteButton({ seq, bcSeq }: CommentDeleteButtonProps) {
  const { mutate, isPending } = useDeleteComment(seq);

  return (
    <button
      type="button"
      onClick={() => mutate(bcSeq)}
      disabled={isPending}
      className="shrink-0 p-1 text-text-tertiary transition-colors hover:text-red-500"
      aria-label="댓글 삭제"
    >
      <Trash2 className="h-3.5 w-3.5" />
    </button>
  );
}
