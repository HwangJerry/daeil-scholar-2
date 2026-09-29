// NoticeCard — Feed card with category, body-first layout, and expandable summary
import { Pin } from 'lucide-react';
import { formatAbsoluteDate } from '../../utils/date';
import type { NoticeItem } from '../../types/api';
import { FeedCard } from './FeedCard';
import { NoticeCardLink } from './NoticeCardLink';
import { NoticeCardSummary } from './NoticeCardSummary';
import { noticeCategoryLabel } from './noticeCategoryLabel';

export function NoticeCard({ item }: { item: NoticeItem }) {
  const categoryLabel = noticeCategoryLabel(item);

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

        <NoticeCardLink seq={item.seq} className="group block">
          <FeedCard.Title className="mb-2">{item.subject}</FeedCard.Title>
          {item.summary && <NoticeCardSummary summary={item.summary} />}
        </NoticeCardLink>
      </FeedCard.Body>

      {item.thumbnailUrl && (
        <NoticeCardLink seq={item.seq} className="block">
          <FeedCard.Image src={item.thumbnailUrl} alt={item.subject} />
        </NoticeCardLink>
      )}
    </FeedCard>
  );
}
