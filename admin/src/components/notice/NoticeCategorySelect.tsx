// NoticeCategorySelect — required category select for a notice; last option opens 카테고리 관리 in a new tab
import { Select } from '../ui/Select.tsx';
import { useFeedCategories } from '../../hooks/useFeedCategories.ts';
import { noticeCategoryOptions } from './noticeCategoryOptions.ts';

const MANAGE_VALUE = 'manage';
const MANAGE_PATH = '/feed-categories';

export interface NoticeCategorySelectProps {
  /** Selected category; null while categories are still loading. */
  value: number | null;
  /** The category the post was saved with, kept selectable even when hidden. */
  savedSeq: number | null;
  onChange: (seq: number) => void;
  disabled?: boolean;
  className?: string;
}

export function NoticeCategorySelect({ value, savedSeq, onChange, disabled, className }: NoticeCategorySelectProps) {
  const { data: categories = [] } = useFeedCategories();
  const options = noticeCategoryOptions(categories, savedSeq);

  return (
    <Select
      aria-label="카테고리"
      required
      value={value ?? ''}
      onChange={(e) => {
        if (e.target.value === MANAGE_VALUE) {
          window.open(MANAGE_PATH, '_blank', 'noopener');
          return; // Controlled value snaps back to the current category.
        }
        onChange(Number(e.target.value));
      }}
      disabled={disabled || options.length === 0}
      className={className}
    >
      {value === null && <option value="">카테고리</option>}
      {options.map((o) => (
        <option key={o.seq} value={o.seq}>{o.label}</option>
      ))}
      <option value={MANAGE_VALUE}>카테고리 관리</option>
    </Select>
  );
}
