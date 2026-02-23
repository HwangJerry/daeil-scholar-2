// NoticeCard — Feed card with category, body-first layout, and expandable summary
import { useState } from 'react';
import { Pin } from 'lucide-react';
import { Link } from 'react-router-dom';
import { cn } from '../../lib/utils';
import { formatAbsoluteDate } from '../../utils/date';
import type { NoticeItem } from '../../types/api';
import { NoticeCardEngagement } from './NoticeCardEngagement';

const CATEGORY_LABELS: Record<string, string> = {
  notice: '공지',
  event: '이벤트',
  scholarship: '장학',
  career: '커리어',
};

const SUMMARY_COLLAPSE_THRESHOLD = 120;

export function NoticeCard({ item }: { item: NoticeItem }) {
  const [isExpanded, setIsExpanded] = useState(false);

  const category = item.category ?? 'notice';
  const categoryLabel = CATEGORY_LABELS[category] ?? '공지';
  const hasLongSummary = item.summary.length > SUMMARY_COLLAPSE_THRESHOLD;

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

        <Link to={`/post/${item.seq}`} className="group block mb-2">
          <h3 className="line-clamp-2 text-body-md font-semibold font-serif text-text-primary group-hover:text-primary transition-colors duration-150">
            {item.subject}
          </h3>
        </Link>

        {item.summary && (
          <div className="mb-1">
            <p
              className={cn(
                'text-body-sm text-text-tertiary leading-relaxed whitespace-pre-line',
                !isExpanded && 'line-clamp-3'
              )}
            >
              {item.summary}
            </p>
            {hasLongSummary && (
              <button
                onClick={() => setIsExpanded((prev) => !prev)}
                className="mt-1 text-body-sm text-text-placeholder hover:text-text-secondary transition-colors"
              >
                {isExpanded ? '접기' : '더 보기'}
              </button>
            )}
          </div>
        )}
      </div>

      {item.thumbnailUrl && (
        <Link to={`/post/${item.seq}`} className="block">
          <img
            src={item.thumbnailUrl}
            alt={item.subject}
            className="w-full aspect-video object-cover"
            loading="lazy"
          />
        </Link>
      )}

      <div className="border-t border-border-subtle mx-5" />

      <NoticeCardEngagement
        seq={item.seq}
        likeCnt={item.likeCnt ?? 0}
        commentCnt={item.commentCnt ?? 0}
        hit={item.hit ?? 0}
      />
    </article>
  );
}
