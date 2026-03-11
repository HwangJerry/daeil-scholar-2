// NoticeCardEngagement — Like toggle, comment panel, and view count for NoticeCard
import { useState } from 'react';
import { Eye, MessageCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useNoticeLikeToggle } from '../../hooks/useNoticeLikeToggle';
import { useAuthRedirect } from '../../hooks/useAuthRedirect';
import { FeedCard } from './FeedCard';
import { HeartButton } from './HeartButton';
import { InlineFeedComments } from './InlineFeedComments';

interface NoticeCardEngagementProps {
  seq: number;
  likeCnt: number;
  commentCnt: number;
  hit: number;
  userLiked: boolean;
}

export function NoticeCardEngagement({
  seq,
  likeCnt,
  commentCnt,
  hit,
  userLiked,
}: NoticeCardEngagementProps) {
  const [isCommentOpen, setIsCommentOpen] = useState(false);
  const onAuthError = useAuthRedirect();
  const { liked, likeCnt: currentLikeCnt, toggle } = useNoticeLikeToggle(seq, likeCnt, userLiked, { onAuthError });

  return (
    <>
      <FeedCard.Footer>
        <HeartButton liked={liked} onToggle={toggle} count={currentLikeCnt} />

        <button
          type="button"
          onClick={() => setIsCommentOpen((prev) => !prev)}
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
      </FeedCard.Footer>

      {isCommentOpen && (
        <InlineFeedComments seq={seq} commentCnt={commentCnt} />
      )}
    </>
  );
}
