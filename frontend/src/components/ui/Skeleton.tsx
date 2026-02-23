// Skeleton — Bone primitive for shimmer skeleton loading states
import { cn } from '../../lib/utils';

interface BoneProps {
  className?: string;
}

export function Bone({ className }: BoneProps) {
  return <div className={cn('skeleton-shimmer rounded', className)} />;
}
