// MessageInbox — Paginated list of received messages with read/delete actions
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Mail, MailOpen, Trash2 } from 'lucide-react';
import { api } from '../../api/client';
import type { MessageItem, MessageListResponse } from '../../types/api';

const PAGE_SIZE = 20;

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}.${month}.${day}`;
}

function truncateContent(content: string, maxLength: number): string {
  if (content.length <= maxLength) return content;
  return content.slice(0, maxLength) + '...';
}

const CONTENT_TRUNCATE_LENGTH = 80;

function InboxMessageRow({ message }: { message: MessageItem }) {
  const queryClient = useQueryClient();
  const isUnread = message.readYn !== 'Y';

  const markReadMutation = useMutation({
    mutationFn: () => api.put(`/api/messages/${message.amSeq}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
      queryClient.invalidateQueries({ queryKey: ['messages', 'unread-count'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.del(`/api/messages/${message.amSeq}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'inbox'] });
      queryClient.invalidateQueries({ queryKey: ['messages', 'unread-count'] });
    },
  });

  const handleClick = () => {
    if (isUnread && !markReadMutation.isPending) {
      markReadMutation.mutate();
    }
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deleteMutation.isPending) return;
    deleteMutation.mutate();
  };

  return (
    <div
      onClick={handleClick}
      className={`group relative rounded-xl border p-4 transition-all duration-150 ${
        isUnread
          ? 'border-primary/20 bg-primary-light/30 cursor-pointer hover:border-primary/30'
          : 'border-border-subtle bg-surface'
      } ${deleteMutation.isPending ? 'opacity-40' : ''}`}
    >
      <div className="flex items-start gap-3">
        {/* Read status icon */}
        <div className="mt-0.5 flex-shrink-0">
          {isUnread ? (
            <Mail size={18} className="text-primary" />
          ) : (
            <MailOpen size={18} className="text-text-placeholder" />
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span
              className={`text-sm font-medium ${
                isUnread ? 'text-text-primary' : 'text-text-secondary'
              }`}
            >
              {message.senderName}
            </span>
            {isUnread && (
              <span className="inline-block h-1.5 w-1.5 rounded-full bg-primary flex-shrink-0" />
            )}
          </div>
          <p
            className={`text-sm leading-relaxed ${
              isUnread ? 'text-text-secondary' : 'text-text-tertiary'
            }`}
          >
            {truncateContent(message.content, CONTENT_TRUNCATE_LENGTH)}
          </p>
          <span className="mt-1.5 block text-xs text-text-placeholder">
            {formatDate(message.regDate)}
          </span>
        </div>

        {/* Delete button */}
        <button
          onClick={handleDelete}
          disabled={deleteMutation.isPending}
          className="flex-shrink-0 rounded-lg p-1.5 text-text-placeholder opacity-0 group-hover:opacity-100 hover:text-error hover:bg-error-light transition-all duration-150"
          aria-label="삭제"
        >
          <Trash2 size={16} />
        </button>
      </div>
    </div>
  );
}

export function MessageInbox() {
  const [page, setPage] = useState(1);

  const { data, isFetching } = useQuery({
    queryKey: ['messages', 'inbox', page],
    queryFn: () =>
      api.get<MessageListResponse>(
        `/api/messages/inbox?page=${page}&size=${PAGE_SIZE}`,
      ),
  });

  return (
    <div className="space-y-3">
      {/* Loading */}
      {isFetching && (
        <div className="flex justify-center py-8">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      )}

      {/* Empty state */}
      {data && data.items.length === 0 && !isFetching && (
        <div className="flex flex-col items-center justify-center py-16">
          <Mail size={40} className="text-text-placeholder mb-3" />
          <p className="text-sm text-text-tertiary">받은 쪽지가 없습니다</p>
        </div>
      )}

      {/* Message list */}
      {data && data.items.length > 0 && (
        <>
          <div className="space-y-2">
            {data.items.map((msg) => (
              <InboxMessageRow key={msg.amSeq} message={msg} />
            ))}
          </div>

          {/* Pagination */}
          {data.totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 py-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="rounded-lg px-3 py-1.5 text-sm text-text-tertiary hover:bg-background disabled:opacity-40 transition-colors duration-150"
              >
                이전
              </button>
              <span className="text-sm text-text-tertiary">
                {page} / {data.totalPages}
              </span>
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
