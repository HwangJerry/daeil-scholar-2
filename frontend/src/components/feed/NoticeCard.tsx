// NoticeCard — Feed card for a notice item with warm editorial styling and category badge
import { Eye, ChevronRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { cn } from '../../lib/utils';
import type { NoticeItem } from '../../types/api';

const CATEGORY_STYLES: Record<string, { bg: string; text: string; border: string; label: string }> = {
  notice: {
    bg: 'bg-cat-notice-bg',
    text: 'text-cat-notice-text',
    border: 'border-cat-notice-border',
    label: '공지',
  },
  event: {
    bg: 'bg-cat-event-bg',
    text: 'text-cat-event-text',
    border: 'border-cat-event-border',
    label: '이벤트',
  },
  scholarship: {
    bg: 'bg-cat-scholarship-bg',
    text: 'text-cat-scholarship-text',
    border: 'border-cat-scholarship-border',
    label: '장학',
  },
  career: {
    bg: 'bg-cat-career-bg',
    text: 'text-cat-career-text',
    border: 'border-cat-career-border',
    label: '커리어',
  },
};

const DEFAULT_CATEGORY = CATEGORY_STYLES.notice;

export function NoticeCard({ item }: { item: NoticeItem }) {
  const categoryStyle = item.category
    ? (CATEGORY_STYLES[item.category] ?? DEFAULT_CATEGORY)
    : DEFAULT_CATEGORY;

  return (
    <Link
      to={`/post/${item.seq}`}
      className={cn(
        'group block overflow-hidden rounded-2xl bg-surface',
        'border border-border-subtle shadow-card',
        'transition-all duration-200 hover:shadow-card-hover hover:border-border-hover hover:-translate-y-0.5'
      )}
    >
      {item.thumbnailUrl && (
        <div className="overflow-hidden">
          <img
            src={item.thumbnailUrl}
            alt={item.subject}
            className="h-40 w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
            loading="lazy"
          />
        </div>
      )}
      <div className="p-4 pt-3">
        <div className="flex items-center justify-between mb-2">
          <span
            className={cn(
              'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold border',
              categoryStyle.bg,
              categoryStyle.text,
              categoryStyle.border
            )}
          >
            {categoryStyle.label}
          </span>
          {item.isPinned === 'Y' && (
            <span className="text-xs text-text-placeholder">고정</span>
          )}
        </div>
        <div className="flex items-start justify-between gap-2">
          <h3 className="flex-1 mb-1 line-clamp-2 text-[15px] font-semibold text-text-primary font-serif group-hover:text-primary transition-colors duration-150">
            {item.subject}
          </h3>
          <ChevronRight
            size={16}
            className="flex-shrink-0 text-text-placeholder opacity-0 group-hover:opacity-100 transition-opacity duration-150 mt-0.5"
          />
        </div>
        {item.summary && (
          <p className="mb-2.5 line-clamp-2 text-[13px] text-text-tertiary leading-relaxed">{item.summary}</p>
        )}
        <div className="flex items-center gap-3 text-xs text-text-placeholder">
          <span>{item.regDate}</span>
          <span className="ml-auto flex items-center gap-0.5">
            <Eye className="h-3.5 w-3.5" />
            {item.hit}
          </span>
        </div>
      </div>
    </Link>
  );
}
