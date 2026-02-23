// AdCard — Sponsored feed card with unified layout matching NoticeCard
import { useState } from 'react';
import { X, ArrowRight } from 'lucide-react';
import { cn } from '../../lib/utils';
import { formatAbsoluteDate } from '../../utils/date';
import { api } from '../../api/client';
import { useAdImpression } from '../../hooks/useAdImpression';
import { useAdLikeToggle } from '../../hooks/useAdLikeToggle';
import { AdCardActions } from './AdCardActions';
import { AdCommentSection } from './AdCommentSection';
import type { AdItem } from '../../types/api';

export function AdCard({ item }: { item: AdItem }) {
  const { ref } = useAdImpression(item.maSeq);
  const [dismissed, setDismissed] = useState(false);
  const [showComments, setShowComments] = useState(false);
  const { liked, likeCnt, toggle: toggleLike } = useAdLikeToggle(item.maSeq, item.likeCnt ?? 0);

  const handleClick = () => {
    api.post(`/api/ad/${item.maSeq}/click`).catch(() => {
      // fire-and-forget
    });
  };

  if (dismissed) return null;

  const ctaLabel = item.cta ?? '자세히 보기';

  return (
    <article
      ref={ref}
      className="rounded-2xl bg-surface border border-border-subtle shadow-card transition-all duration-200 hover:shadow-card-hover hover:border-border-hover overflow-hidden"
    >
      <div className={cn('px-5 pt-5', item.imageUrl ? 'pb-4' : 'pb-0')}>
        <div className="flex items-start justify-between mb-1">
          <a
            href={item.maUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={handleClick}
            className="group flex items-baseline gap-2 flex-1 min-w-0"
          >
            <h3 className="line-clamp-2 text-body-md font-semibold font-serif text-text-primary group-hover:text-primary transition-colors duration-150">
              {item.maName}
            </h3>
            <span className="shrink-0 text-[10px] font-normal text-text-placeholder/50 tracking-wide uppercase leading-none">
              sponsored
            </span>
          </a>
          <button
            onClick={() => setDismissed(true)}
            className="ml-2 shrink-0 text-text-placeholder hover:text-text-secondary transition-colors p-0.5 -mr-1"
            aria-label="광고 닫기"
          >
            <X size={14} />
          </button>
        </div>

        <span className="text-caption text-text-placeholder mb-2 block">
          {item.regDate ? formatAbsoluteDate(item.regDate) : ''}
        </span>

        {item.titleLabel && (
          <div className="mb-1">
            <p className="text-body-sm text-text-tertiary leading-relaxed line-clamp-3 whitespace-pre-line">
              {item.titleLabel}
            </p>
          </div>
        )}
      </div>

      {item.imageUrl ? (
        <a
          href={item.maUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={handleClick}
          className="block"
        >
          <img
            src={item.imageUrl}
            alt={item.maName}
            className="w-full aspect-video object-cover"
            loading="lazy"
          />
        </a>
      ) : (
        <div className="border-t border-border-subtle mx-5" />
      )}

      <div className="px-5 py-3">
        <a
          href={item.maUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={handleClick}
          className="group inline-flex items-center gap-1.5 rounded-lg border border-border-hover px-4 py-2 text-body-sm font-medium text-text-primary transition-all duration-150 hover:border-primary hover:text-primary"
        >
          {ctaLabel}
          <ArrowRight
            size={14}
            className="transition-transform duration-150 group-hover:translate-x-0.5"
          />
        </a>
      </div>

      <div className="border-t border-border-subtle mx-5" />

      <AdCardActions
        liked={liked}
        likeCnt={likeCnt}
        showComments={showComments}
        commentCnt={item.commentCnt ?? 0}
        onLikeToggle={toggleLike}
        onCommentToggle={() => setShowComments((v) => !v)}
      />

      {showComments && <AdCommentSection maSeq={item.maSeq} />}
    </article>
  );
}
