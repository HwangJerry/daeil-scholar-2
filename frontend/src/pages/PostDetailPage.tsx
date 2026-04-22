// PostDetailPage — 게시글 상세 페이지 오케스트레이터
import { useParams, Link } from 'react-router-dom';
import { usePostDetail } from '../hooks/usePostDetail';
import { PageMeta } from '../components/seo/PageMeta';
import { PostContent } from '../components/post/PostContent';
import { PostNavigation } from '../components/post/PostNavigation';

function PostDetailSkeleton() {
  return (
    <div className="mx-auto max-w-2xl space-y-4 rounded-xl bg-surface p-6 shadow-card border border-border-subtle">
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

export function PostDetailPage() {
  const { seq } = useParams<{ seq: string }>();
  const { data: post, isLoading, isError } = usePostDetail(seq);

  if (isLoading) {
    return <PostDetailSkeleton />;
  }

  if (isError || !post) {
    return (
      <div className="mx-auto max-w-2xl rounded-xl bg-surface p-6 text-center shadow-card border border-border-subtle">
        <p className="text-sm text-text-tertiary">게시글을 찾을 수 없습니다.</p>
        <Link to="/" className="mt-3 inline-block text-sm text-primary hover:underline">
          피드로 돌아가기
        </Link>
      </div>
    );
  }

  return (
    <>
      <PageMeta
        title={post.subject}
        description={post.summary || undefined}
        ogImage={post.thumbnailUrl || undefined}
        canonicalPath={`/post/${post.seq}`}
        ogType="article"
        articleData={{ headline: post.subject, publishedAt: post.regDate }}
        breadcrumbs={[
          { name: '홈', url: '/' },
          { name: post.subject, url: `/post/${post.seq}` },
        ]}
      />
      <article className="mx-auto max-w-2xl overflow-hidden rounded-xl bg-surface shadow-md border-transparent animate-fade-in-up">
        <PostContent post={post} />
        <PostNavigation currentSeq={post.seq} />
      </article>
    </>
  );
}
