// MessageOutbox — Paginated list of sent messages with delete action
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Send, Trash2 } from 'lucide-react';
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

function OutboxMessageRow({ message }: { message: MessageItem }) {
  const queryClient = useQueryClient();
  const isRead = message.readYn === 'Y';

  const deleteMutation = useMutation({
    mutationFn: () => api.del(`/api/messages/${message.amSeq}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages', 'outbox'] });
    },
  });

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deleteMutation.isPending) return;
    deleteMutation.mutate();
  };

  return (
    <div
      className={`group relative rounded-xl border border-border-subtle bg-surface p-4 transition-all duration-150 ${
        deleteMutation.isPending ? 'opacity-40' : ''
      }`}
    >
      <div className="flex items-start gap-3">
        {/* Icon */}
        <div className="mt-0.5 flex-shrink-0">
          <Send size={18} className="text-text-placeholder" />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-sm font-medium text-text-secondary">
              {message.recvrName}
            </span>
            <span
              className={`text-xs px-1.5 py-0.5 rounded-md ${
                isRead
                  ? 'bg-emerald-50 text-emerald-700'
                  : 'bg-background text-text-placeholder'
              }`}
            >
              {isRead ? '읽음' : '안읽음'}
            </span>
          </div>
          <p className="text-sm leading-relaxed text-text-tertiary">
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

export function MessageOutbox() {
  const [page, setPage] = useState(1);

  const { data, isFetching } = useQuery({
    queryKey: ['messages', 'outbox', page],
    queryFn: () =>
      api.get<MessageListResponse>(
        `/api/messages/outbox?page=${page}&size=${PAGE_SIZE}`,
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
          <Send size={40} className="text-text-placeholder mb-3" />
          <p className="text-sm text-text-tertiary">보낸 쪽지가 없습니다</p>
        </div>
      )}

      {/* Message list */}
      {data && data.items.length > 0 && (
        <>
          <div className="space-y-2">
            {data.items.map((msg) => (
              <OutboxMessageRow key={msg.amSeq} message={msg} />
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
