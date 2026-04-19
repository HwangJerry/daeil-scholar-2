// MessageDetailModal — Modal wrapper for message detail content
import { Modal } from '../ui/Modal';
import { MessageDetailContent } from './MessageDetailContent';
import type { MessageItem } from '../../types/api';

interface MessageDetailModalProps {
  message: MessageItem;
  type: 'inbox' | 'outbox';
  onClose: () => void;
}

export function MessageDetailModal({ message, type, onClose }: MessageDetailModalProps) {
  return (
    <Modal onClose={onClose} maxWidth="max-w-lg">
      <MessageDetailContent message={message} type={type} onClose={onClose} />
    </Modal>
  );
}
