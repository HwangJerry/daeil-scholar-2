// MessagePage — Chat-per-user messaging interface with split desktop / full-screen mobile layout
import { useParams, useNavigate } from 'react-router-dom';
import { AuthGuard } from '../components/auth/AuthGuard';
import { ConversationList } from '../components/message/ConversationList';
import { ChatThread } from '../components/message/ChatThread';
import { useConversations } from '../hooks/useConversations';
import { useResponsive } from '../hooks/useResponsive';

function MessagePageContent() {
  const { otherSeq: otherSeqParam } = useParams<{ otherSeq?: string }>();
  const navigate = useNavigate();
  const { isMobile } = useResponsive();
  const { data: conversationsData } = useConversations();

  const otherSeq = otherSeqParam ? parseInt(otherSeqParam, 10) : null;
  const otherName = conversationsData?.items.find((c) => c.otherSeq === otherSeq)?.otherName ?? '...';

  const handleSelect = (seq: number) => {
    navigate(`/messages/${seq}`);
  };

  const handleBack = () => {
    navigate('/messages');
  };

  if (isMobile) {
    if (otherSeq) {
      // Fixed overlay above BottomNav (z-50) so the chat fills the full viewport
      return (
        <div className="fixed inset-0 z-[60] flex flex-col bg-surface">
          <ChatThread otherSeq={otherSeq} otherName={otherName} onBack={handleBack} />
        </div>
      );
    }
    return (
      <div className="pb-20">
        <div className="px-4 py-4 border-b border-border-subtle flex items-center justify-between">
          <h1 className="text-xl font-bold text-text-primary font-serif">쪽지함</h1>
        </div>
        <ConversationList activeOtherSeq={otherSeq} onSelect={handleSelect} />
      </div>
    );
  }

  // Desktop: negate Layout's md:pt-6 / md:pb-12 / md:px-6 so the split panel
  // fills exactly from the sticky TopNav (h-14 = 56px) to the bottom of the viewport.
  return (
    <div className="flex -mt-6 -mb-12 -mx-6 h-[calc(100vh-56px)]">
      {/* Left panel — conversation list */}
      <div className="w-80 border-r border-border flex flex-col overflow-hidden">
        <div className="px-4 py-4 border-b border-border-subtle flex-shrink-0 flex items-center justify-between">
          <h1 className="text-xl font-bold text-text-primary font-serif">쪽지함</h1>
        </div>
        <div className="flex-1 overflow-y-auto">
          <ConversationList activeOtherSeq={otherSeq} onSelect={handleSelect} />
        </div>
      </div>

      {/* Right panel — chat thread or placeholder */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {otherSeq ? (
          <ChatThread otherSeq={otherSeq} otherName={otherName} />
        ) : (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-text-tertiary">대화를 선택하세요</p>
          </div>
        )}
      </div>
    </div>
  );
}

export function MessagePage() {
  return (
    <AuthGuard>
      <MessagePageContent />
    </AuthGuard>
  );
}
