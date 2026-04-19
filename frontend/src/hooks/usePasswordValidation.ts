// usePasswordValidation — Client-side validation for the password change form.
const MIN_PASSWORD_LENGTH = 8;
const HAS_LETTER = /[a-zA-Z]/;
const HAS_NUMBER = /[0-9]/;
const HAS_SPECIAL = /[^a-zA-Z0-9]/;

/** Returns an error message if the password fails strength requirements, or null if it passes. */
export function checkPasswordStrength(password: string): string | null {
  if (password.length < MIN_PASSWORD_LENGTH) {
    return `비밀번호는 ${MIN_PASSWORD_LENGTH}자 이상이어야 합니다.`;
  }
  if (!HAS_LETTER.test(password) || !HAS_NUMBER.test(password) || !HAS_SPECIAL.test(password)) {
    return '비밀번호는 영문, 숫자, 특수문자를 모두 포함해야 합니다.';
  }
  return null;
}

interface PasswordFields {
  newPassword: string;
  confirmPassword: string;
}

type ValidationResult = { ok: true } | { ok: false; reason: string };

export function usePasswordValidation() {
  function validate({ newPassword, confirmPassword }: PasswordFields): ValidationResult {
    const strengthError = checkPasswordStrength(newPassword);
    if (strengthError) return { ok: false, reason: strengthError };
    if (newPassword !== confirmPassword) {
      return { ok: false, reason: '새 비밀번호가 일치하지 않습니다.' };
    }
    return { ok: true };
  }

  return { validate };
}
