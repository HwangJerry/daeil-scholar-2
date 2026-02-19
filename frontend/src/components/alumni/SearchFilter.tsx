// SearchFilter — Alumni search filter bar with dropdowns for class/dept/jobCat and text inputs
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { cn } from '../../lib/utils';
import { api } from '../../api/client';
import { Button } from '../ui/Button';
import type { AlumniFilters } from '../../types/api';

export interface AlumniSearchParams {
  fn: string;
  dept: string;
  name: string;
  company: string;
  position: string;
  jobCat: string;
}

interface SearchFilterProps {
  onSearch: (params: AlumniSearchParams) => void;
}

export function SearchFilter({ onSearch }: SearchFilterProps) {
  const [params, setParams] = useState<AlumniSearchParams>({
    fn: '',
    dept: '',
    name: '',
    company: '',
    position: '',
    jobCat: '',
  });

  const { data: filters } = useQuery({
    queryKey: ['alumni', 'filters'],
    queryFn: () => api.get<AlumniFilters>('/api/alumni/filters'),
    staleTime: 60 * 60_000,
  });

  const handleChange = (field: keyof AlumniSearchParams, value: string) => {
    setParams((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSearch(params);
  };

  const inputClass =
    'h-9 rounded-lg border border-border px-3 text-sm outline-none focus:ring-2 focus:ring-primary/15 focus:border-primary/30 transition-shadow duration-150';
  const selectClass = cn(inputClass, 'bg-surface');

  return (
    <form onSubmit={handleSubmit} className="rounded-xl bg-surface p-4 shadow-card border border-border">
      <div className="flex flex-wrap gap-2">
        <select
          value={params.fn}
          onChange={(e) => handleChange('fn', e.target.value)}
          className={selectClass}
        >
          <option value="">기수 전체</option>
          {filters?.fnList.map((fn) => (
            <option key={fn} value={fn}>
              {fn}기
            </option>
          ))}
        </select>
        <select
          value={params.dept}
          onChange={(e) => handleChange('dept', e.target.value)}
          className={selectClass}
        >
          <option value="">학과 전체</option>
          {filters?.deptList.map((dept) => (
            <option key={dept} value={dept}>
              {dept}
            </option>
          ))}
        </select>
        <select
          value={params.jobCat}
          onChange={(e) => handleChange('jobCat', e.target.value)}
          className={selectClass}
        >
          <option value="">업종 전체</option>
          {filters?.jobCategories?.map((cat) => (
            <option key={cat.seq} value={String(cat.seq)}>
              {cat.name}
            </option>
          ))}
        </select>
        <input
          type="text"
          placeholder="이름"
          value={params.name}
          onChange={(e) => handleChange('name', e.target.value)}
          className={inputClass}
        />
        <input
          type="text"
          placeholder="회사"
          value={params.company}
          onChange={(e) => handleChange('company', e.target.value)}
          className={inputClass}
        />
        <input
          type="text"
          placeholder="직책"
          value={params.position}
          onChange={(e) => handleChange('position', e.target.value)}
          className={inputClass}
        />
        <Button type="submit" size="sm">
          검색
        </Button>
      </div>
    </form>
  );
}
