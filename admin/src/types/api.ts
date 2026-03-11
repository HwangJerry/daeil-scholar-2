// API contract types for Admin SPA — mirrors backend model/admin.go + shared user types

export interface APIError {
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
}

export interface UpdateNoticeRequest {
  subject: string;
  contentMd: string;
  isPinned?: string;
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
}

export interface AdminAdCreateRequest {
  maName: string;
  maUrl: string;
  maImg: string;
  maStatus: string;
  adTier: string;
  adTitleLabel: string;
  maIndx: number;
}

export interface AdminAdUpdateRequest {
  maName: string;
  maUrl: string;
  maImg: string;
  maStatus: string;   // 'Y' | 'N'
  adTier: string;     // 'PREMIUM' | 'GOLD' | 'NORMAL'
  adTitleLabel: string;
  maIndx: number;
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
  dcNote: string;
  isActive: string;
  regDate: string;
}

export interface DonationConfigUpdateRequest {
  goal: number;
  manualAdj: number;
  note: string;
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
}

// --- Dashboard ---

export interface DashboardStats {
  totalMembers: number;
  kakaoLinkedMembers: number;
  recentLoginCount: number;
  pendingApprovals: number;
  totalNotices: number;
  donation: DonationSummary;
  adStats: DashboardAdStats;
}

export interface DonationSummary {
  totalAmount: number;
  manualAdj: number;
  displayAmount: number;
  donorCount: number;
  goalAmount: number;
  achievementRate: number;
  snapshotDate: string;
}

export interface DashboardAdStats {
  totalImpressions: number;
  totalClicks: number;
  ctr: number;
}

// --- Donation Order ---

export interface AdminDonationOrderItem {
  oSeq: number;
  usrSeq: number;
  usrName: string;
  amount: number;
  payType: string;
  payment: string;
  payDate: string;
  gate: string;
  regDate: string;
}

export interface AdminDonationOrderListResponse {
  items: AdminDonationOrderItem[];
  total: number;
}

export interface AdminDonationOrderUpdateRequest {
  payment: string;
  amount: number;
}
