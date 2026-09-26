// MemberAdvancedFilters — cohort, department and join-date range filters for the member list
import { DEPARTMENTS } from '../../constants/departments.ts';
import type { MemberAdvancedFilters as Filters } from '../../hooks/useMemberList.ts';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';

interface MemberAdvancedFiltersProps {
  filters: Filters;
  onChange: (name: keyof Filters, value: string) => void;
  onReset: () => void;
}

export function MemberAdvancedFilters({ filters, onChange, onReset }: MemberAdvancedFiltersProps) {
  const hasFilter = Object.values(filters).some(Boolean);
  const isRangeReversed = Boolean(filters.regFrom && filters.regTo && filters.regFrom > filters.regTo);

  return (
    <div className="flex flex-wrap items-end gap-2">
      <label className="space-y-1 text-xs text-cool-gray">
        <span className="block">기수</span>
        <Input
          aria-label="기수"
          inputMode="numeric"
          placeholder="예: 30"
          value={filters.fn}
          onChange={(e) => onChange('fn', e.target.value)}
          className="h-9 w-20"
        />
      </label>
      <label className="space-y-1 text-xs text-cool-gray">
        <span className="block">학과</span>
        <Select aria-label="학과" value={filters.dept} onChange={(e) => onChange('dept', e.target.value)} className="h-9 w-28 py-1">
          <option value="">전체</option>
          {DEPARTMENTS.map((dept) => <option key={dept} value={dept}>{dept}</option>)}
        </Select>
      </label>
      <label className="space-y-1 text-xs text-cool-gray">
        <span className="block">가입일 시작</span>
        <Input aria-label="가입일 시작" type="date" value={filters.regFrom} onChange={(e) => onChange('regFrom', e.target.value)} className="h-9 w-40" />
      </label>
      <label className="space-y-1 text-xs text-cool-gray">
        <span className="block">가입일 끝</span>
        <Input aria-label="가입일 끝" type="date" value={filters.regTo} onChange={(e) => onChange('regTo', e.target.value)} className="h-9 w-40" />
      </label>
      {hasFilter && (
        <button type="button" onClick={onReset} className="h-9 px-2 text-xs text-cool-gray hover:text-dark-slate">
          필터 초기화
        </button>
      )}
      {isRangeReversed && <p role="alert" className="w-full text-xs text-error-text">가입일 시작이 끝보다 늦어 기간 조건을 적용하지 않았습니다.</p>}
    </div>
  );
}
