// ChatThread — Full conversation view with message bubbles, pagination, and auto-scroll
import { Fragment, useEffect, useRef, useState } from 'react';
import { ArrowLeft, ChevronUp } from 'lucide-react';
import { useConversationMessages } from '../../hooks/useConversationMessages';
import { useMarkConversationRead } from '../../hooks/useMarkConversationRead';
import { useAuth } from '../../hooks/useAuth';
import { MessageBubble } from './MessageBubble';
import { MessageInput } from './MessageInput';
import { Bone } from '../ui/Skeleton';
import { ChatDateDivider } from './ChatDateDivider';
import { formatMessageDate, formatDividerDate } from '../../utils/messageFormat';

interface ChatThreadProps {
  otherSeq: number;
  otherName: string;
  onBack?: () => void;
}

function ChatThreadSkeleton() {
  return (
    <div className="flex flex-col gap-3 p-4">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className={`flex ${i % 2 === 0 ? 'justify-start' : 'justify-end'}`}>
          <Bone className={`h-9 rounded-2xl ${i % 3 === 0 ? 'w-48' : 'w-32'}`} />
        </div>
      ))}
    </div>
  );
}

export function ChatThread({ otherSeq, otherName, onBack }: ChatThreadProps) {
  const [page, setPage] = useState(1);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const { user } = useAuth();
  const { data, isLoading, isError } = useConversationMessages(otherSeq, page);
  const { mutate: markRead } = useMarkConversationRead();

  useEffect(() => {
    if (otherSeq > 0) {
      setPage(1);
      markRead(otherSeq);
    }
  }, [otherSeq, markRead]);

  useEffect(() => {
    if (page === 1) {
      scrollContainerRef.current?.scrollTo({ top: scrollContainerRef.current.scrollHeight, behavior: 'instant' });
    }
  }, [data, page]);

  const handleMessageSent = () => {
    markRead(otherSeq);
    scrollContainerRef.current?.scrollTo({ top: scrollContainerRef.current.scrollHeight, behavior: 'smooth' });
  };

  const totalPages = data?.totalPages ?? 0;
  const items = data?.items ?? [];

  return (
    <div className="flex flex-col h-full">
      {/* Mobile header — hidden on desktop */}
      {onBack && (
        <div className="relative flex items-center px-4 py-3 border-b border-border-subtle flex-shrink-0 md:hidden">
          <button
            type="button"
            onClick={onBack}
            className="absolute left-4 rounded-lg p-1.5 text-text-tertiary hover:bg-background transition-colors duration-150"
            aria-label="뒤로가기"
          >
            <ArrowLeft size={20} />
          </button>
          <span className="mx-auto text-base font-semibold text-text-primary font-serif">{otherName}</span>
        </div>
      )}

      {/* Desktop header — always shown, hidden on mobile when onBack is present */}
      <div className={`flex items-center gap-2 px-4 py-3 border-b border-border-subtle flex-shrink-0 ${onBack ? 'hidden md:flex' : 'flex'}`}>
        <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center flex-shrink-0">
          <span className="text-xs font-bold text-white">{otherName.charAt(0) || '?'}</span>
        </div>
        <span className="text-base font-semibold text-text-primary font-serif">{otherName}</span>
      </div>

      {/* Message list */}
      <div ref={scrollContainerRef} className="flex-1 overflow-y-auto px-4 py-3">
        {isLoading && <ChatThreadSkeleton />}

        {isError && (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-text-tertiary">메시지를 불러올 수 없습니다</p>
          </div>
        )}

        {!isLoading && !isError && (
          <>
            {totalPages > 1 && page < totalPages && (
              <div className="flex justify-center mb-4">
                <button
                  type="button"
                  onClick={() => setPage((p) => p + 1)}
                  className="flex items-center gap-1 text-xs text-text-tertiary hover:text-text-secondary transition-colors duration-150 px-3 py-1.5 rounded-full border border-border"
                >
                  <ChevronUp size={14} />
                  더 이전 메시지 보기
                </button>
              </div>
            )}

            {items.length === 0 ? (
              <div className="flex items-center justify-center h-full">
                <p className="text-sm text-text-tertiary">대화를 시작해보세요</p>
              </div>
            ) : (
              items.map((message, index) => {
                const prevMessage = items[index - 1];
                const showDivider =
                  !prevMessage ||
                  formatMessageDate(prevMessage.regDate) !== formatMessageDate(message.regDate);

                return (
                  <Fragment key={message.amSeq}>
                    {showDivider && <ChatDateDivider label={formatDividerDate(message.regDate)} />}
                    <MessageBubble message={message} currentUserSeq={user?.usrSeq ?? 0} />
                  </Fragment>
                );
              })
            )}

          </>
        )}
      </div>

      {/* Input */}
      <MessageInput key={otherSeq} recipientSeq={otherSeq} onMessageSent={handleMessageSent} />
    </div>
  );
}
