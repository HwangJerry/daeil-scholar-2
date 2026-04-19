// MessageInput — Chat compose area with pill-style input (IG DM), Enter-to-send, and send button
import { Send } from 'lucide-react';
import { useMessageComposition } from '../../hooks/useMessageComposition';

const CHAR_COUNTER_THRESHOLD = 800;

interface MessageInputProps {
  recipientSeq: number;
  onMessageSent?: () => void;
}

export function MessageInput({ recipientSeq, onMessageSent }: MessageInputProps) {
  const {
    content,
    setContent,
    submit,
    isPending,
    isContentEmpty,
    errorMessage,
    maxLength,
  } = useMessageComposition({
    recipientSeq,
    onSuccess: onMessageSent ?? (() => {}),
  });

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      if (!isContentEmpty && !isPending) {
        submit();
      }
    }
  };

  return (
    <div className="border-t border-border-subtle bg-surface px-3 py-2">
      {errorMessage && (
        <p className="text-xs text-error mb-1.5 px-1">{errorMessage}</p>
      )}
      <div className="flex items-center gap-2 rounded-full border border-border bg-background px-4 py-2">
        {/* Textarea */}
        <div className="flex-1 min-w-0 relative">
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            maxLength={maxLength}
            rows={1}
            placeholder="메시지를 입력하세요"
            disabled={isPending}
            className="w-full resize-none bg-transparent text-sm text-text-primary outline-none placeholder:text-text-placeholder max-h-32 overflow-y-auto"
            style={{ minHeight: '24px' }}
          />
          {content.length > CHAR_COUNTER_THRESHOLD && (
            <span className="absolute bottom-0 right-0 text-[10px] text-text-placeholder">
              {content.length}/{maxLength}
            </span>
          )}
        </div>

        {/* Send button — visible only when content is not empty */}
        <button
          type="button"
          onClick={submit}
          disabled={isContentEmpty || isPending}
          className={`flex-shrink-0 w-8 h-8 rounded-full bg-primary flex items-center justify-center transition-all duration-150 ${isContentEmpty ? 'invisible' : 'visible'}`}
          aria-label="보내기"
        >
          {isPending ? (
            <div className="w-3.5 h-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
          ) : (
            <Send size={14} className="text-white" />
          )}
        </button>
      </div>
    </div>
  );
}
