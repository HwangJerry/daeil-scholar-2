// HistorySection — API-backed editorial timeline for the landing page
import { useState } from 'react';
import { ArrowRight, CalendarDays, CircleAlert, RefreshCw } from 'lucide-react';
import type { HistoryItem, HistoryYearGroup } from '../../hooks/useHistory';
import { useHistory } from '../../hooks/useHistory';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import { Bone } from '../ui/Skeleton';

const SKELETON_GROUP_COUNT = 3;
const INITIAL_VISIBLE_HISTORY_GROUP_COUNT = 3;
const HISTORY_TIMELINE_ID = 'history-timeline';

function displayDate(date: string) {
  const parts = date.split('-');
  if (parts.length < 3) return date;
  return `${parts[1]}.${parts[2]}`;
}

function HistoryTimelineItem({ item }: { item: HistoryItem }) {
  return (
    <li className={cn('relative md:grid md:grid-cols-[72px_minmax(0,1fr)] md:gap-6')}>
      <span
        aria-hidden="true"
        className={cn(
          'absolute -left-[29px] top-1.5 size-2.5 rounded-full border-2 border-background bg-primary md:-left-[37px]',
        )}
      />
      <time
        dateTime={item.eventDate}
        className={cn('text-body-sm font-medium tabular-nums text-text-tertiary')}
      >
        {displayDate(item.eventDate)}
      </time>
      <p className={cn('mt-1 text-body-md leading-7 text-text-secondary md:mt-0')}>
        {item.text}
      </p>
    </li>
  );
}

function HistoryYear({ group }: { group: HistoryYearGroup }) {
  return (
    <li className={cn('md:grid md:grid-cols-[120px_minmax(0,1fr)] md:gap-10')}>
      <h3
        className={cn(
          'font-serif text-3xl font-bold tracking-tight text-text-primary md:text-right md:text-4xl',
        )}
      >
        {group.year}
      </h3>
      <ul
        className={cn(
          'mt-5 space-y-7 border-l border-border pl-7 md:mt-0 md:space-y-8 md:pl-9',
        )}
      >
        {group.items.map((item) => (
          <HistoryTimelineItem key={item.heSeq} item={item} />
        ))}
      </ul>
    </li>
  );
}

function HistoryTimeline({ groups }: { groups: HistoryYearGroup[] }) {
  return (
    <ol id={HISTORY_TIMELINE_ID} className={cn('space-y-12 md:space-y-16')}>
      {groups.map((group) => (
        <HistoryYear key={group.year} group={group} />
      ))}
    </ol>
  );
}

function HistorySkeletonGroup() {
  return (
    <div className={cn('md:grid md:grid-cols-[120px_minmax(0,1fr)] md:gap-10')}>
      <Bone className={cn('h-9 w-24 md:ml-auto md:h-10')} />
      <div className={cn('mt-5 space-y-7 border-l border-border pl-7 md:mt-0 md:pl-9')}>
        {[0, 1].map((itemIndex) => (
          <div
            key={itemIndex}
            className={cn('md:grid md:grid-cols-[72px_minmax(0,1fr)] md:gap-6')}
          >
            <Bone className={cn('h-4 w-12')} />
            <div className={cn('mt-2 space-y-2 md:mt-0')}>
              <Bone className={cn('h-4 w-full')} />
              <Bone className={cn('h-4 w-3/4')} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function HistoryLoading() {
  return (
    <div
      role="status"
      aria-label="연혁 불러오는 중"
      className={cn('space-y-12 md:space-y-16')}
    >
      {Array.from({ length: SKELETON_GROUP_COUNT }, (_, index) => (
        <HistorySkeletonGroup key={index} />
      ))}
    </div>
  );
}

function HistoryEmpty() {
  return (
    <div className={cn('border-y border-border py-14 text-center sm:py-16')}>
      <CalendarDays aria-hidden="true" className={cn('mx-auto size-7 text-primary')} />
      <h3 className={cn('mt-4 font-serif text-xl font-bold text-text-primary')}>
        연혁을 준비하고 있습니다
      </h3>
      <p className={cn('mt-2 text-sm leading-6 text-text-tertiary')}>
        장학회의 발자취를 정리해 곧 소개하겠습니다.
      </p>
    </div>
  );
}

interface HistoryErrorProps {
  isFetching: boolean;
  onRetry: () => void;
}

function HistoryError({ isFetching, onRetry }: HistoryErrorProps) {
  return (
    <div
      role="alert"
      className={cn(
        'flex flex-col items-center border-y border-error-border bg-error-subtle px-5 py-12 text-center sm:py-14',
      )}
    >
      <CircleAlert aria-hidden="true" className={cn('size-7 text-error-text')} />
      <h3 className={cn('mt-4 font-serif text-xl font-bold text-text-primary')}>
        연혁을 불러오지 못했습니다
      </h3>
      <p className={cn('mt-2 text-sm leading-6 text-text-tertiary')}>
        잠시 후 다시 시도해 주세요.
      </p>
      <Button
        type="button"
        variant="outline"
        disabled={isFetching}
        onClick={onRetry}
        className={cn('mt-6 min-h-11')}
      >
        <RefreshCw aria-hidden="true" className={cn('size-4')} />
        다시 시도
      </Button>
    </div>
  );
}

export function HistorySection() {
  const { data, isError, isFetching, isLoading, refetch } = useHistory();
  const [isExpanded, setIsExpanded] = useState(false);
  const groups = (data ?? []).filter((group) => group.items.length > 0);
  const visibleGroups = isExpanded
    ? groups
    : groups.slice(0, INITIAL_VISIBLE_HISTORY_GROUP_COUNT);
  const hasHiddenGroups = groups.length > INITIAL_VISIBLE_HISTORY_GROUP_COUNT;

  return (
    <section
      id="history"
      aria-labelledby="history-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] border-t border-border-subtle bg-background px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-[1080px]')}>
        <header className={cn('max-w-2xl')}>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Our History
          </p>
          <h2
            id="history-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            연혁
          </h2>
          <p className={cn('mt-5 text-sm leading-7 text-text-secondary sm:text-base')}>
            동문의 뜻을 모아 후배들의 내일을 밝혀 온 장학회의 발자취입니다.
          </p>
        </header>

        <div className={cn('mt-10 sm:mt-12 md:mt-16')}>
          {isLoading && <HistoryLoading />}
          {!isLoading && isError && (
            <HistoryError isFetching={isFetching} onRetry={() => void refetch()} />
          )}
          {!isLoading && !isError && groups.length === 0 && <HistoryEmpty />}
          {!isLoading && !isError && groups.length > 0 && (
            <>
              <HistoryTimeline groups={visibleGroups} />
              {hasHiddenGroups && !isExpanded && (
                <div className={cn('mt-10 flex justify-center md:mt-12')}>
                  <Button
                    type="button"
                    variant="outline"
                    aria-controls={HISTORY_TIMELINE_ID}
                    onClick={() => setIsExpanded(true)}
                    className={cn('min-h-11 min-w-36')}
                  >
                    연혁 더 보기
                    <ArrowRight aria-hidden="true" className={cn('size-4')} />
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </section>
  );
}
