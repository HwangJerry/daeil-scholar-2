// AdCommentSection — Auth-aware section shell composing ad comment input and list
import { useAuth } from '../../hooks/useAuth';
import { AdCommentInput } from './AdCommentInput';
import { AdCommentList } from './AdCommentList';

interface AdCommentSectionProps {
  maSeq: number;
}

export function AdCommentSection({ maSeq }: AdCommentSectionProps) {
  const { isLoggedIn } = useAuth();

  return (
    <section className="border-t border-border-subtle px-5 pt-4 pb-3 animate-fade-in-up">
      {isLoggedIn ? (
        <AdCommentInput maSeq={maSeq} />
      ) : (
        <p className="mb-4 rounded-lg bg-background-secondary px-3 py-2.5 text-xs text-text-tertiary">
          댓글을 작성하려면 로그인이 필요합니다.
        </p>
      )}
      <AdCommentList maSeq={maSeq} />
    </section>
  );
}
