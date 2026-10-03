// Notice API contract types for Admin SPA — mirrors backend model/admin.go notice rows and admin feed requests
import type { FileAttachment } from './api.ts';

export interface AdminNoticeListItem {
  seq: number;
  subject: string;
  regDate: string;
  regName: string;
  hit: number;
  openYn: string;
  isPinned: string;
  contentFormat: 'LEGACY' | 'MARKDOWN';
  categorySeq: number;
  categoryName: string;
  /** True when the post shows the foundation's official profile (대일외고장학회). */
  officialProfile: boolean;
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
  categorySeq: number;
  categoryName: string;
  /** True when the post shows the foundation's official profile (대일외고장학회). */
  officialProfile: boolean;
  files: FileAttachment[];
}

export interface CreateNoticeRequest {
  subject: string;
  contentMd: string;
  isPinned?: string;
  attachedFileSeqs?: number[];
  categorySeq?: number;
  /** Omitted means true: new posts default to the official profile. */
  officialProfile?: boolean;
}

export interface UpdateNoticeRequest {
  subject: string;
  contentMd: string;
  isPinned?: string;
  attachedFileSeqs?: number[];
  categorySeq?: number;
  /** Omitted keeps the stored byline. */
  officialProfile?: boolean;
}
