// LikeButton — Toggle like with optimistic update; redirects to login if unauthenticated
import { useState, type CSSProperties } from 'react';
import { useNavigate } from 'react-router-dom';
import { cn } from '../../lib/utils';
import { useAuth } from '../../hooks/useAuth';
import { useLikeToggle } from '../../hooks/useLikeToggle';

interface LikeButtonProps {
  seq: number;
  liked: boolean;
  likeCnt: number;
}

const POP_DURATION_MS = 700;
const UNSET_DURATION_MS = 300;
const DOT_ANGLES = [0, 60, 120, 180, 240, 300];

type AnimState = 'idle' | 'pop' | 'unset';

export function LikeButton({ seq, liked, likeCnt }: LikeButtonProps) {
  const navigate = useNavigate();
  const { isLoggedIn } = useAuth();
  const { mutate, isPending } = useLikeToggle(seq);
  const [anim, setAnim] = useState<AnimState>('idle');

  const handleClick = () => {
    if (!isLoggedIn) {
      navigate('/login');
      return;
    }
    if (isPending) return;
    if (!liked) {
      setAnim('pop');
      setTimeout(() => setAnim('idle'), POP_DURATION_MS);
    } else {
      setAnim('unset');
      setTimeout(() => setAnim('idle'), UNSET_DURATION_MS);
    }
    mutate();
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isPending}
      className={cn(
        'inline-flex items-center gap-1.5 p-1 text-sm font-medium transition-colors',
        liked ? 'text-error' : 'text-text-tertiary hover:text-error',
      )}
    >
      <span className="relative inline-flex items-center justify-center">
        <svg
          width={16}
          height={16}
          viewBox="0 0 24 24"
          fill={liked ? 'var(--color-error)' : 'none'}
          stroke={liked ? 'var(--color-error)' : 'var(--color-text-tertiary)'}
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
          className={cn(
            anim === 'pop' && 'animate-heart-pop',
            anim === 'unset' && 'animate-heart-unset',
          )}
        >
          <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
        </svg>

        {anim === 'pop' && (
          <span
            className="absolute animate-heart-ring rounded-full pointer-events-none"
            style={{
              width: 32,
              height: 32,
              border: '2px solid var(--color-error)',
              left: '50%',
              top: '50%',
            }}
          />
        )}

        {anim === 'pop' && DOT_ANGLES.map((deg) => (
          <span
            key={deg}
            className="absolute animate-heart-dot rounded-full pointer-events-none"
            style={{
              width: 4,
              height: 4,
              background: 'var(--color-error)',
              left: '50%',
              top: '50%',
              '--deg': `${deg}deg`,
            } as CSSProperties}
          />
        ))}
      </span>

      <span>{likeCnt}</span>
    </button>
  );
}
