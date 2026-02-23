// date.ts — Date formatting helpers

/**
 * Formats a date string as a relative or absolute Korean date.
 * - Under 1 minute: "방금 전"
 * - Under 1 hour: "N분 전"
 * - Under 1 day: "N시간 전"
 * - Otherwise: "YYYY.MM.DD"
 */
export function formatRelativeDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = Date.now();
  const diffMs = now - date.getTime();

  if (diffMs < 0 || isNaN(diffMs)) {
    return dateStr;
  }

  const MINUTE = 60_000;
  const HOUR = 3_600_000;
  const DAY = 86_400_000;

  if (diffMs < MINUTE) return '방금 전';
  if (diffMs < HOUR) return `${Math.floor(diffMs / MINUTE)}분 전`;
  if (diffMs < DAY) return `${Math.floor(diffMs / HOUR)}시간 전`;

  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}.${m}.${d}`;
}

/**
 * Formats a date string as an absolute Korean date: "YYYY. MM. DD"
 */
export function formatAbsoluteDate(dateStr: string): string {
  const date = new Date(dateStr);
  if (isNaN(date.getTime())) return dateStr;

  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}. ${m}. ${d}`;
}
