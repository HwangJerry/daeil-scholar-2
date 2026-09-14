// accountDeletions — Account-erasure queue and operator controls API.
import { api } from './client';

export type DeletionStatus = 'pending' | 'processing' | 'completed' | 'cancelled';
export interface ErasureTarget {
 target: string;
 status: 'pending' | 'running' | 'complete' | 'not_applicable' | 'manual' | 'failed';
 evidenceReference: string;
 code: string;
 attempts: number;
 lastAttemptAt: string | null;
 updatedAt: string;
}
export interface ReceiptWorkResolution {
 action: 'receipt_work';
 receiptWorkStatus: 'not_required' | 'active' | 'completed';
 originalStorage: string;
 evidenceReference: string;
 contactSecured: boolean;
 contactErased: boolean;
 resultNotified: boolean;
}
export interface ReceiptWork {
 status: 'unreviewed' | 'not_required' | 'active' | 'completed';
 originalStorage: string;
 evidenceReference: string;
 updatedAt: string;
 completedAt: string | null;
}
export interface AccountDeletion {
 canCancel?: boolean;
 cancelledAt?: string | null;
 scheduledAt?: string | null;
 expeditedAt?: string | null;
 needsAttention?: boolean;
 contextExpiresAt?: string;
 receiptWork?: ReceiptWork;
 receiptWorkPending?: boolean;
 targets?: ErasureTarget[];
  requestId: number;
  userSeq: number | null;
  status: DeletionStatus;
  requestedAt: string;
  targetAt: string;
  dueAt: string;
  completedAt: string | null;
  retainedRecords: string;
  retentionUntil: string | null;
  evidenceReference: string;
  processingMode?: "automatic" | "manual";
  autoStage?: string;
  databaseErased?: boolean;
  nextAttemptAt?: string | null;
  automationUpdatedAt?: string | null;
  autoCode?: string;
  socialUnlinkStalled?: boolean;
}
export function fetchAccountDeletions(status: DeletionStatus, before: number) {
  return api.get<{ items: AccountDeletion[] }>(`/api/admin/account-deletions?status=${status}&before=${before}`);
}
export function resolveAccountDeletion(id: number, evidence: ReceiptWorkResolution | { action: 'automatic' | 'manual' | 'schedule' | 'retry_social' } | { action: 'cancel_verified'; evidenceReference: string } | { action: 'target'; target: string; targetStatus: 'manual' | 'complete' | 'not_applicable'; evidenceReference: string }) {
  return api.put<void>(`/api/admin/account-deletions/${id}`, evidence);
}

export interface ProxyIntakeResult {
  receipt: AccountDeletion;
  receiptToken: string;
  cancelToken: string;
}
// Registers an emailed request after identity verification; tokens are shown once.
export function createAccountDeletionOnBehalf(userSeq: number, evidenceReference: string) {
  return api.post<ProxyIntakeResult>('/api/admin/account-deletions', { userSeq, evidenceReference });
}
