// LatestNewsSection — Landing-page editorial preview of the public notice feed
import { useCallback, useMemo, useState } from 'react';
import type { LucideIcon } from 'lucide-react';
import { AlertCircle, ArrowRight, CalendarDays, LoaderCircle, Newspaper } from 'lucide-react';
import type { NoticeItem } from '../../types/api';
import { useFeedPagination } from '../../hooks/useFeedPagination';
import { useHeroNotice } from '../../hooks/useHeroNotice';
import { useInfiniteScroll } from '../../hooks/useInfiniteScroll';
import { cn } from '../../lib/utils';
import { formatAbsoluteDate } from '../../utils/date';
import { FeedCardSkeleton } from '../feed/FeedCardSkeleton';
import { HeroSection } from '../feed/HeroSection';
import { NoticeCardLink } from '../feed/NoticeCardLink';
import { Button } from '../ui/Button';

const INITIAL_VISIBLE_NOTICE_COUNT = 4;
const VISIBLE_NOTICE_INCREMENT = 4;
const LOADING_SKELETON_COUNT = 4;
const SKELETON_STAGGER_SECONDS = 0.05;
const STAGGER_CLASSES = ['stagger-1', 'stagger-2', 'stagger-3', 'stagger-4', 'stagger-5'];

interface NewsStatePanelProps {
  description: string;
  icon: LucideIcon;
  title: string;
  onRetry?: () => void;
}

function NewsStatePanel({ description, icon: Icon, title, onRetry }: NewsStatePanelProps) {
  return (
    <div
      className={cn(
        'flex min-h-52 flex-col items-center justify-center rounded-xl border border-border-subtle bg-surface px-6 py-10 text-center shadow-xs',
      )}
    >
      <span
        className={cn(
          'mb-4 inline-flex size-11 items-center justify-center rounded-full bg-primary-light text-primary',
        )}
      >
        <Icon aria-hidden="true" className={cn('size-5')} />
      </span>
      <h3 className={cn('font-serif text-lg font-semibold text-text-primary')}>{title}</h3>
      <p className={cn('mt-2 max-w-xs text-body-sm leading-relaxed text-text-tertiary')}>
        {description}
      </p>
      {onRetry && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onRetry}
          className={cn('mt-5 min-h-11')}
        >
          다시 시도
        </Button>
      )}
    </div>
  );
}

function LatestNewsHero() {
  const { data: hero, isError, isLoading, refetch } = useHeroNotice();

  if (isLoading) {
    return (
      <div role="status" aria-label="대표 소식 불러오는 중">
        <HeroSection variant="landing" />
      </div>
    );
  }

  if (isError) {
    return (
      <NewsStatePanel
        icon={AlertCircle}
        title="대표 소식을 불러오지 못했습니다"
        description="잠시 후 다시 시도해 주세요. 최신 소식 목록은 계속 확인할 수 있습니다."
        onRetry={() => void refetch()}
      />
    );
  }

  if (!hero) {
    return (
      <NewsStatePanel
        icon={Newspaper}
        title="대표 공지가 없습니다"
        description="새로운 대표 소식이 등록되면 이곳에서 가장 먼저 알려드릴게요."
      />
    );
  }

  return <HeroSection variant="landing" />;
}

interface LatestNewsListItemProps {
  className?: string;
  item: NoticeItem;
}

function LatestNewsListItem({ className, item }: LatestNewsListItemProps) {
  return (
    <li
      className={cn(
        'border-b border-border-subtle last:border-b-0',
        className,
      )}
    >
      <NoticeCardLink
        seq={item.seq}
        className={cn(
          'group grid grid-cols-[minmax(0,1fr)_5.5rem] gap-4 py-5 focus-visible:rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:grid-cols-[minmax(0,1fr)_7rem]',
        )}
      >
        <div className={cn('min-w-0 self-center')}>
          <div
            className={cn(
              'mb-2 flex items-center gap-1.5 text-caption font-medium text-text-placeholder',
            )}
          >
            <CalendarDays aria-hidden="true" className={cn('size-3.5')} />
            <time dateTime={item.regDate}>{formatAbsoluteDate(item.regDate)}</time>
          </div>
          <h3
            className={cn(
              'line-clamp-2 font-serif text-body-md font-semibold leading-snug text-text-primary transition-colors group-hover:text-primary sm:text-base',
            )}
          >
            {item.subject}
          </h3>
          {item.summary && (
            <p
              className={cn(
                'mt-2 line-clamp-2 text-body-sm leading-relaxed text-text-tertiary',
              )}
            >
              {item.summary}
            </p>
          )}
        </div>

        <div
          className={cn(
            'aspect-square overflow-hidden rounded-lg border border-border-subtle bg-primary-light',
          )}
        >
          {item.thumbnailUrl ? (
            <img
              src={item.thumbnailUrl}
              alt=""
              loading="lazy"
              className={cn(
                'h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]',
              )}
            />
          ) : (
            <span className={cn('flex h-full items-center justify-center text-primary-muted')}>
              <Newspaper aria-hidden="true" className={cn('size-6')} />
            </span>
          )}
        </div>
      </NoticeCardLink>
    </li>
  );
}

