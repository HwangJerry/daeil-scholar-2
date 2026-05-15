// SearchFilter — Alumni search bar with name input, dropdown filters, and active filter pills
import { useCallback, useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Search, RotateCcw, X } from 'lucide-react';
import { Badge } from '../ui/Badge';
import { cn } from '../../lib/utils';
import { api } from '../../api/client';
import type { AlumniFilters } from '../../types/api';

export interface AlumniSearchParams {
  fn: string;
  dept: string;
  name: string;
  jobCat: string;
}

interface SearchFilterProps {
  onSearch: (params: AlumniSearchParams) => void;
}

const DROPDOWN_FILTER_KEYS = ['fn', 'dept', 'jobCat'] as const;
type DropdownFilterKey = typeof DROPDOWN_FILTER_KEYS[number];

const BASE_SELECT_CLASS =
  'h-10 rounded-lg border border-border pl-3 pr-8 text-sm outline-none focus:ring-2 focus:ring-primary/15 focus:border-primary/30 transition-all duration-150 cursor-pointer';
const ACTIVE_SELECT_CLASS =
  'bg-primary text-white border-primary font-semibold';
const INACTIVE_SELECT_CLASS = 'bg-surface text-text-secondary';
const NAME_SEARCH_DEBOUNCE_MS = 400;

