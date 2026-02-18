// PostEngagement — Like button and comment section for a post
import { LikeButton } from './LikeButton';
import { CommentSection } from './CommentSection';

interface PostEngagementProps {
  seq: number;
  userLiked: boolean;
  likeCnt: number;
}

export function PostEngagement({ seq, userLiked, likeCnt }: PostEngagementProps) {
  return (
    <>
      <div className="mt-6 flex items-center gap-3">
        <LikeButton seq={seq} liked={userLiked} likeCnt={likeCnt} />
      </div>

      <CommentSection seq={seq} />
    </>
  );
}
