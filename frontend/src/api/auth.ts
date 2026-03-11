// auth.ts — Authentication API client functions.
import { api } from './client';
import type { AuthUser, LoginRequest } from '../types/api';

export interface RegisterRequest {
  usrId: string;
  password: string;
  name: string;
  phone: string;
  fn: string;
  email: string;
  nick?: string;
  dept?: string;
  jobCat?: number | null;
  bizName?: string;
  bizDesc?: string;
  bizAddr?: string;
}

/** Legacy ID/PW login. */
export function legacyLogin(req: LoginRequest): Promise<AuthUser> {
  return api.post<AuthUser>('/api/auth/login', req);
}

/** New member registration with ID/password. */
export function register(req: RegisterRequest): Promise<AuthUser> {
  return api.post<AuthUser>('/api/auth/register', req);
}

