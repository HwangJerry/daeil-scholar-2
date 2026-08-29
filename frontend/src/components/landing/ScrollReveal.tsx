// ScrollReveal — Viewport-triggered motion boundary for landing-page sections
import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';
import { useScrollReveal } from './useScrollReveal';

interface ScrollRevealProps {
  children: ReactNode;
  className?: string;
}

export function ScrollReveal({ children, className }: ScrollRevealProps) {
  const { elementRef, isRevealed } = useScrollReveal();

  return (
    <div
      ref={elementRef}
      data-scroll-reveal=""
      data-revealed={isRevealed ? 'true' : 'false'}
      className={cn('landing-scroll-reveal', className)}
    >
      {children}
    </div>
  );
}
