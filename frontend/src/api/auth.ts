// auth.ts — Authentication API client functions.
import { api } from './client';
import type {
  AuthUser,
  LoginRequest,
  SocialLinkPrefillResponse,
  SocialLinkPhotoUploadResponse,
} from '../types/api';

export interface RegisterRequest {
  usrId: string;
  password: string;
  name: string;
  phone: string;
  fn: string;
  email: string;
  fmDept?: string;
  jobCat?: number | null;
  bizName?: string;
  bizDesc?: string;
  bizAddr?: string;
  position?: string;
  tags?: string[];
  usrPhonePublic?: 'Y' | 'N';
  usrEmailPublic?: 'Y' | 'N';
}

/** Legacy ID/PW login. */
export function legacyLogin(req: LoginRequest): Promise<AuthUser> {
  return api.post<AuthUser>('/api/auth/login', req);
}

/** New member registration with ID/password. */
export function register(req: RegisterRequest): Promise<AuthUser> {
  return api.post<AuthUser>('/api/auth/register', req);
}

/** Fetch cached social-provider data (email/nickname/profileImage) for the signup form. */
export function getSocialLinkPrefill(token: string): Promise<SocialLinkPrefillResponse> {
  return api.get<SocialLinkPrefillResponse>(
    `/api/auth/social/link/prefill?token=${encodeURIComponent(token)}`,
  );
}

/** Upload a replacement profile photo during the pre-signup flow (token-gated, no DB write). */
export function uploadSocialLinkPhoto(
  token: string,
  file: File,
): Promise<SocialLinkPhotoUploadResponse> {
  const form = new FormData();
  form.append('token', token);
  form.append('file', file);
  return api.upload<SocialLinkPhotoUploadResponse>('/api/auth/social/link/photo', form);
}
