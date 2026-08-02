// PostContent — Public notice composition without member engagement controls
import type { NoticeDetail } from '../../types/api';
import { PostHeader } from './PostHeader';
import { PostBody } from './PostBody';

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
      />

      <PostBody
        contentHtml={post.contentHtml}
        files={post.files}
      />

    </div>
  );
}
