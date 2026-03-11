// api.ts — Shared API request/response type definitions
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

export interface NoticeItem {
  type: 'notice';
  seq: number;
  subject: string;
  summary: string;
  thumbnailUrl: string | null;
  regDate: string;
  regName: string;
  hit: number;
  likeCnt: number;
  commentCnt: number;
  isPinned: string;
  userLiked: boolean;
  category?: string;
}

export interface AdItem {
  type: 'ad';
  maSeq: number;
  maName: string;
  maUrl: string;
  imageUrl: string;
  adTier: 'PREMIUM' | 'GOLD' | 'NORMAL';
  titleLabel: string;
  sponsor?: string;
  cta?: string;
  regDate?: string;
  likeCnt: number;
  commentCnt: number;
  hit: number;
  userLiked: boolean;
}

export interface AdComment {
  acSeq: number;
  maSeq: number;
  usrSeq: number;
  nickname: string;
  contents: string;
  regDate: string;
}

export type FeedItem = NoticeItem | AdItem;

export interface FeedResponse {
  items: FeedItem[];
  nextCursor: string;
  hasMore: boolean;
}

export interface HeroNotice {
  seq: number;
  subject: string;
  summary: string;
  thumbnailUrl: string | null;
  regDate: string;
  regName: string;
  hit: number;
  likeCnt: number;
  commentCnt: number;
  isPinned: string;
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
  userLiked: boolean;
  files: FileAttachment[];
}

export interface LikeToggleResponse {
  liked: boolean;
  likeCnt: number;
}

export interface Comment {
  bcSeq: number;
  joinSeq: number;
  usrSeq: number;
  regName: string;
  contents: string;
  regDate: string;
}

export interface CommentCreateRequest {
  contents: string;
}

export interface PostSibling {
  seq: number;
  subject: string;
}

export interface PostSiblings {
  prev: PostSibling | null;
  next: PostSibling | null;
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

export interface AlumniItem {
  fmSeq: number;
  fmName: string;
  fmFn: string;
  fmDept: string;
  company: string;
  position: string;
  phone: string;
  email: string;
  bizName: string;
  bizDesc: string;
  bizAddr: string;
  jobCatName: string;
  jobCatColor: string;
  tags: string[];
  photo: string;
  usrSeq: number;
}

export interface AlumniSearchResponse {
  items: AlumniItem[];
  totalCount: number;
  weeklyCount: number;
  page: number;
  size: number;
  totalPages: number;
}

export interface JobCategory {
  seq: number;
  name: string;
  color: string;
}

export interface AlumniFilters {
  fnList: string[];
  deptList: string[];
  jobCategories: JobCategory[];
}

export interface UserProfile {
  usrSeq: number;
  usrName: string;
  usrNick: string;
  usrPhone: string;
  usrEmail: string;
  usrFn: string;
  usrPhoto: string | null;
  bizName: string;
  bizDesc: string;
  bizAddr: string;
  jobCat: number;
  jobCatName: string;
  jobCatColor: string;
  tags: string[];
  fmDept: string;
  regDate: string;
  usrPhonePublic: 'Y' | 'N';
  usrEmailPublic: 'Y' | 'N';
  usrBizCard: string | null;
}

export interface ProfileUpdateRequest {
  usrNick: string;
  usrPhone: string;
  usrEmail: string;
  bizName: string;
  bizDesc: string;
  bizAddr: string;
  jobCat: number | null;
  tags: string[];
  usrPhonePublic: 'Y' | 'N';
  usrEmailPublic: 'Y' | 'N';
}

export interface SocialLinkRequest {
  token: string;
  name: string;
  phone: string;
  fn: string;
  nick?: string;
  dept?: string;
  jobCat?: number | null;
  bizName?: string;
  bizDesc?: string;
  bizAddr?: string;
}

export type KakaoLinkRequest = SocialLinkRequest;

export interface LoginRequest {
  usrId: string;
  password: string;
}

export interface AlumniWidgetItem {
  fmName: string;
}

export interface AlumniWidgetResponse {
  items: AlumniWidgetItem[];
  totalCount: number;
}

export interface MessageItem {
  amSeq: number;
  senderSeq: number;
  recvrSeq: number;
  content: string;
  readYn: string;
  regDate: string;
  readDate: string;
  senderName: string;
  recvrName: string;
}

export interface MessageListResponse {
  items: MessageItem[];
  totalCount: number;
  page: number;
  size: number;
  totalPages: number;
}

export interface NotificationItem {
  anSeq: number;
  usrSeq: number;
  anType: string;
  anTitle: string;
  anBody: string;
  anRefSeq: number | null;
  readYn: string;
  regDate: string;
}

export interface NotificationListResponse {
  items: NotificationItem[];
  totalCount: number;
  unreadCount: number;
  page: number;
  size: number;
  totalPages: number;
}

export interface BadgeResponse {
  unreadMessages: number;
  unreadNotifications: number;
}
