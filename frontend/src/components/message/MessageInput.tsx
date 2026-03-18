// MessageInput — Chat compose area with Enter-to-send, character counter, and send button
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
    if (e.key === 'Enter' && !e.shiftKey) {
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
      <div className="flex items-end gap-2">
        <div className="flex-1 relative">
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            maxLength={maxLength}
            rows={1}
            placeholder="메시지를 입력하세요"
            disabled={isPending}
            className="w-full resize-none rounded-2xl border border-border bg-background px-3 py-2 text-sm text-text-primary outline-none transition-shadow duration-150 placeholder:text-text-placeholder focus:ring-2 focus:ring-primary/15 focus:border-primary/30 max-h-32 overflow-y-auto"
            style={{ minHeight: '38px' }}
          />
          {content.length > CHAR_COUNTER_THRESHOLD && (
            <span className="absolute bottom-2 right-3 text-[10px] text-text-placeholder">
              {content.length}/{maxLength}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={submit}
          disabled={isContentEmpty || isPending}
          className="flex-shrink-0 w-9 h-9 rounded-full bg-primary flex items-center justify-center transition-opacity duration-150 disabled:opacity-40"
          aria-label="보내기"
        >
          {isPending ? (
            <div className="w-4 h-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
          ) : (
            <Send size={15} className="text-white" />
          )}
        </button>
      </div>
    </div>
  );
}
