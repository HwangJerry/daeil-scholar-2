// DisclosureListPage — Editorial public-disclosure archive with complete query states
import { AlertCircle, ArrowRight, Files, LoaderCircle } from 'lucide-react';
import { DisclosureListItem } from '../components/disclosure/DisclosureListItem';
import { DisclosurePageHeader } from '../components/disclosure/DisclosurePageHeader';
import { DisclosureStatePanel } from '../components/disclosure/DisclosureStatePanel';
import { PageMeta } from '../components/seo/PageMeta';
import { Bone } from '../components/ui/Skeleton';
import { Button } from '../components/ui/Button';
import { useDisclosureList } from '../hooks/useDisclosureList';

const DISCLOSURE_SKELETON_COUNT = 5;

function DisclosureListSkeleton() {
  return (
    <div role="status" aria-label="의무공시 목록 불러오는 중">
      {Array.from({ length: DISCLOSURE_SKELETON_COUNT }, (_, index) => (
        <div
          key={index}
          className="grid min-h-32 gap-5 border-b border-border py-7 last:border-b-0 md:grid-cols-[minmax(0,1fr)_12rem_2.75rem] md:items-center md:gap-10"
        >
          <div className="space-y-3">
            <Bone className="h-2.5 w-28" />
            <Bone className="h-7 w-2/3" />
            <Bone className="h-3.5 w-1/2" />
          </div>
          <Bone className="h-3.5 w-36" />
          <Bone className="hidden size-11 rounded-full md:block" />
        </div>
      ))}
    </div>
  );
}

export function DisclosureListPage() {
  const {
    items,
    hasMore,
    isError,
    isFetching,
    isLoading,
    loadMore,
    refetch,
  } = useDisclosureList();

  const hasItems = items.length > 0;
  const archiveCountLabel = (() => {
    if (!hasItems) return '공시 자료를 확인해 주세요.';
    if (hasMore) return `현재 ${items.length}개의 공시 자료`;
    return `총 ${items.length}개의 공시 자료`;
  })();

  return (
    <>
      <PageMeta
        title="의무공시"
        description="대일외국어고등학교 장학회의 공익법인 의무공시 자료를 확인하세요."
        canonicalPath="/disclosure"
      />
      <DisclosurePageHeader
        title="의무공시"
        description="장학회의 운영과 재정 정보를 투명하게 공개합니다. 연도별 공시 자료와 원본 문서를 확인할 수 있습니다."
      />

      <section
        aria-labelledby="disclosure-archive-heading"
        className="bg-background px-5 py-14 sm:px-8 md:px-6 md:py-20"
      >
        <div className="mx-auto max-w-[1080px]">
          <header className="mb-8 flex flex-col gap-3 border-b border-border pb-7 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-2xs font-semibold uppercase tracking-[0.2em] text-text-placeholder">
                Archive
              </p>
              <h2
                id="disclosure-archive-heading"
                className="mt-2 font-serif text-2xl font-semibold tracking-tight text-text-primary sm:text-3xl"
              >
                연도별 공개 자료
              </h2>
            </div>
            <p className="text-body-sm text-text-tertiary">{archiveCountLabel}</p>
          </header>

          {isLoading && <DisclosureListSkeleton />}

          {!isLoading && isError && !hasItems && (
            <DisclosureStatePanel
              icon={AlertCircle}
              title="공시 자료를 불러오지 못했습니다"
              description="네트워크 연결을 확인한 뒤 다시 시도해 주세요."
              onRetry={() => void refetch()}
            />
          )}

          {!isLoading && !isError && !hasItems && (
            <DisclosureStatePanel
              icon={Files}
              title="등록된 공시 자료가 없습니다"
              description="새로운 공시 자료가 등록되면 이곳에서 확인할 수 있습니다."
            />
          )}

          {hasItems && (
            <>
              <ol aria-label="의무공시 자료 목록">
                {items.map((item) => (
                  <DisclosureListItem key={item.seq} item={item} />
                ))}
              </ol>

              {isError && (
                <div
                  role="alert"
                  className="mt-6 flex flex-col gap-3 rounded-lg border border-error-border bg-error-subtle px-4 py-3 text-body-sm text-error-text sm:flex-row sm:items-center sm:justify-between"
                >
                  <span>추가 공시 자료를 불러오지 못했습니다.</span>
                  <Button type="button" variant="outline" size="sm" onClick={() => void refetch()}>
                    다시 시도
                  </Button>
                </div>
              )}

              {hasMore && !isError && (
                <div className="mt-10 flex justify-center">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void loadMore()}
                    disabled={isFetching}
                    className="min-h-11 min-w-40"
                  >
                    {isFetching ? (
                      <>
                        <LoaderCircle aria-hidden="true" className="size-4 animate-spin" />
                        불러오는 중
                      </>
                    ) : (
                      <>
                        공시 자료 더 보기
                        <ArrowRight aria-hidden="true" className="size-4" />
                      </>
                    )}
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </section>
    </>
  );
}
