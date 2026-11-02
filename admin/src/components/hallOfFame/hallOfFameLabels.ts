// hallOfFameLabels — display labels for Hall of Fame tiers, kinds, and import diff fields
import type { HallKind, HallSource, HallTier } from '../../types/hallOfFame.ts';

export const EMPTY_VALUE_LABEL = '—';
export const UNLISTED_TIER_LABEL = '미등재';
export const ROLE_LABEL_FIELD_LABEL = '직함(공개 라벨)';

export const HALL_KIND_LABELS: Record<HallKind, string> = {
  alumni: '동문',
  companion: '동행',
  teacher: '교사',
  guest: '지인·비동문',
  org: '법인',
};

export const HALL_SOURCE_LABELS: Record<HallSource, string> = {
  excel: '엑셀',
  manual: '수동',
};

export function formatTier(tier: HallTier | null) {
  return tier === null ? UNLISTED_TIER_LABEL : `${tier.toLocaleString()}만원`;
}

/** Only these diff fields render, so a stray phone field in a response can never reach the screen. */
const IMPORT_FIELD_LABELS: Record<string, string> = {
  displayName: '이름',
  kind: '종류',
  cohort: '기수',
  department: '과',
  tierManWon: '등급',
  contactName: '담당자',
  roleLabel: ROLE_LABEL_FIELD_LABEL,
};

export function getImportFieldLabel(field: string): string | null {
  return IMPORT_FIELD_LABELS[field] ?? null;
}

export function formatImportValue(field: string, value: string | number | null) {
  if (value === null || value === '') return EMPTY_VALUE_LABEL;
  if (field === 'tierManWon') return formatTier(value as HallTier);
  if (field === 'kind') return HALL_KIND_LABELS[value as HallKind] ?? String(value);
  return String(value);
}
