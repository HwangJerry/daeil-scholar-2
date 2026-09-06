// messageReports — API contract for the restricted message moderation queue.
import { api } from './client';

export type ReportStatus = 'open' | 'removed' | 'dismissed';
export interface MessageReport {
  id: number;
  messageId: number;
  reporterSeq: number;
  reportedSeq: number;
  reason: string;
  details: string;
  content: string;
  status: ReportStatus;
  moderatorNote: string;
  createdAt: string;
}

export function fetchMessageReports(status: ReportStatus, before: number) {
  return api.get<{ items: MessageReport[] }>(`/api/admin/message-reports?status=${status}&before=${before}`);
}

export function resolveMessageReport(id: number, status: Exclude<ReportStatus, 'open'>, note: string) {
  return api.put<void>(`/api/admin/message-reports/${id}`, { status, note });
}
