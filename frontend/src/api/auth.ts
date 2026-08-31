// auth.ts — Authentication API client functions.
import { api } from './client';
import type {
  AccountConnections,
  AuthUser,
  LoginRequest,
  SocialDisconnectResponse,
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

/** Fetch the login methods currently connected to the authenticated account. */
export function getAccountConnections(): Promise<AccountConnections> {
  return api.get<AccountConnections>('/api/auth/account/connections');
}

/** Disconnect a social login method from the authenticated account. */
export function disconnectSocial(
  provider: 'kakao' | 'apple',
): Promise<SocialDisconnectResponse> {
  return api.del<SocialDisconnectResponse>(`/api/auth/social/${provider}`);
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
