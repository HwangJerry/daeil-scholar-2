// DisclosureDetailPage — Editorial disclosure document with download-focused attachments
import { ArrowLeft, Eye, FileQuestion } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';
import { HtmlContent } from '../components/common/HtmlContent';
import { DisclosureAttachmentList } from '../components/disclosure/DisclosureAttachmentList';
import { DisclosurePageHeader } from '../components/disclosure/DisclosurePageHeader';
import { PageMeta } from '../components/seo/PageMeta';
import { Bone } from '../components/ui/Skeleton';
import { useDisclosureDetail } from '../hooks/useDisclosureDetail';
import { formatAbsoluteDate } from '../utils/date';

function DisclosureDetailSkeleton() {
  return (
    <div role="status" aria-label="의무공시 상세 불러오는 중">
      <div className="border-b border-border-subtle bg-surface px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <div className="mx-auto max-w-[1080px] space-y-5">
          <Bone className="h-4 w-32" />
          <Bone className="h-3 w-36" />
          <Bone className="h-12 w-3/4 max-w-2xl" />
          <Bone className="h-4 w-52" />
        </div>
      </div>
      <div className="px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <div className="mx-auto max-w-4xl space-y-4">
          <Bone className="h-5 w-full" />
          <Bone className="h-5 w-4/5" />
          <Bone className="h-5 w-2/3" />
        </div>
      </div>
    </div>
  );
}

function DisclosureDetailError() {
  return (
    <>
      <DisclosurePageHeader
        title="공시 자료를 찾을 수 없습니다"
        description="요청한 공시 자료가 없거나 일시적으로 불러올 수 없습니다."
      />
      <div className="bg-background px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <div className="mx-auto flex max-w-[1080px] flex-col items-center border-y border-border px-6 py-14 text-center">
          <span className="inline-flex size-12 items-center justify-center rounded-full bg-primary-light text-primary">
            <FileQuestion aria-hidden="true" className="size-5" />
          </span>
          <p className="mt-4 max-w-sm text-body-sm leading-relaxed text-text-tertiary">
            주소를 다시 확인하거나 의무공시 목록에서 원하는 자료를 선택해 주세요.
          </p>
          <Link
            to="/disclosure"
            className="mt-6 inline-flex min-h-11 items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            의무공시 목록
          </Link>
        </div>
      </div>
    </>
  );
}

export function DisclosureDetailPage() {
  const { seq } = useParams<{ seq: string }>();
  const { data: post, isLoading, isError } = useDisclosureDetail(seq);

  if (isLoading) {
    return (
      <>
        <PageMeta title="의무공시 자료" canonicalPath={`/disclosure/${seq ?? ''}`} />
        <DisclosureDetailSkeleton />
      </>
    );
  }

  if (isError || !post) {
    return (
      <>
        <PageMeta title="공시 자료를 찾을 수 없습니다" canonicalPath="/disclosure" />
        <DisclosureDetailError />
      </>
    );
  }

  return (
    <>
      <PageMeta
        title={post.subject}
        description={post.summary || undefined}
        canonicalPath={`/disclosure/${post.seq}`}
        ogType="article"
        breadcrumbs={[
          { name: '의무공시', url: '/disclosure' },
          { name: post.subject, url: `/disclosure/${post.seq}` },
        ]}
      />

      <header className="border-b border-border-subtle bg-surface px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <div className="mx-auto max-w-[1080px]">
          <Link
            to="/disclosure"
            className="inline-flex min-h-11 items-center gap-2 rounded-md text-body-sm font-medium text-text-tertiary transition-colors hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface"
          >
            <ArrowLeft aria-hidden="true" className="size-4" />
            의무공시 목록
          </Link>
          <p className="mt-8 text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder">
            Public Disclosure
          </p>
          <h1 className="mt-3 max-w-4xl font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl">
            {post.subject}
          </h1>
          <dl className="mt-6 flex flex-wrap items-center gap-x-4 gap-y-2 text-caption text-text-tertiary">
            <div>
              <dt className="sr-only">작성자</dt>
              <dd className="font-medium text-text-secondary">{post.regName}</dd>
            </div>
            <div>
              <dt className="sr-only">등록일</dt>
              <dd>
                <time dateTime={post.regDate}>{formatAbsoluteDate(post.regDate)}</time>
              </dd>
            </div>
            <div className="inline-flex items-center gap-1.5">
              <dt className="sr-only">조회수</dt>
              <Eye aria-hidden="true" className="size-3.5" />
              <dd>{post.hit}</dd>
            </div>
          </dl>
        </div>
      </header>

      <section aria-label="공시 자료 본문" className="bg-background px-5 py-14 sm:px-8 md:px-6 md:py-20">
        <article className="mx-auto max-w-4xl">
          <HtmlContent
            html={post.contentHtml}
            className="prose-headings:font-serif prose-headings:text-text-primary prose-p:leading-8 prose-p:text-text-secondary"
          />

          <DisclosureAttachmentList files={post.files} />

          <div className="mt-14 border-t border-border pt-8">
            <Link
              to="/disclosure"
              className="inline-flex min-h-11 items-center gap-2 rounded-md text-sm font-medium text-text-secondary transition-colors hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
            >
              <ArrowLeft aria-hidden="true" className="size-4" />
              의무공시 목록으로 돌아가기
            </Link>
          </div>
        </article>
      </section>
    </>
  );
}