export function SearchFilter({ onSearch }: SearchFilterProps) {
  const [params, setParams] = useState<AlumniSearchParams>({
    fn: '', dept: '', name: '', jobCat: '',
  });
  const [inputValue, setInputValue] = useState('');
  const [isComposing, setIsComposing] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const { data: filters } = useQuery({
    queryKey: ['alumni', 'filters'],
    queryFn: () => api.get<AlumniFilters>('/api/alumni/filters'),
    staleTime: 60 * 60_000,
  });

  const hasActiveFilter = !!(params.fn || params.dept || params.name || params.jobCat);
  const activeDropdownFilters = DROPDOWN_FILTER_KEYS.filter((k) => params[k] !== '');

  const commitName = useCallback((value: string) => {
    const trimmedValue = value.trim();
    if (trimmedValue === params.name) return;

    const next = { ...params, name: trimmedValue };
    setParams(next);
    onSearch(next);
  }, [onSearch, params]);

  useEffect(() => {
    if (isComposing) return;

    const trimmedValue = inputValue.trim();
    if (trimmedValue === params.name) return;

    const timerId = window.setTimeout(() => {
      commitName(inputValue);
    }, NAME_SEARCH_DEBOUNCE_MS);

    return () => window.clearTimeout(timerId);
  }, [commitName, inputValue, isComposing, params.name]);

  const handleSearchSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    commitName(inputValue);
    inputRef.current?.blur();
  };

  const handleSelectChange = (field: keyof AlumniSearchParams, value: string) => {
    const next = { ...params, [field]: value };
    setParams(next);
    onSearch(next);
  };

  const clearFilter = (field: keyof AlumniSearchParams) => {
    if (field === 'name') setInputValue('');
    const next = { ...params, [field]: '' };
    setParams(next);
    onSearch(next);
  };

  const resetAll = () => {
    setInputValue('');
    const next = { fn: '', dept: '', name: '', jobCat: '' };
    setParams(next);
    onSearch(next);
  };

  const getDropdownLabel = (field: DropdownFilterKey): string => {
    if (field === 'fn') return `${params.fn}기`;
    if (field === 'dept') return params.dept;
    if (field === 'jobCat') {
      const cat = filters?.jobCategories.find((c) => String(c.seq) === params.jobCat);
      return cat?.name ?? params.jobCat;
    }
    return '';
  };

  return (
    <div className="rounded-xl bg-surface p-3.5 shadow-card border border-border flex flex-col gap-3">
      {/* Search + filters: 모바일=2행, sm+=단일 행 */}
      <div className="flex flex-col min-[600px]:flex-row gap-2">

        {/* Unit 1 — 검색창 + 초기화 버튼 */}
        <div className="flex gap-2 items-center min-[600px]:flex-1">
          <form
            onSubmit={handleSearchSubmit}
            className="flex-1 h-10 flex items-center gap-2 bg-background border border-border rounded-lg px-3"
          >
            <Search size={13} className="text-text-placeholder flex-shrink-0" />
            <input
              ref={inputRef}
              type="search"
              enterKeyHint="search"
              autoComplete="off"
              placeholder="이름 또는 태그로 검색"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onCompositionStart={() => setIsComposing(true)}
              onCompositionEnd={(e) => {
                setIsComposing(false);
                setInputValue(e.currentTarget.value);
              }}
              className="flex-1 border-none bg-transparent outline-none text-sm text-text-primary placeholder:text-text-placeholder"
            />
            {inputValue && (
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => { setInputValue(''); clearFilter('name'); }}
                className="text-text-placeholder hover:text-text-secondary transition-colors"
              >
                <X size={14} />
              </button>
            )}
          </form>
          {hasActiveFilter && (
            <button
              onClick={resetAll}
              className="flex-shrink-0 flex items-center gap-1.5 h-10 px-3 rounded-lg border border-border text-[12px] text-text-tertiary hover:border-border-hover hover:text-text-primary transition-colors duration-150 whitespace-nowrap bg-surface"
            >
              <RotateCcw size={11} />
              초기화
            </button>
          )}
        </div>

        {/* Unit 2 — 드롭다운 3개: 모바일 grid, 데스크탑 flex */}
        <div className="grid grid-cols-3 min-[600px]:flex gap-2 min-[600px]:items-center">
          <select
            value={params.fn}
            onChange={(e) => handleSelectChange('fn', e.target.value)}
            className={cn(BASE_SELECT_CLASS, 'w-full min-[600px]:w-auto min-[600px]:flex-none', params.fn ? ACTIVE_SELECT_CLASS : INACTIVE_SELECT_CLASS)}
          >
            <option value="">기수</option>
            {filters?.fnList?.map((fn) => (
              <option key={fn} value={fn}>{fn}기</option>
            ))}
          </select>
          <select
            value={params.dept}
            onChange={(e) => handleSelectChange('dept', e.target.value)}
            className={cn(BASE_SELECT_CLASS, 'w-full min-[600px]:w-auto min-[600px]:flex-none', params.dept ? ACTIVE_SELECT_CLASS : INACTIVE_SELECT_CLASS)}
          >
            <option value="">학과</option>
            {filters?.deptList?.map((dept) => (
              <option key={dept} value={dept}>{dept}</option>
            ))}
          </select>
          <select
            value={params.jobCat}
            onChange={(e) => handleSelectChange('jobCat', e.target.value)}
            className={cn(BASE_SELECT_CLASS, 'w-full min-[600px]:w-auto min-[600px]:flex-none', params.jobCat ? ACTIVE_SELECT_CLASS : INACTIVE_SELECT_CLASS)}
          >
            <option value="">업종</option>
            {filters?.jobCategories?.map((cat) => (
              <option key={cat.seq} value={String(cat.seq)}>{cat.name}</option>
            ))}
          </select>
        </div>

      </div>

      {/* Row 2: Active filter pills — shown when any dropdown filter is active */}
      {activeDropdownFilters.length > 0 && (
        <div className="flex gap-1.5 flex-wrap items-center">
          {activeDropdownFilters.map((field) => {
            const label = getDropdownLabel(field);
            if (!label) return null;
            return (
              <Badge key={field} variant="category" className="gap-1">
                {label}
                <button
                  onClick={() => clearFilter(field)}
                  className="text-cat-notice-text/70 hover:text-cat-notice-text transition-colors leading-none"
                >
                  <X size={12} />
                </button>
              </Badge>
            );
          })}
        </div>
      )}
    </div>
  );
}
