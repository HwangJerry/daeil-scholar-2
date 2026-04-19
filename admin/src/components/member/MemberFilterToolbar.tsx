// MemberFilterToolbar — composes status filter, result count, and search input into a single toolbar row
import { StatusFilterDropdown } from './StatusFilterDropdown.tsx';
import { MemberSearchInput } from './MemberSearchInput.tsx';

interface MemberFilterToolbarProps {
  statusFilter: string;
  search: string;
  total: number | null;
  onStatusChange: (value: string) => void;
  onSearchChange: (value: string) => void;
}

export function MemberFilterToolbar({
  statusFilter,
  search,
  total,
  onStatusChange,
  onSearchChange,
}: MemberFilterToolbarProps) {
  return (
    <div className="flex items-center gap-2">
      <StatusFilterDropdown value={statusFilter} onChange={onStatusChange} />
      <span className="text-xs text-cool-gray">{total !== null ? `${total}명` : '—'}</span>
      <MemberSearchInput value={search} onChange={onSearchChange} />
    </div>
  );
}
