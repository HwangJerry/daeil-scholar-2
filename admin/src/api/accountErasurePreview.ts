// accountErasurePreview — Read-only erasure plan and plan-bound immediate processing.
import { api } from './client';

export interface ErasurePreviewRow {
  before: (string | null)[];
  after?: (string | null)[];
  note?: string;
}
export interface ErasurePreviewTable {
  table: string;
  action: 'delete' | 'anonymize';
  columns: string[];
  maskedColumns: string[];
  changedColumns: string[];
  count: number;
  rows: ErasurePreviewRow[];
}
export interface ErasureUnhandledReference { table: string; column: string; count: number }
export interface ErasurePreview {
  requestId: number;
  generatedAt: string;
  planDigest: string;
  blockers: string[];
  tables: ErasurePreviewTable[];
  files: string[];
  unhandled: ErasureUnhandledReference[];
}

export function fetchErasurePreview(id: number) {
  return api.get<ErasurePreview>(`/api/admin/account-deletions/${id}/preview`);
}
// The server re-reads the records and refuses if they differ from this reviewed plan.
export function expediteReviewedErasure(id: number, reviewedPlanDigest: string) {
  return api.put<void>(`/api/admin/account-deletions/${id}`, { action: 'expedite', reviewedPlanDigest });
}
