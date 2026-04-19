// MessageBubble — Single chat bubble aligned by sender/receiver with timestamp
import type { MessageItem } from '../../types/api';
import { formatChatTime } from '../../utils/messageFormat';

interface MessageBubbleProps {
  message: MessageItem;
  currentUserSeq: number;
}

export function MessageBubble({ message, currentUserSeq }: MessageBubbleProps) {
  const isSent = message.senderSeq === currentUserSeq;

  return (
    <div className={`flex ${isSent ? 'justify-end' : 'justify-start'} mb-1.5`}>
      <div className={`max-w-[75%] flex flex-col ${isSent ? 'items-end' : 'items-start'}`}>
        <div
          className={`px-3 py-2 rounded-2xl text-sm leading-relaxed whitespace-pre-wrap break-words ${
            isSent
              ? 'bg-primary text-white rounded-br-none'
              : 'bg-background text-text-primary rounded-bl-none border border-border-subtle'
          }`}
        >
          {message.content}
        </div>
        <div className="flex items-center gap-1 mt-0.5">
          {isSent && (
            <span className="text-[10px] text-text-placeholder">
              {message.readYn === 'Y' ? '읽음' : '1'}
            </span>
          )}
          <span className="text-[10px] text-text-placeholder">{formatChatTime(message.regDate)}</span>
        </div>
      </div>
    </div>
  );
}
