// accountDeletions — Durable manual account-erasure queue and verification API.
import { api } from './client';

export type DeletionStatus = 'pending' | 'processing' | 'completed';
export interface AccountDeletion {
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
export function resolveAccountDeletion(id: number, evidence: DeletionEvidence | { action: 'start' }) {
  return api.put<void>(`/api/admin/account-deletions/${id}`, evidence);
}
