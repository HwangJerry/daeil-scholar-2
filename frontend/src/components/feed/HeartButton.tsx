// HeartButton — Animated like button with heartPop/heartUnset/heartRing/heartDot effects
import { useState } from 'react';
import { cn } from '../../lib/utils';

interface HeartButtonProps {
  liked: boolean;
  onToggle: () => void;
  count: number;
  dark?: boolean;
}

const POP_DURATION_MS = 700;
const UNSET_DURATION_MS = 300;
const DOT_ANGLES = [0, 60, 120, 180, 240, 300];

type AnimState = 'idle' | 'pop' | 'unset';

export function HeartButton({ liked, onToggle, count, dark = false }: HeartButtonProps) {
  const [anim, setAnim] = useState<AnimState>('idle');

  const handleClick = () => {
    onToggle();
    if (!liked) {
      setAnim('pop');
      setTimeout(() => setAnim('idle'), POP_DURATION_MS);
    } else {
      setAnim('unset');
      setTimeout(() => setAnim('idle'), UNSET_DURATION_MS);
    }
  };

  const activeColor = dark ? 'var(--color-like-on-dark)' : 'var(--color-error)';
  const idleColor = dark ? 'color-mix(in srgb, var(--color-white) 50%, transparent)' : 'var(--color-text-placeholder)';
  const heartColor = liked ? activeColor : idleColor;

  return (
    <button
      type="button"
      onClick={handleClick}
      className="inline-flex items-center gap-1"
      aria-label={liked ? '좋아요 취소' : '좋아요'}
      aria-pressed={liked}
    >
      <span className="relative inline-flex items-center justify-center">
        <svg
          width={13}
          height={13}
          viewBox="0 0 24 24"
          fill={liked ? heartColor : 'none'}
          stroke={heartColor}
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
            className="absolute w-7 h-7 left-1/2 top-1/2 border-2 animate-heart-ring rounded-full pointer-events-none"
            style={{
              borderColor: activeColor,
            }}
          />
        )}

        {anim === 'pop' && DOT_ANGLES.map((deg) => (
          <span
            key={deg}
            className="absolute w-1 h-1 left-1/2 top-1/2 animate-heart-dot rounded-full pointer-events-none"
            style={{
              background: activeColor,
              '--deg': `${deg}deg`,
            } as React.CSSProperties}
          />
        ))}
      </span>

      <span className="text-xs" style={{ color: heartColor }}>
        {count}
      </span>
    </button>
  );
}
