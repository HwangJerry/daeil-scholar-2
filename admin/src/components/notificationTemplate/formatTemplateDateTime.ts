// formatTemplateDateTime — Seoul-time formatting for notification template metadata
import { formatSeoulDateTime } from '../../lib/formatSeoulDateTime.ts';

export { formatOperatorLabel as formatTemplateOperator } from '../../lib/formatSeoulDateTime.ts';

/** A template that has never been overridden has no timestamp to show. */
export function formatTemplateDateTime(value: string | null): string {
  return value === null ? '수정된 적 없음' : formatSeoulDateTime(value);
}
