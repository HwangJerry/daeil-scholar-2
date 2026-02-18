// auth.ts — Authentication API client functions.
import { api } from './client';
import type { AuthUser, LoginRequest } from '../types/api';

/** Legacy ID/PW login. */
export function legacyLogin(req: LoginRequest): Promise<AuthUser> {
  return api.post<AuthUser>('/api/auth/login', req);
}
