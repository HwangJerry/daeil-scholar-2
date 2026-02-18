// PostContent — Composition layer delegating to PostHeader, PostBody, PostEngagement
import type { NoticeDetail } from '../../types/api';
import { PostHeader } from './PostHeader';
import { PostBody } from './PostBody';
import { PostEngagement } from './PostEngagement';

interface PostContentProps {
  post: NoticeDetail;
}

export function PostContent({ post }: PostContentProps) {
  return (
    <div className="p-5 md:p-6">
      <PostHeader
        subject={post.subject}
        regName={post.regName}
        regDate={post.regDate}
        hit={post.hit}
        likeCnt={post.likeCnt}
        commentCnt={post.commentCnt}
      />

      <PostBody
        thumbnailUrl={post.thumbnailUrl}
        subject={post.subject}
        contentHtml={post.contentHtml}
        files={post.files}
      />

      <PostEngagement
        seq={post.seq}
        userLiked={post.userLiked}
        likeCnt={post.likeCnt}
      />
    </div>
  );
}
