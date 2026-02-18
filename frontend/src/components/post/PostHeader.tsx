// PostHeader — 게시글 제목 및 메타데이터 표시
import { Eye, Heart, MessageCircle } from 'lucide-react';

interface PostHeaderProps {
  subject: string;
  regName: string;
  regDate: string;
  hit: number;
  likeCnt: number;
  commentCnt: number;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}.${m}.${day}`;
}

export function PostHeader({ subject, regName, regDate, hit, likeCnt, commentCnt }: PostHeaderProps) {
  return (
    <header className="mb-4">
      <h1 className="mb-2 text-xl font-bold text-text-primary">{subject}</h1>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-placeholder">
        <span>{regName}</span>
        <span>{formatDate(regDate)}</span>
        <span className="inline-flex items-center gap-1">
          <Eye className="h-3.5 w-3.5" />
          {hit}
        </span>
        {likeCnt > 0 && (
          <span className="inline-flex items-center gap-1">
            <Heart className="h-3.5 w-3.5" />
            {likeCnt}
          </span>
        )}
        {commentCnt > 0 && (
          <span className="inline-flex items-center gap-1">
            <MessageCircle className="h-3.5 w-3.5" />
            {commentCnt}
          </span>
        )}
      </div>
    </header>
  );
}
