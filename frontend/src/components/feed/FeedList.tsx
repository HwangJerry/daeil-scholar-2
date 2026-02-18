// Renders the infinite-scrolling feed of notice and ad cards
import { useInfiniteScroll } from '../../hooks/useInfiniteScroll';
import { useFeedPagination } from '../../hooks/useFeedPagination';
import { NoticeCard } from './NoticeCard';
import { AdCard } from './AdCard';

export function FeedList() {
  const { items, hasMore, isFetching, loadMore } = useFeedPagination();

  const { sentinelRef } = useInfiniteScroll({
    hasMore,
    isFetching,
    onLoadMore: loadMore,
  });

  if (items.length === 0 && !isFetching) {
    return <p className="py-12 text-center text-sm text-text-tertiary">게시글이 없습니다.</p>;
  }

  const staggerClass = (idx: number) => {
    const delays = ['stagger-1', 'stagger-2', 'stagger-3', 'stagger-4', 'stagger-5'];
    return `animate-fade-in-up ${delays[idx % delays.length]}`;
  };

  return (
    <div className="flex flex-col gap-4">
      {items.map((item, idx) => {
        if (item.type === 'ad') {
          return (
            <div key={`ad-${item.maSeq}-${idx}`} className={staggerClass(idx)}>
              <AdCard item={item} />
            </div>
          );
        }
        return (
          <div key={`notice-${item.seq}`} className={staggerClass(idx)}>
            <NoticeCard item={item} />
          </div>
        );
      })}
      {isFetching && (
        <div className="flex justify-center py-6">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      )}
      {hasMore && <div ref={sentinelRef} className="h-4" />}
    </div>
  );
}
