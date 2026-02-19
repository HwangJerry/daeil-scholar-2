// AlumniPage — Alumni directory with search filters and vertical card list
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { AuthGuard } from '../components/auth/AuthGuard';
import { SearchFilter, type AlumniSearchParams } from '../components/alumni/SearchFilter';
import { AlumniCard } from '../components/alumni/AlumniCard';
import type { AlumniSearchResponse } from '../types/api';

function buildSearchUrl(params: AlumniSearchParams, page: number): string {
  const qs = new URLSearchParams();
  if (params.fn) qs.set('fn', params.fn);
  if (params.dept) qs.set('dept', params.dept);
  if (params.name) qs.set('name', params.name);
  if (params.company) qs.set('company', params.company);
  if (params.position) qs.set('position', params.position);
  if (params.jobCat) qs.set('jobCat', params.jobCat);
  qs.set('page', String(page));
  qs.set('size', '20');
  return `/api/alumni?${qs.toString()}`;
}

function AlumniContent() {
  const { user } = useAuth();
  const [searchParams, setSearchParams] = useState<AlumniSearchParams>({
    fn: '', dept: '', name: '', company: '', position: '', jobCat: '',
  });
  const [page, setPage] = useState(1);

  const { data, isFetching } = useQuery({
    queryKey: ['alumni', 'search', searchParams, page],
    queryFn: () => api.get<AlumniSearchResponse>(buildSearchUrl(searchParams, page)),
  });

  const handleSearch = (params: AlumniSearchParams) => {
    setSearchParams(params);
    setPage(1);
  };

  return (
    <div className="max-w-2xl mx-auto px-4">
      {/* Header */}
      <div className="pt-6 pb-2">
        <h1 className="text-xl font-bold text-text-primary font-serif">전체 보기</h1>
        <p className="text-sm text-text-tertiary mt-1">
          이름순으로 정렬된 모든 동문 리스트입니다.
        </p>
      </div>

      {/* Sticky search filter */}
      <div className="sticky top-0 md:top-14 z-30 -mx-4 bg-background/95 backdrop-blur-lg px-4 pt-3 pb-3 shadow-xs">
        <SearchFilter onSearch={handleSearch} />
      </div>

      <div className="space-y-4 mt-3">
        {isFetching && (
          <div className="flex justify-center py-8">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}

        {data && data.items.length === 0 && !isFetching && (
          <p className="py-12 text-center text-sm text-text-tertiary">검색 결과가 없습니다.</p>
        )}

        {data && data.items.length > 0 && (
          <>
            <div className="grid gap-3">
              {data.items.map((item) => (
                <AlumniCard key={item.fmSeq} item={item} currentUsrSeq={user?.usrSeq} />
              ))}
            </div>

            {data.totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 py-4">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="rounded-lg px-3 py-1.5 text-sm text-text-tertiary hover:bg-background disabled:opacity-40 transition-colors duration-150"
                >
                  이전
                </button>
                <span className="text-sm text-text-tertiary">
                  {page} / {data.totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(data.totalPages, p + 1))}
                  disabled={page >= data.totalPages}
                  className="rounded-lg px-3 py-1.5 text-sm text-text-tertiary hover:bg-background disabled:opacity-40 transition-colors duration-150"
                >
                  다음
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export function AlumniPage() {
  return (
    <AuthGuard>
      <AlumniContent />
    </AuthGuard>
  );
}
