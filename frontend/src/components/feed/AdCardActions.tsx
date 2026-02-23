// AdCardActions — Controlled action buttons row for ad cards: like, comment count
import { MessageCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import { HeartButton } from './HeartButton';

interface AdCardActionsProps {
  liked: boolean;
  likeCnt: number;
  showComments: boolean;
  commentCnt: number;
  onLikeToggle: () => void;
  onCommentToggle: () => void;
}

export function AdCardActions({
  liked,
  likeCnt,
  showComments,
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
        className={cn(
          'inline-flex items-center gap-1 ml-4 transition-colors',
          showComments ? 'text-primary' : 'hover:text-text-secondary',
        )}
      >
        <MessageCircle size={13} className={cn(showComments && 'fill-current')} />
        {commentCnt}
      </button>
    </div>
  );
}
