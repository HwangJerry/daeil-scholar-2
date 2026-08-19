// API contract types for Admin SPA — mirrors backend model/admin.go + shared user types

export interface APIError {
  code: string;
  message: string;
  errors?: APIFieldError[];
}

export interface APIFieldError {
  rowIndex: number;
  field: string;
  code: string;
  message: string;
}

export interface AuthUser {
  usrSeq: number;
  usrId: string;
  usrName: string;
  usrStatus: string;
}

// --- Notice ---

export interface AdminNoticeListItem {
  seq: number;
  subject: string;
  regDate: string;
  regName: string;
  hit: number;
  openYn: string;
  isPinned: string;
  contentFormat: 'LEGACY' | 'MARKDOWN';
}

export interface AdminNoticeListResponse {
  items: AdminNoticeListItem[];
  total: number;
}

export interface NoticeDetail {
  seq: number;
  subject: string;
  contentHtml: string;
  contentFormat: 'LEGACY' | 'MARKDOWN';
  contentMd?: string;
  summary: string;
  thumbnailUrl: string | null;
  regDate: string;
  regName: string;
  hit: number;
  likeCnt: number;
  commentCnt: number;
  isPinned: string;
  files: FileAttachment[];
}

export interface FileAttachment {
  fSeq: number;
  fGate: string;
  fJoinSeq: number;
  typeName: string;
  fileName: string;
  fileSize: string;
  filePath: string;
  fileOrgName: string;
  openYn: string;
}

export interface CreateNoticeRequest {
  subject: string;
  contentMd: string;
  isPinned?: string;
  attachedFileSeqs?: number[];
}

export interface UpdateNoticeRequest {
  subject: string;
  contentMd: string;
  isPinned?: string;
  attachedFileSeqs?: number[];
}

// --- Disclosure (공익법인 의무공시) ---

export interface AdminDisclosureListItem {
  seq: number;
  subject: string;
  regDate: string;
  regName: string;
  hit: number;
  openYn: string;
  contentFormat: 'LEGACY' | 'MARKDOWN';
}

export interface AdminDisclosureListResponse {
  items: AdminDisclosureListItem[];
  total: number;
}

export interface CreateDisclosureRequest {
  subject: string;
  contentMd: string;
  attachedFileSeqs?: number[];
}

export interface UpdateDisclosureRequest {
  subject: string;
  contentMd: string;
  attachedFileSeqs?: number[];
}

// --- Ad ---

export interface AdminAdListItem {
  maSeq: number;
  maName: string | null;
  maUrl: string | null;
  maImg: string | null;
  maStatus: string;
  adTier: 'PREMIUM' | 'GOLD' | 'NORMAL';
  adTitleLabel: string;
  maIndx: number;
  adStartDate: string | null;
  adEndDate: string | null;
}

export interface AdminAdCreateRequest {
  maName: string;
  maUrl: string;
  maImg: string;
  maStatus: string;
  adTier: string;
  adTitleLabel: string;
  maIndx: number;
  adStartDate?: string;
  adEndDate?: string;
}

export interface AdminAdUpdateRequest {
  maName: string;
  maUrl: string;
  maImg: string;
  maStatus: string;   // 'Y' | 'N'
  adTier: string;     // 'PREMIUM' | 'GOLD' | 'NORMAL'
  adTitleLabel: string;
  maIndx: number;
  adStartDate?: string;
  adEndDate?: string;
}

export interface AdminAdStatsItem {
  maSeq: number;
  viewCount: number;
  clickCount: number;
}

// --- Donation ---

export interface DonationConfig {
  dcSeq: number;
  dcGoal: number;
  dcManualAdj: number;
  dcManualDonorCnt: number;
  dcTierSproutMin: number;
  dcTierSaplingMin: number;
  dcTierTreeMin: number;
  dcTierBloomingMin: number;
  dcTierFruitingMin: number;
  dcNote: string;
  dcOverwrite: string; // "Y" | "N"
  isActive: string;
  regDate: string;
}

export interface DonationConfigUpdateRequest {
  goal: number;
  manualAdj: number;
  manualDonorCnt: number;
  tierSproutMin: number;
  tierSaplingMin: number;
  tierTreeMin: number;
  tierBloomingMin: number;
  tierFruitingMin: number;
  note: string;
  overwrite: boolean;
}

export interface DonationSnapshot {
  dsDate: string;
  dsTotal: number;
  dsManualAdj: number;
  dsDonorCnt: number;
  dsGoal: number;
}

// --- Member ---

export interface AdminMemberListItem {
  usrSeq: number;
  usrId: string;
  usrName: string;
  usrStatus: string;
  usrFn: string | null;
  usrPhone: string | null;
  usrEmail: string | null;
  usrDept: string | null;
  regDate: string | null;
  visitDate: string | null;
}

export interface AdminMemberListResponse {
  items: AdminMemberListItem[];
  total: number;
}

export interface AdminMemberDetail {
  usrSeq: number;
  usrId: string;
  usrName: string;
  usrStatus: string;
  usrFn: string | null;
  usrPhone: string | null;
  usrEmail: string | null;
  usrNick: string | null;
  usrPhoto: string | null;
  regDate: string | null;
  visitCnt: number;
  visitDate: string | null;
}

