// formatPolicyDateTime — shared Seoul-time formatting for app update policy screens
const POLICY_DATE_TIME_FORMATTER = new Intl.DateTimeFormat('ko-KR', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'Asia/Seoul',
});

export function formatPolicyDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : POLICY_DATE_TIME_FORMATTER.format(date);
}

export function formatOperator(operator: number | null) {
  return operator === null ? '시스템' : `관리자 #${operator}`;
}
