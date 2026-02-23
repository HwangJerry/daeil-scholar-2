// NoticeCardEngagement — Like toggle and comment panel for NoticeCard
import { useState } from 'react';
import { useNoticeLikeToggle } from '../../hooks/useNoticeLikeToggle';
import { FeedEngagementBar } from './FeedEngagementBar';
import { InlineFeedComments } from './InlineFeedComments';

interface NoticeCardEngagementProps {
  seq: number;
  likeCnt: number;
  commentCnt: number;
  hit: number;
}

export function NoticeCardEngagement({
  seq,
  likeCnt,
  commentCnt,
  hit,
}: NoticeCardEngagementProps) {
  const [isCommentOpen, setIsCommentOpen] = useState(false);
  const { liked, likeCnt: currentLikeCnt, toggle } = useNoticeLikeToggle(seq, likeCnt);

  return (
    <>
      <FeedEngagementBar
        liked={liked}
        onLikeToggle={toggle}
        likeCnt={currentLikeCnt}
        commentCnt={commentCnt}
        hit={hit}
        isCommentOpen={isCommentOpen}
        onCommentToggle={() => setIsCommentOpen((prev) => !prev)}
      />
      {isCommentOpen && (
        <InlineFeedComments seq={seq} commentCnt={commentCnt} />
      )}
    </>
  );
}
