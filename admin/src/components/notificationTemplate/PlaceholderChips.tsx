// PlaceholderChips — inserts an allowed `{token}` into the field the administrator was editing
import type { NotificationTemplatePlaceholder } from '../../types/notificationTemplate.ts';

interface PlaceholderChipsProps {
  placeholders: NotificationTemplatePlaceholder[];
  disabled?: boolean;
  onInsert: (name: string) => void;
}

export function PlaceholderChips({ placeholders, disabled, onInsert }: PlaceholderChipsProps) {
  if (placeholders.length === 0) {
    return <p className="mt-2 text-xs text-cool-gray">이 문구에는 치환 항목이 없습니다.</p>;
  }

  return (
    <div className="mt-2">
      <div className="flex flex-wrap gap-1.5">
        {placeholders.map((placeholder) => (
          <button
            key={placeholder.name}
            type="button"
            disabled={disabled}
            onClick={() => onInsert(placeholder.name)}
            className="rounded-full border border-border bg-background px-2.5 py-1 text-xs font-medium text-royal-indigo transition-colors hover:bg-soft-sky disabled:cursor-not-allowed disabled:opacity-50"
          >
            {`{${placeholder.name}}`}
            {placeholder.required && <span className="ml-1 text-error-text">필수</span>}
          </button>
        ))}
      </div>
      <ul className="mt-1.5 space-y-0.5 text-xs text-cool-gray">
        {placeholders.map((placeholder) => (
          <li key={placeholder.name}>
            {`{${placeholder.name}}`} — {placeholder.description} (예: {placeholder.sample})
          </li>
        ))}
      </ul>
    </div>
  );
}
