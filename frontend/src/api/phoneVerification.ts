// phoneVerification.ts — API client for the signup SMS phone-verification endpoints.
import { api } from './client';

export interface PhoneVerificationRequestResult {
  verificationId: string;
  expiresInSec: number;
}

export interface PhoneVerificationConfirmResult {
  verificationToken: string;
  expiresInSec: number;
}

/** Send a one-time code by SMS to the given phone number. */
export function requestPhoneVerification(
  phone: string,
): Promise<PhoneVerificationRequestResult> {
  return api.post<PhoneVerificationRequestResult>(
    '/api/auth/phone/verification/request',
    { phone },
  );
}

/** Exchange a received code for the short-lived token that signup requires. */
export function confirmPhoneVerification(
  verificationId: string,
  code: string,
): Promise<PhoneVerificationConfirmResult> {
  return api.post<PhoneVerificationConfirmResult>(
    '/api/auth/phone/verification/confirm',
    { verificationId, code },
  );
}
