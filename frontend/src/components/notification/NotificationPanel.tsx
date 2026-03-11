// NotificationPanel — Dropdown panel showing recent notifications with mark-all-read
import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  markNotificationRead,
  markAllNotificationsRead,
} from '../../api/notifications';
import type { NotificationItem } from '../../api/notifications';
import { useNotificationList } from '../../hooks/useNotificationList';
import { NotificationListItem } from './NotificationListItem';

interface NotificationPanelProps {
  onClose: () => void;
}

export function NotificationPanel({ onClose }: NotificationPanelProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data, isLoading } = useNotificationList(1, 20);

  const markReadMutation = useMutation({
    mutationFn: (seq: number) => markNotificationRead(seq),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['badges'] });
    },
  });

  const markAllReadMutation = useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['badges'] });
    },
  });

  const handleNotificationClick = useCallback(
    (item: NotificationItem) => {
      if (item.readYn !== 'Y') {
        markReadMutation.mutate(item.anSeq);
      }

      onClose();

      const routeByType: Record<string, string> = {
        NEW_MESSAGE: '/messages',
        NEW_COMMENT: `/post/${item.anRefSeq}`,
        NEW_LIKE: `/post/${item.anRefSeq}`,
      };

      const target = routeByType[item.anType];
      if (target) {
        navigate(target);
      }
    },
    [markReadMutation, navigate, onClose],
  );

  const handleMarkAllRead = () => {
    markAllReadMutation.mutate();
  };

  const items = data?.items ?? [];
  const hasUnread = (data?.unreadCount ?? 0) > 0;

  return (
    <div className="absolute right-0 top-full mt-2 w-80 rounded-xl border border-border-subtle bg-surface shadow-lg z-50">
      <div className="flex items-center justify-between border-b border-border-subtle px-4 py-2.5">
        <h3 className="text-sm font-semibold text-text-primary">알림</h3>
        {hasUnread && (
          <button
            type="button"
            onClick={handleMarkAllRead}
            disabled={markAllReadMutation.isPending}
            className="text-xs text-primary hover:text-primary/80 transition-colors disabled:opacity-50"
          >
            모두 읽음
          </button>
        )}
      </div>

      <div className="max-h-96 overflow-y-auto">
        {isLoading && (
          <div className="px-4 py-8 text-center text-sm text-text-placeholder">
            불러오는 중...
          </div>
        )}

        {!isLoading && items.length === 0 && (
          <div className="px-4 py-8 text-center text-sm text-text-placeholder">
            알림이 없습니다.
          </div>
        )}

        {items.map((item) => (
          <NotificationListItem
            key={item.anSeq}
            item={item}
            onClick={handleNotificationClick}
          />
        ))}
      </div>
    </div>
  );
}
