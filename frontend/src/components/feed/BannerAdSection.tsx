// BannerAdSection — Accessible landing-page sponsor banner carousel
import { useState } from 'react';
import { ChevronLeft, ChevronRight, ExternalLink } from 'lucide-react';
import { api } from '../../api/client';
import { useActiveBannerAd } from '../../hooks/useActiveBannerAd';
import { useBannerAdImpression } from '../../hooks/useBannerAdImpression';
import { cn } from '../../lib/utils';
import type { BannerAd } from '../../types/api';

const SWIPE_THRESHOLD_PX = 40;

export function BannerAdSection() {
  const { data: banner, isLoading, isError } = useActiveBannerAd();

  if (isLoading || isError || !banner || banner.images.length === 0) {
    return null;
  }

  return <BannerAdCarousel banner={banner} />;
}

function BannerAdCarousel({ banner }: { banner: BannerAd }) {
  const sortedImages = [...banner.images].sort((left, right) => left.sortOrder - right.sortOrder);
  const hasMultipleImages = sortedImages.length > 1;
  const { ref } = useBannerAdImpression(banner.bnSeq);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [touchStartX, setTouchStartX] = useState<number | null>(null);

  const moveToSlide = (nextIndex: number) => {
    const imageCount = sortedImages.length;

    if (imageCount === 0) return;
    if (!hasMultipleImages) {
      setSelectedIndex(0);
      return;
    }

    const normalizedIndex = (nextIndex + imageCount) % imageCount;
    setSelectedIndex(normalizedIndex);
  };

  const handleClick = () => {
    api.post(`/api/banner-ad/${banner.bnSeq}/click`).catch(() => {});
  };

  const handleTouchStart = (event: React.TouchEvent<HTMLDivElement>) => {
    setTouchStartX(event.changedTouches[0]?.clientX ?? null);
  };

  const handleTouchEnd = (event: React.TouchEvent<HTMLDivElement>) => {
    if (touchStartX === null) return;

    const touchEndX = event.changedTouches[0]?.clientX ?? touchStartX;
    const swipeDistance = touchStartX - touchEndX;
    if (Math.abs(swipeDistance) >= SWIPE_THRESHOLD_PX) {
      moveToSlide(selectedIndex + (swipeDistance > 0 ? 1 : -1));
    }

    setTouchStartX(null);
  };

  return (
    <section
      ref={ref}
      aria-labelledby="banner-ad-heading"
      aria-roledescription="carousel"
      className={cn(
        'relative overflow-hidden rounded-xl border border-border bg-gradient-to-br from-primary-light via-surface to-background shadow-card',
      )}
    >
      <div
        aria-hidden="true"
        className={cn('pointer-events-none absolute inset-x-0 top-0 h-24 bg-surface/40 blur-3xl')}
      />
      <div className={cn('relative p-5 pb-4 md:p-6')}>
        <div className={cn('mb-4 flex items-start justify-between gap-3')}>
          <div>
            <p
              className={cn(
                'mb-1 text-2xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
              )}
            >
              Banner Ad
            </p>
            <h2
              id="banner-ad-heading"
              className={cn('font-serif text-lg font-semibold text-text-primary')}
            >
              {banner.bnName}
            </h2>
          </div>
          <span
            className={cn(
              'inline-flex items-center gap-1 rounded-full bg-surface/80 px-2.5 py-1 text-caption font-medium text-text-secondary',
            )}
          >
            <ExternalLink aria-hidden="true" size={12} />
            스폰서 링크
          </span>
        </div>

        {hasMultipleImages ? (
          <div className={cn('relative')}>
            <div
              className={cn('overflow-hidden rounded-lg')}
              onTouchStart={handleTouchStart}
              onTouchEnd={handleTouchEnd}
            >
              <div
                className={cn('flex transition-transform duration-300 ease-out')}
                style={{ transform: `translateX(-${selectedIndex * 100}%)` }}
              >
                {sortedImages.map((image) => (
                  <div key={image.bniSeq} className={cn('min-w-0 flex-[0_0_100%]')}>
                    <BannerAdLink
                      href={banner.bnUrl}
                      name={banner.bnName}
                      imageUrl={image.imageUrl}
                      onClick={handleClick}
                    />
                  </div>
                ))}
              </div>
            </div>

            <button
              type="button"
              onClick={() => moveToSlide(selectedIndex - 1)}
              className={cn(
                'absolute left-3 top-1/2 inline-flex size-11 -translate-y-1/2 items-center justify-center rounded-full bg-primary/70 text-surface backdrop-blur-sm transition-colors hover:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-surface focus-visible:ring-offset-2 focus-visible:ring-offset-primary',
              )}
              aria-label="이전 배너 이미지"
            >
              <ChevronLeft aria-hidden="true" size={18} />
            </button>

            <button
              type="button"
              onClick={() => moveToSlide(selectedIndex + 1)}
              className={cn(
                'absolute right-3 top-1/2 inline-flex size-11 -translate-y-1/2 items-center justify-center rounded-full bg-primary/70 text-surface backdrop-blur-sm transition-colors hover:bg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-surface focus-visible:ring-offset-2 focus-visible:ring-offset-primary',
              )}
              aria-label="다음 배너 이미지"
            >
              <ChevronRight aria-hidden="true" size={18} />
            </button>

            <div className={cn('mt-2 flex items-center justify-center')}>
              {sortedImages.map((image, index) => {
                const isActive = index === selectedIndex;

                return (
                  <button
                    key={image.bniSeq}
                    type="button"
                    onClick={() => moveToSlide(index)}
                    className={cn(
                      'group/indicator inline-flex size-11 items-center justify-center rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background',
                    )}
                    aria-label={`${index + 1}번 배너로 이동`}
                    aria-pressed={isActive}
                  >
                    <span
                      aria-hidden="true"
                      className={cn(
                        'h-2 rounded-full transition-[width,background-color]',
                        isActive
                          ? 'w-6 bg-primary'
                          : 'w-2 bg-text-placeholder/40 group-hover/indicator:bg-text-placeholder/70',
                      )}
                    />
                  </button>
                );
              })}
            </div>
          </div>
        ) : (
          <BannerAdLink href={banner.bnUrl} name={banner.bnName} imageUrl={sortedImages[0].imageUrl} onClick={handleClick} />
        )}
      </div>
    </section>
  );
}

function BannerAdLink({
  href,
  name,
  imageUrl,
  onClick,
}: {
  href: string;
  name: string;
  imageUrl: string;
  onClick: () => void;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      onClick={onClick}
      className={cn(
        'group block overflow-hidden rounded-lg bg-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background',
      )}
    >
      <div className={cn('relative aspect-[16/7] min-h-[180px] overflow-hidden bg-background')}>
        <img
          src={imageUrl}
          alt={name}
          className={cn('h-full w-full object-cover')}
        />
        <div
          aria-hidden="true"
          className={cn(
            'pointer-events-none absolute inset-0 bg-gradient-to-t from-primary/20 via-transparent to-transparent',
          )}
        />
      </div>
    </a>
  );
}
