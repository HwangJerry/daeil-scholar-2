// MessageInbox — Paginated list of received messages with read/delete actions
import { useState } from 'react';
import { Mail, MailOpen, Trash2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { BottomSheet } from '../ui/BottomSheet';
import { MessageDetailModal } from './MessageDetailModal';
import { MessageDetailContent } from './MessageDetailContent';
import { MessageRowSkeleton } from './MessageRowSkeleton';
import { useInboxMessages } from '../../hooks/useInboxMessages';
import { useMarkMessageRead } from '../../hooks/useMarkMessageRead';
import { useDeleteInboxMessage } from '../../hooks/useDeleteInboxMessage';
import { useResponsive } from '../../hooks/useResponsive';
import { formatMessageDate, truncateContent, CONTENT_TRUNCATE_LENGTH } from '../../utils/messageFormat';
import type { MessageItem } from '../../types/api';

function InboxMessageRow({ message }: { message: MessageItem }) {
  const isUnread = message.readYn !== 'Y';
  const [showDetail, setShowDetail] = useState(false);
  const { isMobile } = useResponsive();
  const markRead = useMarkMessageRead(message.amSeq);
  const deleteMsg = useDeleteInboxMessage(message.amSeq);

  const handleClick = () => {
    setShowDetail(true);
    if (isUnread && !markRead.isPending) {
      markRead.mutate();
    }
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deleteMsg.isPending) return;
    deleteMsg.mutate();
  };

  return (
    <>
      {showDetail && (
        isMobile ? (
          <BottomSheet maxHeight="full" onClose={() => setShowDetail(false)}>
            <MessageDetailContent message={message} type="inbox" onClose={() => setShowDetail(false)} />
          </BottomSheet>
        ) : (
          <MessageDetailModal message={message} type="inbox" onClose={() => setShowDetail(false)} />
        )
      )}
      <div
        onClick={handleClick}
        className={cn(
          'group relative rounded-2xl border p-4 transition-all duration-150 cursor-pointer',
          isUnread
            ? 'border-primary/20 bg-primary-light/20 hover:border-primary/30 border-l-2 border-l-primary'
            : 'border-border-subtle bg-surface hover:border-border',
          deleteMsg.isPending && 'opacity-40',
        )}
      >
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex-shrink-0">
            {isUnread ? (
              <Mail size={18} className="text-primary" />
            ) : (
              <MailOpen size={18} className="text-text-placeholder" />
            )}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className={cn('text-sm font-semibold font-serif', isUnread ? 'text-text-primary' : 'text-text-secondary')}>
                {message.senderName}
              </span>
              {isUnread && (
                <span className="inline-block h-1.5 w-1.5 rounded-full bg-primary flex-shrink-0" />
              )}
            </div>
            <p className={cn('text-sm leading-relaxed', isUnread ? 'text-text-secondary' : 'text-text-tertiary')}>
              {truncateContent(message.content, CONTENT_TRUNCATE_LENGTH)}
            </p>
            <span className="mt-1.5 block text-xs text-text-placeholder">
              {formatMessageDate(message.regDate)}
            </span>
          </div>

          <button
            onClick={handleDelete}
            disabled={deleteMsg.isPending}
            className="flex-shrink-0 rounded-lg p-1.5 text-text-placeholder opacity-0 group-hover:opacity-100 hover:text-error hover:bg-error-light transition-all duration-150"
            aria-label="삭제"
          >
            <Trash2 size={16} />
          </button>
        </div>
      </div>
    </>
  );
}

export function MessageInbox() {
  const [page, setPage] = useState(1);
  const { data, isFetching, isLoading, isError } = useInboxMessages(page);

  return (
    <div className="space-y-3">
      {isLoading && (
        <div className="space-y-2">
          {[0, 1, 2].map((i) => (
            <div key={i} className="animate-fade-in-up" style={{ animationDelay: `${i * 0.05}s` }}>
              <MessageRowSkeleton />
            </div>
          ))}
        </div>
      )}

      {!isLoading && isFetching && (
        <div className="flex justify-center py-4">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      )}

      {isError && !isLoading && (
        <div className="flex flex-col items-center justify-center py-16">
          <p className="text-sm text-error">쪽지를 불러올 수 없습니다. 잠시 후 다시 시도해주세요.</p>
        </div>
      )}

      {!isLoading && !isError && data && data.items.length === 0 && !isFetching && (
        <div className="flex flex-col items-center justify-center py-16">
          <Mail size={40} className="text-text-placeholder mb-3" />
          <p className="text-sm text-text-tertiary">받은 쪽지가 없습니다</p>
        </div>
      )}

      {!isLoading && !isError && data && data.items.length > 0 && (
        <>
          <div className="space-y-2">
            {data.items.map((msg) => (
              <InboxMessageRow key={msg.amSeq} message={msg} />
            ))}
          </div>

          {data.totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 py-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="rounded-lg px-3 py-1.5 text-sm text-text-tertiary hover:bg-background disabled:opacity-40 transition-colors duration-150"
              >
                이전
              </button>
              <span className="text-sm text-text-tertiary">{page} / {data.totalPages}</span>
              <button
                onClick={() => setPage((p) => Math.min(data.totalPages, p + 1))}
                disabled={page >= data.totalPages}
                className="rounded-lg px-3 py-1.5 text-sm text-text-tertiary hover:bg-background disabled:opacity-40 transition-colors duration-150"
              >
                다음
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
