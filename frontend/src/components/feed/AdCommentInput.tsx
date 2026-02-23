// AdCommentInput — Comment submission form for ad cards with inline error feedback
import { useState } from 'react';
import { Send } from 'lucide-react';
import { useAddAdComment } from '../../hooks/useAdComments';
import { useAuthRedirect } from '../../hooks/useAuthRedirect';

interface AdCommentInputProps {
  maSeq: number;
}

export function AdCommentInput({ maSeq }: AdCommentInputProps) {
  const [text, setText] = useState('');
  const onAuthError = useAuthRedirect();
  const { mutate, isPending, isError, error, reset } = useAddAdComment(maSeq, { onAuthError });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setText(e.target.value);
    if (isError) reset();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = text.trim();
    if (!trimmed || isPending) return;
    mutate({ contents: trimmed }, { onSuccess: () => setText('') });
  };

  return (
    <div className="mb-4">
      <form onSubmit={handleSubmit} className="flex items-center gap-2">
        <input
          type="text"
          value={text}
          onChange={handleChange}
          placeholder="댓글을 입력하세요"
          maxLength={500}
          className="flex-1 rounded-lg border border-border-subtle bg-background-secondary px-3 py-2 text-sm text-text-primary placeholder:text-text-tertiary focus:border-primary focus:outline-none"
        />
        <button
          type="submit"
          disabled={isPending || !text.trim()}
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary text-white transition-colors hover:bg-primary/90 disabled:opacity-40"
        >
          <Send className="h-4 w-4" />
        </button>
      </form>
      {isError && (
        <p className="mt-1.5 text-xs text-error-text">
          {error?.message || '댓글 등록에 실패했습니다. 다시 시도해주세요.'}
        </p>
      )}
    </div>
  );
}
