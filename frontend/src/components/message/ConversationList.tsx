// ConversationList — Scrollable list of conversation summaries with loading and empty states
import { useConversations } from '../../hooks/useConversations';
import { ConversationItem } from './ConversationItem';
import { Bone } from '../ui/Skeleton';

interface ConversationListProps {
  activeOtherSeq: number | null;
  onSelect: (seq: number) => void;
}

function ConversationListSkeleton() {
  return (
    <div className="divide-y divide-border-subtle">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-4 py-3">
          <Bone className="w-12 h-12 rounded-full flex-shrink-0" />
          <div className="flex-1 space-y-2">
            <Bone className="h-3.5 w-32" />
            <Bone className="h-3 w-48" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function ConversationList({ activeOtherSeq, onSelect }: ConversationListProps) {
  const { data, isLoading, isError } = useConversations();

  if (isLoading) return <ConversationListSkeleton />;

  if (isError) {
    return (
      <div className="flex items-center justify-center py-16">
        <p className="text-sm text-text-tertiary">대화 목록을 불러올 수 없습니다</p>
      </div>
    );
  }

  const items = data?.items ?? [];

  if (items.length === 0) {
    return (
      <div className="flex items-center justify-center py-16">
        <p className="text-sm text-text-tertiary">대화 내역이 없습니다</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-border-subtle">
      {items.map((conversation) => (
        <ConversationItem
          key={conversation.otherSeq}
          conversation={conversation}
          isActive={activeOtherSeq === conversation.otherSeq}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}
