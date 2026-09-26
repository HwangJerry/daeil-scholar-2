// memberProfileForm — form values, validation and request mapping for the admin member profile editor
import { isValidDepartment } from '../../constants/departments.ts';
import type { AdminMemberDetail, AdminMemberProfileInput } from '../../types/api.ts';

export interface MemberProfileFormValues {
  usrName: string;
  usrPhone: string;
  usrEmail: string;
  usrFn: string;
  usrDept: string;
}

export type MemberProfileFormErrors = Partial<Record<keyof MemberProfileFormValues, string>>;

// Limits mirror backend validation (WEO_MEMBER column sizes, canonical phone digits).
const NAME_MAX_CHARS = 100;
const EMAIL_MAX_CHARS = 200;
const COHORT_MAX = 99;
const PHONE_MIN_DIGITS = 7;
const PHONE_MAX_DIGITS = 15;
const PHONE_ALLOWED_PATTERN = /^[0-9\-\s]+$/;
const EMAIL_PATTERN = /^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/;

export function createMemberProfileFormValues(member: AdminMemberDetail): MemberProfileFormValues {
  return {
    usrName: member.usrName ?? '',
    usrPhone: member.usrPhone ?? '',
    usrEmail: member.usrEmail ?? '',
    usrFn: member.usrFn ?? '',
    usrDept: member.usrDept && isValidDepartment(member.usrDept) ? member.usrDept : '',
  };
}

export function validateMemberProfileForm(values: MemberProfileFormValues): MemberProfileFormErrors {
  const errors: MemberProfileFormErrors = {};
  const name = values.usrName.trim();
  const phone = values.usrPhone.trim();
  const email = values.usrEmail.trim();
  const cohort = values.usrFn.trim();

  if (!name || [...name].length > NAME_MAX_CHARS) {
    errors.usrName = `이름을 ${NAME_MAX_CHARS}자 이내로 입력해 주세요.`;
  }
  if (phone) {
    const digits = phone.replace(/\D/g, '');
    const isPhoneValid = PHONE_ALLOWED_PATTERN.test(phone)
      && digits.length >= PHONE_MIN_DIGITS
      && digits.length <= PHONE_MAX_DIGITS;
    if (!isPhoneValid) errors.usrPhone = '숫자와 하이픈으로 7~15자리 번호를 입력해 주세요.';
  }
  if (email && (!EMAIL_PATTERN.test(email) || email.length > EMAIL_MAX_CHARS)) {
    errors.usrEmail = '이메일 형식이 올바르지 않습니다.';
  }
  if (cohort) {
    const cohortNumber = Number(cohort);
    if (!/^\d+$/.test(cohort) || cohortNumber < 1 || cohortNumber > COHORT_MAX) {
      errors.usrFn = `기수는 1~${COHORT_MAX} 사이 숫자로 입력해 주세요.`;
    }
  }
  if (values.usrDept && !isValidDepartment(values.usrDept)) {
    errors.usrDept = '학과를 목록에서 선택해 주세요.';
  }
  return errors;
}

export function toMemberProfileInput(values: MemberProfileFormValues): AdminMemberProfileInput {
  return {
    usrName: values.usrName.trim(),
    usrPhone: values.usrPhone.trim(),
    usrEmail: values.usrEmail.trim(),
    usrFn: values.usrFn.trim(),
    usrDept: values.usrDept,
  };
}
