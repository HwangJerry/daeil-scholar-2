// PostHeader — 게시글 제목 및 메타데이터 표시 (피드 카드 톤앤매너 통일)
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
    <header className="mb-5">
      <h1 className="mb-3 text-xl font-bold font-serif text-text-primary leading-snug">{subject}</h1>
      <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-text-tertiary">
        <span className="text-text-secondary font-medium">{regName}</span>
        <span>·</span>
        <span>{formatDate(regDate)}</span>
        <span>·</span>
        <span className="inline-flex items-center gap-1">
          <Eye className="h-3.5 w-3.5" />
          {hit}
        </span>
        <span>·</span>
        <span className="inline-flex items-center gap-1">
          <Heart className="h-3.5 w-3.5" />
          {likeCnt}
        </span>
        <span>·</span>
        <span className="inline-flex items-center gap-1">
          <MessageCircle className="h-3.5 w-3.5" />
          {commentCnt}
        </span>
      </div>
    </header>
  );
}
