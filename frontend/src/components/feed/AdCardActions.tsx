// AdCardActions — Controlled action buttons row for ad cards: like, comment count
import { MessageCircle } from 'lucide-react';
import { HeartButton } from './HeartButton';

interface AdCardActionsProps {
  liked: boolean;
  likeCnt: number;
  commentCnt: number;
  onLikeToggle: () => void;
  onCommentToggle: () => void;
}

export function AdCardActions({
  liked,
  likeCnt,
  commentCnt,
  onLikeToggle,
  onCommentToggle,
}: AdCardActionsProps) {
  return (
    <div className="flex items-center px-5 py-2.5 text-xs text-text-placeholder">
      <HeartButton liked={liked} onToggle={onLikeToggle} count={likeCnt} />
      <button
        type="button"
        onClick={onCommentToggle}
        className="inline-flex items-center gap-1 ml-4 transition-colors hover:text-text-secondary"
      >
        <MessageCircle size={13} />
        {commentCnt}
      </button>
    </div>
  );
}
