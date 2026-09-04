// DisclosureStatePanel — Cohesive empty and error state for the disclosure archive
import type { LucideIcon } from 'lucide-react';
import { Button } from '../ui/Button';

interface DisclosureStatePanelProps {
  description: string;
  icon: LucideIcon;
  title: string;
  onRetry?: () => void;
}

export function DisclosureStatePanel({
  description,
  icon: Icon,
  title,
  onRetry,
}: DisclosureStatePanelProps) {
  return (
    <div className="flex min-h-64 flex-col items-center justify-center border-y border-border px-6 py-12 text-center">
      <span className="mb-4 inline-flex size-12 items-center justify-center rounded-full bg-primary-light text-primary">
        <Icon aria-hidden="true" className="size-5" />
      </span>
      <h2 className="font-serif text-xl font-semibold text-text-primary">{title}</h2>
      <p className="mt-2 max-w-sm text-body-sm leading-relaxed text-text-tertiary">
        {description}
      </p>
      {onRetry && (
        <Button type="button" variant="outline" onClick={onRetry} className="mt-6 min-h-11">
          다시 시도
        </Button>
      )}
    </div>
  );
}
