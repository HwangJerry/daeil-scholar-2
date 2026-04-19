// StatusFilterDropdown — status filter button, dropdown menu, and active filter chip
import { useState, useRef, useEffect } from 'react';
import { ListFilter, ChevronDown, Check, X } from 'lucide-react';

export const STATUS_OPTIONS = [
  { value: '', label: '전체' },
  { value: 'AAA', label: '탈퇴' },
  { value: 'ABA', label: '휴면' },
  { value: 'ACA', label: '정지' },
  { value: 'BAA', label: '승인거절' },
  { value: 'BBB', label: '승인대기' },
  { value: 'CCC', label: '승인회원' },
  { value: 'ZZZ', label: '운영자' },
];

const STATUS_LABELS: Record<string, string> = Object.fromEntries(
  STATUS_OPTIONS.filter((o) => o.value).map((o) => [o.value, o.label])
);

interface StatusFilterDropdownProps {
  value: string;
  onChange: (value: string) => void;
}

export function StatusFilterDropdown({ value, onChange }: StatusFilterDropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <>
      <div ref={containerRef} className="relative">
        <button
          onClick={() => setIsOpen((o) => !o)}
          className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-sm
            ${value
              ? 'border-royal-indigo bg-indigo-50 text-royal-indigo'
              : 'border-border-light bg-white text-cool-gray hover:border-cool-gray'
            }`}
        >
          <ListFilter size={14} />
          <span>상태</span>
          <ChevronDown size={13} className={`transition-transform ${isOpen ? 'rotate-180' : ''}`} />
        </button>
        {isOpen && (
          <div className="absolute top-full left-0 z-10 mt-1 w-36 rounded-lg border border-border-light bg-white shadow-md py-1">
            {STATUS_OPTIONS.map(({ value: optValue, label }) => (
              <button
                key={optValue}
                onClick={() => { onChange(optValue); setIsOpen(false); }}
                className={`w-full flex items-center justify-between px-3 py-1.5 text-xs hover:bg-background
                  ${value === optValue ? 'text-royal-indigo font-semibold' : 'text-dark-slate'}`}
              >
                {label}
                {value === optValue && <Check size={11} />}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* 활성 필터 칩 */}
      {value && (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 bg-indigo-50 text-royal-indigo rounded-full text-xs font-medium">
          {STATUS_LABELS[value]}
          <button onClick={() => onChange('')} aria-label="필터 해제">
            <X size={11} />
          </button>
        </span>
      )}
    </>
  );
}
