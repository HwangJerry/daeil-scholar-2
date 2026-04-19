// AlumniPage — Alumni directory with table layout, search filters, and skeleton loading
import { useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { AuthGuard } from '../components/auth/AuthGuard';
import { PageMeta } from '../components/seo/PageMeta';
import { SearchFilter, type AlumniSearchParams } from '../components/alumni/SearchFilter';
import { AlumniCard } from '../components/alumni/AlumniCard';
import { Bone } from '../components/ui/Skeleton';
import { useInfiniteScroll } from '../hooks/useInfiniteScroll';
import type { AlumniSearchResponse, AlumniItem } from '../types/api';

const PAGE_SIZE = 30;
const SKELETON_ROW_COUNT = 8;
const STAGGER_CLASSES = ['stagger-1', 'stagger-2', 'stagger-3', 'stagger-4', 'stagger-5'];

function buildSearchUrl(params: AlumniSearchParams, page: number): string {
  const qs = new URLSearchParams();
  if (params.fn) qs.set('fn', params.fn);
  if (params.dept) qs.set('dept', params.dept);
  if (params.name) qs.set('name', params.name);
  if (params.jobCat) qs.set('jobCat', params.jobCat);
  qs.set('page', String(page));
  qs.set('size', String(PAGE_SIZE));
  return `/api/alumni?${qs.toString()}`;
}

function TableColumnHeader() {
  return (
    <div className="grid items-center px-5 py-2.5 gap-2 bg-background border-b border-border grid-cols-[44px_1fr_48px] md:grid-cols-[44px_1fr_200px_160px_72px]">
      <div />
      <span className="text-xs font-medium text-text-placeholder">이름 / 기수</span>
      <span className="hidden md:block text-xs font-medium text-text-placeholder">소속</span>
      <span className="hidden md:block text-xs font-medium text-text-placeholder">직책</span>
      <div />
    </div>
  );
}

function AlumniTableSkeleton() {
  return (
    <div className="bg-surface border border-border rounded-2xl overflow-hidden">
      <TableColumnHeader />
      {Array.from({ length: SKELETON_ROW_COUNT }, (_, i) => (
        <div
          key={i}
          className={`grid items-center px-5 py-3.5 gap-2 animate-fade-in-up grid-cols-[44px_1fr_48px] md:grid-cols-[44px_1fr_200px_160px_72px] ${i < SKELETON_ROW_COUNT - 1 ? 'border-b border-border-subtle' : ''}`}
          style={{ animationDelay: `${i * 0.05}s` }}
        >
          <Bone className="w-8 h-8 rounded-full" />
          <div className="space-y-1.5">
            <Bone className="h-3.5 w-24" />
            <Bone className="h-2.5 w-32" />
          </div>
          <Bone className="hidden md:block h-3 w-28" />
          <Bone className="hidden md:block h-3 w-16" />
          <Bone className="h-7 w-10 rounded-lg ml-auto" />
        </div>
      ))}
    </div>
  );
}

function AlumniContent() {
  const { user } = useAuth();
  const [searchParams, setSearchParams] = useState<AlumniSearchParams>({
    fn: '', dept: '', name: '', jobCat: '',
  });

  const { data, fetchNextPage, hasNextPage, isFetching, isLoading } = useInfiniteQuery({
    queryKey: ['alumni', 'search', searchParams],
    queryFn: ({ pageParam }) =>
      api.get<AlumniSearchResponse>(buildSearchUrl(searchParams, pageParam as number)),
    initialPageParam: 1,
    getNextPageParam: (lastPage) =>
      lastPage.page < lastPage.totalPages ? lastPage.page + 1 : undefined,
  });

  const { sentinelRef } = useInfiniteScroll({
    hasMore: !!hasNextPage,
    isFetching,
    onLoadMore: fetchNextPage,
  });

  const allItems: AlumniItem[] = data?.pages.flatMap((p) => p.items) ?? [];
  const totalCount = data?.pages[0]?.totalCount ?? 0;

  const handleSearch = (params: AlumniSearchParams) => {
    setSearchParams(params);
  };

  return (
    <div className="max-w-[1080px] mx-auto px-4">
      {/* Header */}
      <div className="pt-6 pb-4">
        <h1 className="text-2xl font-bold text-text-primary font-serif">동문찾기</h1>
        {totalCount > 0 && (
          <p className="text-sm text-text-tertiary mt-1">
            {totalCount.toLocaleString()}명의 동문
          </p>
        )}
      </div>

      {/* Sticky search filter */}
      <div className="sticky top-0 md:top-14 z-30 -mx-4 bg-background/95 backdrop-blur-lg px-4 pt-3 pb-3 shadow-xs">
        <SearchFilter onSearch={handleSearch} />
      </div>

      <div className="mt-4">
        {/* Initial loading skeleton */}
        {isLoading && <AlumniTableSkeleton />}

        {/* Empty state */}
        {!isLoading && allItems.length === 0 && (
          <div className="flex flex-col items-center justify-center py-16">
            <span className="text-4xl mb-3">🔍</span>
            <p className="text-sm text-text-tertiary">검색 결과가 없습니다</p>
          </div>
        )}

        {/* Table */}
        {allItems.length > 0 && (
          <div className="bg-surface border border-border rounded-2xl overflow-hidden">
            <TableColumnHeader />
            {allItems.map((item, i) => (
              <div
                key={item.fmSeq || `u${item.usrSeq}`}
                className={`animate-fade-in-up ${STAGGER_CLASSES[i % 5]}`}
              >
                <AlumniCard
                  item={item}
                  currentUsrSeq={user?.usrSeq}
                  isLast={i === allItems.length - 1}
                />
              </div>
            ))}
          </div>
        )}

        {/* Load-more spinner */}
        {isFetching && allItems.length > 0 && (
          <div className="flex justify-center py-4">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}

        {/* Infinite scroll sentinel */}
        <div ref={sentinelRef} className="h-1" />

        {/* Footer count */}
        {allItems.length > 0 && (
          <p className="mt-3 text-right text-xs text-text-placeholder">
            {allItems.length.toLocaleString()} / {totalCount.toLocaleString()}명 표시 중
          </p>
        )}
      </div>
    </div>
  );
}

export function AlumniPage() {
  return (
    <AuthGuard>
      <PageMeta title="동문 검색" noIndex />
      <AlumniContent />
    </AuthGuard>
  );
}
