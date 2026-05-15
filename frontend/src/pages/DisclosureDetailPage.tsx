// DisclosureDetailPage — 공익법인 의무공시 상세 (본문 + 첨부 다운로드)
import { useParams, Link } from 'react-router-dom';
import { Eye, ArrowLeft } from 'lucide-react';
import { useDisclosureDetail } from '../hooks/useDisclosureDetail';
import { PageMeta } from '../components/seo/PageMeta';
import { HtmlContent } from '../components/common/HtmlContent';
import { FileAttachments } from '../components/post/FileAttachments';
import Footer from '../components/layout/Footer';

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}.${m}.${day}`;
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-3xl space-y-4 rounded-xl bg-surface p-6 shadow-card border border-border-subtle">
      <div className="h-6 w-3/4 rounded skeleton-shimmer" />
      <div className="h-4 w-1/2 rounded skeleton-shimmer" />
      <div className="space-y-2">
        <div className="h-4 rounded skeleton-shimmer" />
        <div className="h-4 rounded skeleton-shimmer" />
        <div className="h-4 w-2/3 rounded skeleton-shimmer" />
      </div>
    </div>
  );
}

export function DisclosureDetailPage() {
  const { seq } = useParams<{ seq: string }>();
  const { data: post, isLoading, isError } = useDisclosureDetail(seq);

  if (isLoading) {
    return (
      <div className="px-4 py-6 pb-20">
        <DetailSkeleton />
      </div>
    );
  }

  if (isError || !post) {
    return (
      <div className="px-4 py-6 pb-20">
        <div className="mx-auto max-w-3xl rounded-xl bg-surface p-6 text-center shadow-card border border-border-subtle">
          <p className="text-sm text-text-tertiary">공시 자료를 찾을 수 없습니다.</p>
          <Link to="/disclosure" className="mt-3 inline-block text-sm text-primary hover:underline">
            의무공시 목록으로
          </Link>
        </div>
      </div>
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
      <div className="px-4 py-6 pb-20 animate-fade-in-up">
        <article className="mx-auto max-w-3xl overflow-hidden rounded-xl bg-surface shadow-md border-transparent">
          <div className="p-5 md:p-6">
            <header className="mb-5">
              <h1 className="mb-3 text-xl font-bold font-serif text-text-primary leading-snug">
                {post.subject}
              </h1>
              <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-text-tertiary">
                <span className="text-text-secondary font-medium">{post.regName}</span>
                <span>·</span>
                <span>{formatDate(post.regDate)}</span>
                <span>·</span>
                <span className="inline-flex items-center gap-1">
                  <Eye className="h-3.5 w-3.5" />
                  {post.hit}
                </span>
              </div>
            </header>

            <HtmlContent html={post.contentHtml} className="text-text-secondary" />
            <FileAttachments files={post.files} />

            <div className="mt-8 border-t border-border-subtle pt-4">
              <Link
                to="/disclosure"
                className="inline-flex items-center gap-1.5 text-sm text-text-tertiary hover:text-primary transition-colors"
              >
                <ArrowLeft className="h-4 w-4" />
                목록으로
              </Link>
            </div>
          </div>
        </article>
        <Footer />
      </div>
    </>
  );
}
