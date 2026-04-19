// useRegisterFormValidation.ts — Client-side validation rules for the registration form.
import { checkPasswordStrength } from './usePasswordValidation';

const USR_ID_REGEX = /^[a-zA-Z0-9]{4,20}$/;

interface RegisterValidationFields {
  usrId: string;
  password: string;
  passwordConfirm: string;
  email: string;
}

/** Returns a validate function that checks required registration fields.
 *  Returns an error string on failure, or null if all checks pass. */
export function useRegisterFormValidation() {
  const validate = (fields: RegisterValidationFields): string | null => {
    if (!USR_ID_REGEX.test(fields.usrId)) {
      return '아이디는 영문자와 숫자로 이루어진 4~20자여야 합니다.';
    }
    const strengthError = checkPasswordStrength(fields.password);
    if (strengthError) return strengthError;
    if (fields.password !== fields.passwordConfirm) {
      return '비밀번호가 일치하지 않습니다.';
    }
    if (!fields.email.includes('@')) {
      return '올바른 이메일 주소를 입력해주세요.';
    }
    return null;
  };

  return { validate };
}
