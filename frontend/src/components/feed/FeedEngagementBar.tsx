// FeedEngagementBar — Interactive engagement footer for feed cards with like toggle and comment toggle
import { Eye, MessageCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import { HeartButton } from './HeartButton';

interface FeedEngagementBarProps {
  likeCnt: number;
  commentCnt: number;
  hit: number;
  isCommentOpen: boolean;
  onCommentToggle: () => void;
  liked: boolean;
  onLikeToggle: () => void;
}

export function FeedEngagementBar({
  likeCnt,
  commentCnt,
  hit,
  isCommentOpen,
  onCommentToggle,
  liked,
  onLikeToggle,
}: FeedEngagementBarProps) {
  return (
    <div className="flex items-center px-5 py-2.5 text-xs text-text-placeholder">
      <HeartButton liked={liked} onToggle={onLikeToggle} count={likeCnt} />

      <button
        type="button"
        onClick={onCommentToggle}
        aria-expanded={isCommentOpen}
        aria-label="댓글 펼치기"
        className={cn(
          'inline-flex items-center gap-1 ml-4 transition-colors',
          isCommentOpen ? 'text-primary' : 'hover:text-text-secondary',
        )}
      >
        <MessageCircle
          size={13}
          className={cn(isCommentOpen && 'fill-current')}
        />
        {commentCnt}
      </button>

      <span className="inline-flex items-center gap-1 ml-auto">
        <Eye size={13} />
        {hit}
      </span>
    </div>
  );
}
