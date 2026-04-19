// ConversationItem — Single conversation row showing contact, last message, and unread indicator (IG DM style)
import type { ConversationSummary } from '../../types/api';
import { formatMessageDate } from '../../utils/messageFormat';

interface ConversationItemProps {
  conversation: ConversationSummary;
  isActive: boolean;
  onSelect: (seq: number) => void;
}

export function ConversationItem({ conversation, isActive, onSelect }: ConversationItemProps) {
  const initial = conversation.otherName.charAt(0) || '?';
  const hasUnread = conversation.unreadCount > 0;

  return (
    <button
      type="button"
      onClick={() => onSelect(conversation.otherSeq)}
      className={`w-full flex items-center gap-3 px-4 py-3 text-left transition-colors duration-150 ${
        isActive ? 'bg-primary-light' : 'hover:bg-primary-light/50'
      }`}
    >
      {/* Avatar with unread dot indicator */}
      <div className="relative flex-shrink-0">
        <div className="w-12 h-12 rounded-full bg-primary ring-2 ring-surface flex items-center justify-center">
          <span className="text-sm font-bold text-white">{initial}</span>
        </div>
        {hasUnread && (
          <span className="absolute bottom-0.5 right-0.5 w-3 h-3 rounded-full bg-primary border-2 border-surface" />
        )}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span className="text-sm font-semibold truncate font-serif text-text-primary">
            {conversation.otherName}
          </span>
          <span className="text-[11px] text-text-placeholder flex-shrink-0">
            {formatMessageDate(conversation.lastDate)}
          </span>
        </div>
        <p className={`text-xs truncate mt-0.5 ${hasUnread ? 'font-medium text-text-primary' : 'text-text-tertiary'}`}>
          {conversation.lastMessage}
        </p>
      </div>
    </button>
  );
}
