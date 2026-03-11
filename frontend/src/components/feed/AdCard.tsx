// AdCard — Sponsored feed card using FeedCard compound component
import { useState } from 'react';
import { X, MessageCircle, ArrowRight } from 'lucide-react';
import { useNavigate, useLocation } from 'react-router-dom';
import { api } from '../../api/client';
import { formatAbsoluteDate } from '../../utils/date';
import { useAdImpression } from '../../hooks/useAdImpression';
import { useAdLikeToggle } from '../../hooks/useAdLikeToggle';
import { FeedCard } from './FeedCard';
import { HeartButton } from './HeartButton';
import type { AdItem } from '../../types/api';

export function AdCard({ item }: { item: AdItem }) {
  const { ref } = useAdImpression(item.maSeq);
  const [dismissed, setDismissed] = useState(false);
  const { liked, likeCnt, toggle: toggleLike } = useAdLikeToggle(item.maSeq, item.likeCnt ?? 0, item.userLiked);
  const navigate = useNavigate();
  const location = useLocation();

  const handleClick = () => {
    api.post(`/api/ad/${item.maSeq}/click`).catch(() => {});
  };

  const handleCommentOpen = () => {
    navigate(`/ad/${item.maSeq}`, {
      state: { backgroundLocation: location, adItem: item },
    });
  };

  if (dismissed) return null;

  const ctaLabel = item.cta ?? '자세히 보기';
  const sponsorLabel = item.sponsor ?? '';

  return (
    <FeedCard ref={ref}>
      <FeedCard.Body hasImage={!!item.imageUrl}>
        <FeedCard.Meta
          action={
            <button
              onClick={() => setDismissed(true)}
              className="ml-2 shrink-0 text-text-placeholder hover:text-text-secondary transition-colors p-0.5 -mr-1"
              aria-label="광고 닫기"
            >
              <X size={14} />
            </button>
          }
        >
          {sponsorLabel && <span>{sponsorLabel}</span>}
          {sponsorLabel && <FeedCard.MetaDot />}
          <span className="text-[10px] uppercase tracking-wider font-medium">광고</span>
          {item.regDate && (
            <>
              <FeedCard.MetaDot />
              <span>{formatAbsoluteDate(item.regDate)}</span>
            </>
          )}
        </FeedCard.Meta>

        <a
          href={item.maUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={handleClick}
          className="group block mb-2"
        >
          <FeedCard.Title>{item.maName}</FeedCard.Title>
        </a>

        {item.titleLabel && <FeedCard.Description>{item.titleLabel}</FeedCard.Description>}
      </FeedCard.Body>

      {item.imageUrl && (
        <a
          href={item.maUrl}
          target="_blank"
          rel="noopener noreferrer"
          onClick={handleClick}
          className="block"
        >
          <FeedCard.Image src={item.imageUrl} alt={item.maName} />
        </a>
      )}

      <FeedCard.Divider />

      <FeedCard.Footer>
        <HeartButton liked={liked} onToggle={toggleLike} count={likeCnt} />

        <button
          type="button"
          onClick={handleCommentOpen}
          className="inline-flex items-center gap-1 ml-4 transition-colors hover:text-text-secondary"
        >
          <MessageCircle size={13} />
          {item.commentCnt ?? 0}
        </button>

        {item.maUrl && (
          <a
            href={item.maUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={handleClick}
            className="group inline-flex items-center gap-1 ml-auto text-body-sm font-medium text-text-tertiary hover:text-primary transition-colors"
          >
            {ctaLabel}
            <ArrowRight
              size={14}
              className="transition-transform duration-150 group-hover:translate-x-0.5"
            />
          </a>
        )}
      </FeedCard.Footer>
    </FeedCard>
  );
}
