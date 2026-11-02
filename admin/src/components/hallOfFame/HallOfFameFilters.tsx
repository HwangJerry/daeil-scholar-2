// HallOfFameFilters — tier, kind, and name/cohort filter controls for the entry list
import { Search } from 'lucide-react';
import {
  HALL_KINDS,
  HALL_TIERS,
  type HallKind,
  type HallTier,
} from '../../types/hallOfFame.ts';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import { formatTier, HALL_KIND_LABELS } from './hallOfFameLabels.ts';

interface HallOfFameFiltersProps {
  tier: HallTier | '';
  kind: HallKind | '';
  search: string;
  onTierChange: (tier: HallTier | '') => void;
  onKindChange: (kind: HallKind | '') => void;
  onSearchChange: (search: string) => void;
}

export function HallOfFameFilters({
  tier,
  kind,
  search,
  onTierChange,
  onKindChange,
  onSearchChange,
}: HallOfFameFiltersProps) {
  return (
    <div className="flex flex-col gap-2 md:flex-row">
      <Select
        aria-label="등급 필터"
        className="md:w-40"
        value={tier}
        onChange={(event) => {
          onTierChange(event.target.value ? (Number(event.target.value) as HallTier) : '');
        }}
      >
        <option value="">전체 등급</option>
        {HALL_TIERS.map((value) => (
          <option key={value} value={value}>{formatTier(value)}</option>
        ))}
      </Select>
      <Select
        aria-label="종류 필터"
        className="md:w-40"
        value={kind}
        onChange={(event) => onKindChange(event.target.value as HallKind | '')}
      >
        <option value="">전체 종류</option>
        {HALL_KINDS.map((value) => (
          <option key={value} value={value}>{HALL_KIND_LABELS[value]}</option>
        ))}
      </Select>
      <div className="relative md:flex-1">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-cool-gray" />
        <Input
          aria-label="이름 또는 기수 검색"
          placeholder="이름 또는 기수 검색"
          className="pl-9"
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </div>
    </div>
  );
}
