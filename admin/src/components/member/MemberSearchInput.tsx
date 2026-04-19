// MemberSearchInput — name search input with inline clear button
import { Search, X } from 'lucide-react';
import { Input } from '../ui/Input.tsx';

interface MemberSearchInputProps {
  value: string;
  onChange: (value: string) => void;
}

export function MemberSearchInput({ value, onChange }: MemberSearchInputProps) {
  return (
    <div className="ml-auto relative">
      <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-cool-gray pointer-events-none" />
      <Input
        aria-label="이름·연락처 검색"
        placeholder="이름·연락처 검색..."
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="pl-8 pr-7 max-w-xs"
      />
      {value && (
        <button
          onClick={() => onChange('')}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-cool-gray hover:text-dark-slate"
          aria-label="검색 초기화"
        >
          <X size={13} />
        </button>
      )}
    </div>
  );
}
