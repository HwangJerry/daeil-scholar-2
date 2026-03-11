// AdDetailModal — Feed-to-modal ad detail view with sticky CTA bar and full comment section
import { useNavigate, useLocation, Navigate } from 'react-router-dom';
import { X, ArrowRight } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { AdCommentSection } from './AdCommentSection';
import { HeartButton } from './HeartButton';
import { useAdLikeToggle } from '../../hooks/useAdLikeToggle';
import { useAuthRedirect } from '../../hooks/useAuthRedirect';
import { formatAbsoluteDate } from '../../utils/date';
import { api } from '../../api/client';
import type { AdItem } from '../../types/api';

export function AdDetailModal() {
  const navigate = useNavigate();
  const location = useLocation();
  const adItem = (location.state as { adItem?: AdItem } | null)?.adItem;

  if (!adItem) {
    return <Navigate to="/" replace />;
  }

  return <AdDetailContent adItem={adItem} onClose={() => navigate(-1)} />;
}

interface AdDetailContentProps {
  adItem: AdItem;
  onClose: () => void;
}

function AdDetailContent({ adItem, onClose }: AdDetailContentProps) {
  const onAuthError = useAuthRedirect();
  const { liked, likeCnt, toggle } = useAdLikeToggle(adItem.maSeq, adItem.likeCnt ?? 0, adItem.userLiked, { onAuthError });

  const handleAdClick = () => {
    api.post(`/api/ad/${adItem.maSeq}/click`).catch(() => {
      // fire-and-forget
    });
  };

  const ctaLabel = adItem.cta ?? '자세히 보기';

  return (
    <Modal onClose={onClose}>
      <div className="flex flex-col max-h-[85vh]">
        <div className="flex-1 overflow-y-auto">
          <div className="flex justify-end p-3 pb-0">
            <button
              onClick={onClose}
              className="rounded-lg p-1.5 text-text-tertiary hover:bg-background transition-colors duration-150"
              aria-label="닫기"
            >
              <X size={20} />
            </button>
          </div>

          {adItem.imageUrl && (
            <img
              src={adItem.imageUrl}
              alt={adItem.maName}
              className="w-full aspect-video object-cover"
            />
          )}

          <div className="p-5">
            <div className="flex items-center gap-1.5 text-caption text-text-placeholder mb-3">
              {adItem.sponsor && (
                <>
                  <span>{adItem.sponsor}</span>
                  <span className="opacity-40">·</span>
                </>
              )}
              <span className="text-[10px] uppercase tracking-wider font-medium">광고</span>
              {adItem.regDate && (
                <>
                  <span className="opacity-40">·</span>
                  <span>{formatAbsoluteDate(adItem.regDate)}</span>
                </>
              )}
            </div>

            <h2 className="text-lg font-semibold font-serif text-text-primary mb-1">
              {adItem.maName}
            </h2>

            {adItem.titleLabel && (
              <p className="text-body-sm text-text-tertiary leading-relaxed whitespace-pre-line mb-4">
                {adItem.titleLabel}
              </p>
            )}
          </div>

          <div className="border-t border-border-subtle mx-5" />

          <div className="px-5 py-2.5">
            <HeartButton liked={liked} onToggle={toggle} count={likeCnt} />
          </div>

          <AdCommentSection maSeq={adItem.maSeq} />
        </div>

        <div className="border-t border-border-subtle p-4">
          <a
            href={adItem.maUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={handleAdClick}
            className="group flex items-center justify-center gap-2 w-full rounded-lg bg-primary text-white py-3 font-semibold transition-colors hover:bg-primary/90"
          >
            {ctaLabel}
            <ArrowRight
              size={16}
              className="transition-transform duration-150 group-hover:translate-x-0.5"
            />
          </a>
        </div>
      </div>
    </Modal>
  );
}
