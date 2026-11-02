// hallOfFame — admin API contract types for Hall of Fame entries and the Excel import
export const HALL_TIERS = [100, 300, 500, 1000] as const;
export type HallTier = (typeof HALL_TIERS)[number];

export const HALL_KINDS = ['alumni', 'companion', 'teacher', 'guest', 'org'] as const;
export type HallKind = (typeof HALL_KINDS)[number];

export type HallSource = 'excel' | 'manual';

export interface HallEntry {
  id: number;
  displayName: string;
  kind: HallKind;
  cohort: number | null;
  department: string | null;
  tierManWon: HallTier | null;
  contactName: string | null;
  roleLabel: string | null;
  consentPublic: boolean;
  visible: boolean;
  source: HallSource;
  createdAt: string;
  updatedAt: string;
}

export interface HallTierCount {
  tierManWon: HallTier | null;
  count: number;
}

export interface HallEntryPage {
  items: HallEntry[];
  total: number;
  page: number;
  size: number;
  /** Counts for the current filters; tierManWon is null for unlisted rows. */
  tierCounts: HallTierCount[];
}

export interface HallEntryFilters {
  tier: HallTier | '';
  kind: HallKind | '';
  q: string;
}

/** Fields an operator edits in the add/edit modal; null clears the value. */
export interface HallEntryInput {
  displayName: string;
  kind: HallKind;
  cohort: number | null;
  department: string | null;
  tierManWon: HallTier | null;
  contactName: string | null;
  roleLabel: string | null;
}

export type HallEntryCreateRequest = HallEntryInput & {
  visible: boolean;
  consentPublic: boolean;
};

export type HallEntryPatch = Partial<HallEntryInput & {
  visible: boolean;
  consentPublic: boolean;
}>;

/** One spreadsheet-derived row. It never carries a phone number or phone hash. */
export interface HallImportRow {
  id?: number;
  rowIndex?: number;
  displayName: string;
  kind: HallKind;
  cohort: number | null;
  department: string | null;
  tierManWon: HallTier | null;
}

export interface HallImportChange {
  field: string;
  before: string | number | null;
  after: string | number | null;
}

export interface HallImportUpdatedRow {
  id: number;
  displayName: string;
  changes: HallImportChange[];
}

export interface HallImportIssue {
  rowIndex: number | null;
  field?: string;
  code?: string;
  message: string;
}

export interface HallImportPreview {
  /** Empty while the file has blocking errors, so such a file cannot be committed. */
  previewToken: string;
  added: HallImportRow[];
  updated: HallImportUpdatedRow[];
  unchangedCount: number;
  removalCandidates: (HallImportRow & { id: number })[];
  warnings: HallImportIssue[];
  errors: HallImportIssue[];
}

export interface HallImportCommitRequest {
  previewToken: string;
  removeIds: number[];
}

export interface HallImportCommitResult {
  addedCount: number;
  updatedCount: number;
  removedCount: number;
}
