// formatSeoulDateTime — shared Seoul-time formatting for admin audit metadata
const SEOUL_DATE_TIME_FORMATTER = new Intl.DateTimeFormat('ko-KR', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'Asia/Seoul',
});

/** Formats a server timestamp in Seoul time, echoing anything unparseable. */
export function formatSeoulDateTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : SEOUL_DATE_TIME_FORMATTER.format(date);
}

/** Names whoever made a change; a null operator is the system itself. */
export function formatOperatorLabel(operator: number | null): string {
  return operator === null ? '시스템' : `관리자 #${operator}`;
}
