// TemplateCounters — live length counters mirroring the server's channel limits
import { cn } from '../../lib/utils.ts';

interface TemplateCounterProps {
  label: string;
  used: number;
  /** The cap reported by the API; nothing is shown when the response omits it. */
  limit: number | undefined;
  unit?: string;
}

/** One `used / limit` readout that turns red once the server would reject it. */
export function TemplateCounter({ label, used, limit, unit }: TemplateCounterProps) {
  if (limit === undefined) return null;
  const isOverLimit = used > limit;

  return (
    <span
      data-testid={`counter-${label}`}
      data-over-limit={isOverLimit ? 'true' : 'false'}
      className={cn('text-xs tabular-nums', isOverLimit ? 'text-error-text' : 'text-cool-gray')}
    >
      {used} / {limit}
      {unit ? ` ${unit}` : ''}
    </span>
  );
}
