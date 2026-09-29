// NoticeCategoryFilter — "카테고리: 전체" filter over every feed category, hidden ones included
import { Select } from '../ui/Select.tsx';
import { useFeedCategories } from '../../hooks/useFeedCategories.ts';

export interface NoticeCategoryFilterProps {
  /** Selected category seq; null means 전체. */
  value: number | null;
  onChange: (seq: number | null) => void;
}

export function NoticeCategoryFilter({ value, onChange }: NoticeCategoryFilterProps) {
  const { data: categories = [] } = useFeedCategories();
  return (
    <Select
      aria-label="카테고리 필터"
      value={value ?? ''}
      onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))}
      className="w-44 shrink-0"
    >
      <option value="">카테고리: 전체</option>
      {categories.map((c) => (
        <option key={c.seq} value={c.seq}>
          {c.name}{c.openYn === 'N' ? ' (숨김)' : ''}
        </option>
      ))}
    </Select>
  );
}
