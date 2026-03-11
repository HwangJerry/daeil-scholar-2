// useRegisterFormValidation.ts — Client-side validation rules for the registration form.
const USR_ID_REGEX = /^[a-zA-Z0-9]{4,20}$/;
const MIN_PASSWORD_LENGTH = 6;

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
    if (fields.password.length < MIN_PASSWORD_LENGTH) {
      return `비밀번호는 ${MIN_PASSWORD_LENGTH}자 이상이어야 합니다.`;
    }
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
