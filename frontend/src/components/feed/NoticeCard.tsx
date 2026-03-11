// NoticeCard — Feed card with category, body-first layout, and expandable summary
import { Pin } from 'lucide-react';
import { formatAbsoluteDate } from '../../utils/date';
import type { NoticeItem } from '../../types/api';
import { FeedCard } from './FeedCard';
import { NoticeCardEngagement } from './NoticeCardEngagement';
import { NoticeCardLink } from './NoticeCardLink';
import { NoticeCardSummary } from './NoticeCardSummary';
import { NOTICE_CATEGORY_LABELS } from './noticeCard.constants';

export function NoticeCard({ item }: { item: NoticeItem }) {
  const category = item.category ?? 'notice';
  const categoryLabel = NOTICE_CATEGORY_LABELS[category] ?? '공지';

  return (
    <FeedCard>
      <FeedCard.Body hasImage={!!item.thumbnailUrl}>
        <FeedCard.Meta
          action={
            item.isPinned === 'Y' && (
              <Pin size={13} className="text-text-placeholder flex-shrink-0" />
            )
          }
        >
          <span>{categoryLabel}</span>
          <FeedCard.MetaDot />
          <span>{formatAbsoluteDate(item.regDate)}</span>
        </FeedCard.Meta>

        <NoticeCardLink seq={item.seq} className="group block mb-2">
          <FeedCard.Title>{item.subject}</FeedCard.Title>
        </NoticeCardLink>

        {item.summary && <NoticeCardSummary summary={item.summary} />}
      </FeedCard.Body>

      {item.thumbnailUrl && (
        <NoticeCardLink seq={item.seq} className="block">
          <FeedCard.Image src={item.thumbnailUrl} alt={item.subject} />
        </NoticeCardLink>
      )}

      <FeedCard.Divider />

      <NoticeCardEngagement
        seq={item.seq}
        likeCnt={item.likeCnt ?? 0}
        commentCnt={item.commentCnt ?? 0}
        hit={item.hit ?? 0}
        userLiked={item.userLiked ?? false}
      />
    </FeedCard>
  );
}