export interface AdminMemberDetailResponse {
  member: AdminMemberDetail;
  kakaoLinked: boolean;
}

export interface AdminMemberStats {
  totalMembers: number;
  kakaoLinkedMembers: number;
  recentLoginCount: number;
  statusBreakdown: Record<string, number>;
}

// --- Upload ---

export interface UploadResponse {
  url: string;
  width?: number;
  height?: number;
  fSeq?: number;
}

// --- Dashboard ---

export interface DashboardStats {
  totalMembers: number;
  kakaoLinkedMembers: number;
  /** @deprecated Prefer dauToday / mauCurrent. */
  recentLoginCount: number;
  pendingApprovals: number;
  totalNotices: number;
  dauToday: number;
  mauCurrent: number;
  donation: DonationSummary;
  adStats: DashboardAdStats;
}

// --- Active Users (DAU/MAU) ---

export interface ActiveUsersPoint {
  date: string;
  dau: number;
  mau: number;
}

export interface ActiveUsersResponse {
  points: ActiveUsersPoint[];
  dauToday: number;
  mauCurrent: number;
}

export interface DonationSummary {
  displayAmount: number;
  goalAmount: number;
  donorCount: number;
  achievementRate: number;
  snapshotDate: string;
  tierThresholds: {
    sprout: number;
    sapling: number;
    tree: number;
    blooming: number;
    fruiting: number;
  };
}

export interface DashboardAdStats {
  totalImpressions: number;
  totalClicks: number;
  ctr: number;
}

// --- Donation Order ---

export type DonationSource = 'happy_nanum' | 'bank_transfer' | 'other';

export type DonationType = 'recurring' | 'one_time' | 'sponsorship';

export type DonationStatus =
  | 'scheduled'
  | 'pending'
  | 'completed'
  | 'partially_refunded'
  | 'cancelled'
  | 'fully_refunded';

export type DonationPaymentMethod =
  | 'card'
  | 'bank'
  | 'virtual_bank'
  | 'mobile'
  | 'admin'
  | 'other';

export interface DonationDonor {
  name: string;
  cohort: string;
  department: string;
  phone: string;
}

export interface DonationOrderInput {
  source: DonationSource;
  accountUsrSeq: number | null;
  transactionNumber: string | null;
  donationDate: string;
  donor: DonationDonor;
  donationType: DonationType;
  grossAmount: number;
  refundedAmount: number;
  status: DonationStatus;
  paymentMethod: DonationPaymentMethod;
  memo: string | null;
}

export interface DonationOrder {
  orderSeq: number;
  accountUsrSeq: number | null;
  source: DonationSource;
  transactionNumber: string | null;
  donationDate: string;
  donor: DonationDonor;
  donationType: DonationType;
  grossAmount: number;
  refundedAmount: number;
  netReceivedAmount: number;
  status: DonationStatus;
  paymentMethod: DonationPaymentMethod;
  memo: string | null;
  lastEditedBy: number;
  lastEditedAt: string;
  lastEditedIp: string;
}

export interface DonationOrderFilters {
  name: string;
  phone: string;
  transactionNumber: string;
  source: DonationSource | '';
  status: DonationStatus | '';
  donationType: DonationType | '';
}

export interface DonationOrderPage {
  items: DonationOrder[];
  total: number;
  page: number;
  size: number;
}

// --- Donation Import ---

export interface ExcelDonationRow {
  rowIndex: number;
  donorName: string;
  donorCohort: string;
  donorDepartment: string;
  donorPhone: string;
  amount: number;
}

export type DonationImportRowStatus =
  | 'matched'
  | 'ambiguous'
  | 'unmatched'
  | 'duplicate';

export interface DonationImportPreviewRow extends ExcelDonationRow {
  donationDate: string;
  status: DonationImportRowStatus;
  matchedUsrSeq: number | null;
  matchedName: string;
  note: string;
  previewToken: string;
}

export interface DonationImportPreviewResult {
  rows: DonationImportPreviewRow[];
  matchedCount: number;
  ambiguousCount: number;
  unmatchedCount: number;
  duplicateCount: number;
}

export interface DonationImportCommitRow extends ExcelDonationRow {
  accountUsrSeq: number | null;
  previewToken: string;
}

export interface DonationImportCommitRequest {
  donationDate: string;
  rows: DonationImportCommitRow[];
}

export interface DonationImportCommitRowResult {
  rowIndex: number;
  success: boolean;
  orderSeq: number | null;
  errorMessage: string;
}

export interface DonationImportCommitResult {
  rows: DonationImportCommitRowResult[];
}

// --- Job Category ---

export interface AdminJobCategory {
  seq: number;
  name: string;
  index: number;
  openYn: 'Y' | 'N';
}

export interface AdminJobCategoryUpsert {
  name: string;
  openYn: 'Y' | 'N';
}

export interface AdminJobCategoryReorderRequest {
  order: number[];
}

// --- History ---

export interface HistoryEntry {
  heSeq: number;
  eventDate: string; // "YYYY-MM-DD"
  text: string;
  sortOrder: number;
}

export interface HistoryUpsertRequest {
  eventDate: string;
  text: string;
  sortOrder: number;
}
