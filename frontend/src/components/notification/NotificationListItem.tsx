// NotificationListItem — Single notification row with type icon, title, time, and read state
import { Mail, MessageCircle, Heart, CheckCircle, Bell } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { NotificationItem } from '../../api/notifications';

interface NotificationListItemProps {
  item: NotificationItem;
  onClick: (item: NotificationItem) => void;
}

const TYPE_ICON_MAP: Record<string, typeof Bell> = {
  NEW_MESSAGE: Mail,
  NEW_COMMENT: MessageCircle,
  NEW_LIKE: Heart,
  REGISTRATION_APPROVED: CheckCircle,
};

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const date = new Date(dateStr).getTime();
  const diffSec = Math.floor((now - date) / 1_000);

  if (diffSec < 60) return '방금 전';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}분 전`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}시간 전`;
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 30) return `${diffDay}일 전`;
  const diffMonth = Math.floor(diffDay / 30);
  if (diffMonth < 12) return `${diffMonth}개월 전`;
  return `${Math.floor(diffMonth / 12)}년 전`;
}

export function NotificationListItem({ item, onClick }: NotificationListItemProps) {
  const isUnread = item.readYn !== 'Y';
  const Icon = TYPE_ICON_MAP[item.anType] ?? Bell;

  return (
    <button
      type="button"
      onClick={() => onClick(item)}
      className={cn(
        'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-surface-hover',
        isUnread && 'bg-primary/5',
      )}
    >
      <div className="mt-0.5 flex-shrink-0 text-text-tertiary">
        <Icon size={18} strokeWidth={1.8} />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <p
            className={cn(
              'truncate text-sm',
              isUnread ? 'font-semibold text-text-primary' : 'text-text-muted',
            )}
          >
            {item.anTitle}
          </p>
          {isUnread && (
            <span className="h-1.5 w-1.5 flex-shrink-0 rounded-full bg-primary" />
          )}
        </div>
        {item.anBody && (
          <p className="mt-0.5 truncate text-xs text-text-placeholder">
            {item.anBody}
          </p>
        )}
        <p className="mt-1 text-[11px] text-text-placeholder">
          {formatRelativeTime(item.regDate)}
        </p>
      </div>
    </button>
  );
}
