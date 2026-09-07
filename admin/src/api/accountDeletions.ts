// accountDeletions — Durable manual account-erasure queue and verification API.
import { api } from './client';

export type DeletionStatus = 'pending' | 'processing' | 'completed';
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
}
export interface DeletionEvidence {
  action: 'complete';
  resultNotified: boolean;
  filesErased: boolean;
  backupsErased: boolean;
  externalDataErased: boolean;
  otherIdentifiersChecked: boolean;
  evidenceReference: string;
  retainedRecords: string;
  retentionUntil: string;
}
export interface DeletionFootprint { table: string; column: string; count: number }
export function fetchAccountDeletions(status: DeletionStatus, before: number) {
  return api.get<{ items: AccountDeletion[] }>(`/api/admin/account-deletions?status=${status}&before=${before}`);
}
export function verifyAccountDeletion(id: number) {
  return api.get<{ items: DeletionFootprint[] }>(`/api/admin/account-deletions/${id}/verification`);
}
export function resolveAccountDeletion(id: number, evidence: DeletionEvidence | ReceiptWorkResolution | { action: 'start' | 'automatic' | 'manual' }) {
  return api.put<void>(`/api/admin/account-deletions/${id}`, evidence);
}
