// In-feed advertisement card with impression tracking and tier badge
import { api } from '../../api/client';
import { useAdImpression } from '../../hooks/useAdImpression';
import { Badge } from '../ui/Badge';
import type { AdItem } from '../../types/api';

const TIER_LABELS: Record<string, string> = {
  PREMIUM: '가장 핫한 동문 소식',
  GOLD: '추천 동문 소식',
  NORMAL: '추천 동문 소식',
};

export function AdCard({ item }: { item: AdItem }) {
  const { ref } = useAdImpression(item.maSeq);

  const handleClick = () => {
    api.post(`/api/ad/${item.maSeq}/click`).catch(() => {
      // fire-and-forget
    });
  };

  const label = item.titleLabel || TIER_LABELS[item.adTier] || '추천 동문 소식';

  return (
    <a
      ref={ref}
      href={item.maUrl}
      target="_blank"
      rel="noopener noreferrer"
      onClick={handleClick}
      className="group block overflow-hidden rounded-xl bg-surface border border-border-subtle shadow-card transition-all duration-200 hover:shadow-card-hover hover:-translate-y-0.5"
    >
      {item.imageUrl && (
        <div className="overflow-hidden">
          <img
            src={item.imageUrl}
            alt={item.maName}
            className="h-40 w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
            loading="lazy"
          />
        </div>
      )}
      <div className="p-4 pt-3">
        <Badge variant="warning" className="mb-1.5">{label}</Badge>
        <h3 className="line-clamp-2 text-[15px] font-semibold text-text-primary group-hover:text-primary">
          {item.maName}
        </h3>
        <p className="mt-1.5 text-xs text-text-placeholder">AD</p>
      </div>
    </a>
  );
}
