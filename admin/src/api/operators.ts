// operators — root-only API for listing, granting, changing and revoking admin roles
import { api } from './client.ts';

export type AdminRole = 'root' | 'operator';

export interface AdminOperator {
  usrSeq: number;
  usrId: string;
  usrName: string;
  adminRole: AdminRole;
  createdAt: string;
  updatedAt: string;
  updatedByName: string;
}

export function fetchOperators() {
  return api.get<{ items: AdminOperator[] }>('/api/admin/operators');
}

export function setOperatorRole(usrSeq: number, role: AdminRole) {
  return api.put<void>(`/api/admin/operators/${usrSeq}`, { role });
}

export function revokeOperator(usrSeq: number) {
  return api.del<void>(`/api/admin/operators/${usrSeq}`);
}
