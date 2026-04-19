// auth.ts — Admin authentication API calls
import { api } from './client.ts';
import type { AuthUser } from '../types/api.ts';

export function adminLogin(req: { usrId: string; password: string }): Promise<AuthUser> {
  return api.post<AuthUser>('/api/auth/login', req);
}
