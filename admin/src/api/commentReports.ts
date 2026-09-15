// commentReports — API contract for the restricted message moderation queue.
import { api } from './client';

export type ReportStatus = 'open' | 'removed' | 'dismissed';
export interface CommentReport {
  id: number;
  postId: number;
  commentId: number;
  visible: boolean;
  reporterSeq: number;
  reportedSeq: number;
  reason: string;
  details: string;
  content: string;
  status: ReportStatus;
  moderatorNote: string;
  createdAt: string;
}

export function fetchCommentReports(status: ReportStatus, before: number) {
  return api.get<{ items: CommentReport[] }>(`/api/admin/comment-reports?status=${status}&before=${before}`);
}

export function resolveCommentReport(id: number, status: Exclude<ReportStatus, 'open'>, note: string) {
  return api.put<void>(`/api/admin/comment-reports/${id}`, { status, note });
}