function LatestNewsList({ heroSeq }: { heroSeq?: number }) {
  const { items, hasMore, isError, isFetching, loadMore, refetch } = useFeedPagination();
  const [visibleCount, setVisibleCount] = useState(INITIAL_VISIBLE_NOTICE_COUNT);
  const [hasRequestedMore, setHasRequestedMore] = useState(false);
  const notices = useMemo(
    () => items.filter((item) => item.seq !== heroSeq),
    [heroSeq, items],
  );
  const visibleNotices = notices.slice(0, visibleCount);
  const hasHiddenLoadedNotices = visibleCount < notices.length;

  const handleLoadMore = useCallback(async () => {
    setHasRequestedMore(true);

    if (hasHiddenLoadedNotices) {
      setVisibleCount((currentCount) => currentCount + VISIBLE_NOTICE_INCREMENT);
      return;
    }

    if (!hasMore || isFetching) return;

    setVisibleCount((currentCount) => currentCount + VISIBLE_NOTICE_INCREMENT);
    await loadMore();
  }, [hasHiddenLoadedNotices, hasMore, isFetching, loadMore]);

  const { sentinelRef } = useInfiniteScroll({
    hasMore: hasRequestedMore && hasMore && !hasHiddenLoadedNotices,
    isFetching,
    onLoadMore: handleLoadMore,
  });

  if (notices.length === 0 && isFetching) {
    return (
      <div role="status" aria-label="최신 소식 목록 불러오는 중">
        {Array.from({ length: LOADING_SKELETON_COUNT }, (_, index) => (
          <div
            key={index}
            className={cn('animate-fade-in-up')}
            style={{ animationDelay: `${index * SKELETON_STAGGER_SECONDS}s` }}
          >
            <FeedCardSkeleton variant="compact" />
          </div>
        ))}
      </div>
    );
  }

  if (notices.length === 0 && isError) {
    return (
      <NewsStatePanel
        icon={AlertCircle}
        title="최신 소식을 불러오지 못했습니다"
        description="네트워크 연결을 확인한 뒤 다시 시도해 주세요."
        onRetry={() => void refetch()}
      />
    );
  }

  if (notices.length === 0) {
    return (
      <NewsStatePanel
        icon={Newspaper}
        title="새로운 소식을 준비하고 있습니다"
        description="장학회 활동과 지원 소식이 등록되면 이곳에서 확인할 수 있습니다."
      />
    );
  }

  const canShowMore = hasHiddenLoadedNotices || hasMore;

  return (
    <div>
      <ol aria-label="최신 공지 목록">
        {visibleNotices.map((item, index) => (
          <LatestNewsListItem
            key={item.seq}
            item={item}
            className={cn(
              'animate-fade-in-up',
              STAGGER_CLASSES[index % STAGGER_CLASSES.length],
            )}
          />
        ))}
      </ol>

      {isError && (
        <div
          role="alert"
          className={cn(
            'mt-4 flex items-center justify-between gap-3 rounded-lg border border-error-border bg-error-subtle px-4 py-3 text-body-sm text-error-text',
          )}
        >
          <span>추가 소식을 불러오지 못했습니다.</span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void refetch()}
            className={cn('min-h-11')}
          >
            다시 시도
          </Button>
        </div>
      )}

      {canShowMore && !isError && (
        <div className={cn('mt-6 flex flex-col items-center gap-3')}>
          <Button
            type="button"
            variant="outline"
            onClick={() => void handleLoadMore()}
            disabled={isFetching}
            className={cn('min-h-11 min-w-36')}
          >
            {isFetching ? (
              <>
                <LoaderCircle aria-hidden="true" className={cn('size-4 animate-spin')} />
                불러오는 중
              </>
            ) : (
              <>
                소식 더 보기
                <ArrowRight aria-hidden="true" className={cn('size-4')} />
              </>
            )}
          </Button>
          {hasMore && !hasHiddenLoadedNotices && (
            <div ref={sentinelRef} aria-hidden="true" className={cn('h-px w-full')} />
          )}
        </div>
      )}
    </div>
  );
}

export function LatestNewsSection() {
  const { data: hero } = useHeroNotice();

  return (
    <section
      id="news"
      aria-labelledby="latest-news-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] bg-background px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-[1080px]')}>
        <header className={cn('mb-9 max-w-2xl md:mb-12')}>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Latest News
          </p>
          <h2
            id="latest-news-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold tracking-tight text-text-primary sm:text-4xl',
            )}
          >
            최근 소식
          </h2>
          <p className={cn('mt-4 text-sm leading-7 text-text-tertiary sm:text-base')}>
            장학회의 새로운 활동과 후배들을 위한 지원 이야기를 빠르게 만나보세요.
          </p>
        </header>

        <div
          className={cn(
            'grid gap-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:items-start lg:gap-10',
          )}
        >
          <div className={cn('lg:sticky lg:top-[calc(var(--landing-header-height)+2rem)]')}>
            <LatestNewsHero />
          </div>
          <LatestNewsList heroSeq={hero?.seq} />
        </div>
      </div>
    </section>
  );
}
