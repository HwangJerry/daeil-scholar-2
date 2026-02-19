// LikeButton — Toggle like with optimistic update; redirects to login if unauthenticated
import { Heart } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { cn } from '../../lib/utils';
import { useAuth } from '../../hooks/useAuth';
import { useLikeToggle } from '../../hooks/useLikeToggle';

interface LikeButtonProps {
  seq: number;
  liked: boolean;
  likeCnt: number;
}

export function LikeButton({ seq, liked, likeCnt }: LikeButtonProps) {
  const navigate = useNavigate();
  const { isLoggedIn } = useAuth();
  const { mutate, isPending } = useLikeToggle(seq);

  const handleClick = () => {
    if (!isLoggedIn) {
      navigate('/login');
      return;
    }
    if (isPending) return;
    mutate();
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isPending}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium transition-colors',
        liked
          ? 'bg-error-subtle text-error'
          : 'bg-background-secondary text-text-tertiary hover:text-error-text',
      )}
    >
      <Heart className={cn('h-4 w-4', liked && 'fill-current')} />
      <span>{likeCnt}</span>
    </button>
  );
}
