// NoticeCard — Feed card with category, body-first layout, and expandable summary
import { Pin } from 'lucide-react';
import { cn } from '../../lib/utils';
import { formatAbsoluteDate } from '../../utils/date';
import type { NoticeItem } from '../../types/api';
import { NoticeCardEngagement } from './NoticeCardEngagement';
import { NoticeCardLink } from './NoticeCardLink';
import { NoticeCardSummary } from './NoticeCardSummary';
import { NOTICE_CATEGORY_LABELS } from './noticeCard.constants';

export function NoticeCard({ item }: { item: NoticeItem }) {
  const category = item.category ?? 'notice';
  const categoryLabel = NOTICE_CATEGORY_LABELS[category] ?? '공지';

  return (
    <article className="rounded-2xl bg-surface border border-border-subtle shadow-card transition-all duration-200 hover:shadow-card-hover hover:border-border-hover overflow-hidden">
      <div className={cn('px-5 pt-5', item.thumbnailUrl ? 'pb-4' : 'pb-5')}>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-1.5 text-caption text-text-placeholder">
            <span>{categoryLabel}</span>
            <span className="opacity-40">·</span>
            <span>{formatAbsoluteDate(item.regDate)}</span>
          </div>
          {item.isPinned === 'Y' && (
            <Pin size={13} className="text-text-placeholder flex-shrink-0" />
          )}
        </div>

        <NoticeCardLink seq={item.seq} className="group block mb-2">
          <h3 className="line-clamp-2 text-body-md font-semibold font-serif text-text-primary group-hover:text-primary transition-colors duration-150">
            {item.subject}
          </h3>
        </NoticeCardLink>

        {item.summary && <NoticeCardSummary summary={item.summary} />}
      </div>

      {item.thumbnailUrl && (
        <NoticeCardLink seq={item.seq} className="block">
          <img
            src={item.thumbnailUrl}
            alt={item.subject}
            className="w-full aspect-video object-cover"
            loading="lazy"
          />
        </NoticeCardLink>
      )}

      <div className="border-t border-border-subtle mx-5" />

      <NoticeCardEngagement
        seq={item.seq}
        likeCnt={item.likeCnt ?? 0}
        commentCnt={item.commentCnt ?? 0}
        hit={item.hit ?? 0}
        userLiked={item.userLiked ?? false}
      />
    </article>
  );
}
