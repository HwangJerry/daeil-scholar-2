// passwordReset.ts — API client functions for password reset flow
import { api } from './client';

export function requestPasswordReset(email: string) {
  return api.post<{ status: string }>('/api/auth/password/reset-request', { email });
}

export function validateResetToken(token: string) {
  return api.get<{ valid: boolean; name?: string }>(
    `/api/auth/password/validate-token?token=${encodeURIComponent(token)}`,
  );
}

export function confirmPasswordReset(token: string, newPassword: string) {
  return api.post<{ status: string }>('/api/auth/password/reset-confirm', {
    token,
    newPassword,
  });
}
