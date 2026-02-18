// Feed card for a single notice item with thumbnail, summary, and metadata
import { Eye } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '../ui/Badge';
import type { NoticeItem } from '../../types/api';

export function NoticeCard({ item }: { item: NoticeItem }) {
  return (
    <Link
      to={`/post/${item.seq}`}
      className="group block overflow-hidden rounded-xl bg-surface border border-border-subtle shadow-card transition-all duration-200 hover:shadow-card-hover hover:-translate-y-0.5"
    >
      {item.thumbnailUrl && (
        <div className="overflow-hidden">
          <img
            src={item.thumbnailUrl}
            alt={item.subject}
            className="h-40 w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
            loading="lazy"
          />
        </div>
      )}
      <div className="p-4 pt-3">
        {item.isPinned === 'Y' && (
          <Badge variant="pinned" className="mb-1.5">고정</Badge>
        )}
        <h3 className="mb-1 line-clamp-2 text-[15px] font-semibold text-text-primary group-hover:text-primary">
          {item.subject}
        </h3>
        {item.summary && (
          <p className="mb-2.5 line-clamp-2 text-[13px] text-text-tertiary leading-relaxed">{item.summary}</p>
        )}
        <div className="flex items-center gap-3 text-xs text-text-placeholder">
          <span>{item.regDate}</span>
          <span>{item.regName}</span>
          <span className="ml-auto flex items-center gap-2">
            <span className="flex items-center gap-0.5"><Eye className="h-3.5 w-3.5" />{item.hit}</span>
            {item.likeCnt > 0 && <span>좋아요 {item.likeCnt}</span>}
            {item.commentCnt > 0 && <span>댓글 {item.commentCnt}</span>}
          </span>
        </div>
      </div>
    </Link>
  );
}
