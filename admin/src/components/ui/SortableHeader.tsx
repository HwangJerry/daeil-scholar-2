// SortableHeader — clickable table header cell with sort direction indicator
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react';
import type { SortState } from '../../hooks/useTableSort.ts';

interface SortableHeaderProps {
  label: string;
  column: string;
  sort: SortState;
  onToggle: (column: string) => void;
  className?: string;
}

export function SortableHeader({ label, column, sort, onToggle, className }: SortableHeaderProps) {
  const isActive = sort.column === column;

  const Icon = (() => {
    if (!isActive || !sort.direction) return ArrowUpDown;
    return sort.direction === 'asc' ? ArrowUp : ArrowDown;
  })();

  return (
    <th className={className}>
      <button
        type="button"
        onClick={() => onToggle(column)}
        className="inline-flex items-center gap-1 font-medium hover:text-dark-slate"
      >
        {label}
        <Icon className={`h-3.5 w-3.5 ${isActive ? 'text-royal-indigo' : 'text-cool-gray/50'}`} />
      </button>
    </th>
  );
}
