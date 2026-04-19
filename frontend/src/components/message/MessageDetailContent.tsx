// MessageDetailContent — Inner message detail view, reusable inside Modal or BottomSheet
import { cn } from '../../lib/utils';
import type { MessageItem } from '../../types/api';

interface MessageDetailContentProps {
  message: MessageItem;
  type: 'inbox' | 'outbox';
  onClose: () => void;
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}.${month}.${day}`;
}

export function MessageDetailContent({ message, type, onClose }: MessageDetailContentProps) {
  const isRead = message.readYn === 'Y';

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <span className="text-xs text-text-placeholder">
              {type === 'inbox' ? '보낸 사람' : '받는 사람'}
            </span>
            <span className="text-sm font-semibold font-serif text-text-primary">
              {type === 'inbox' ? message.senderName : message.recvrName}
            </span>
            {type === 'outbox' && (
              <span
                className={cn(
                  'text-xs px-1.5 py-0.5 rounded-md',
                  isRead
                    ? 'bg-success-subtle text-success-text'
                    : 'bg-background text-text-placeholder',
                )}
              >
                {isRead ? '읽음' : '안읽음'}
              </span>
            )}
          </div>
          <span className="text-xs text-text-placeholder">
            {formatDate(message.regDate)}
          </span>
        </div>
      </div>

      {/* Divider */}
      <div className="border-t border-border-subtle" />

      {/* Content */}
      <p className="text-sm leading-relaxed text-text-secondary whitespace-pre-wrap min-h-[80px]">
        {message.content}
      </p>

      {/* Footer */}
      <div className="flex justify-end pt-2">
        <button
          onClick={onClose}
          className="rounded-xl px-4 py-2 text-sm font-medium bg-background text-text-secondary hover:bg-surface-hover transition-colors duration-150"
        >
          닫기
        </button>
      </div>
    </div>
  );
}
