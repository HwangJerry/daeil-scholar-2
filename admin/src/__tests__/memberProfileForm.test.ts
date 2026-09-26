import { describe, expect, it } from 'vitest';
import {
  createMemberProfileFormValues,
  toMemberProfileInput,
  validateMemberProfileForm,
  type MemberProfileFormValues,
} from '../components/member/memberProfileForm.ts';
import type { AdminMemberDetail } from '../types/api.ts';

const MEMBER: AdminMemberDetail = {
  usrSeq: 42, usrId: 'hong', usrName: '홍길동', usrStatus: 'CCC', usrFn: '30', usrDept: '국제어과',
  usrPhone: '01011112222', usrEmail: null, usrNick: null, usrPhoto: null, regDate: null, visitCnt: 0, visitDate: null,
};

const VALID: MemberProfileFormValues = { usrName: '홍길동', usrPhone: '010-3333-4444', usrEmail: 'm@example.com', usrFn: '30', usrDept: '영어' };

describe('memberProfileForm', () => {
  it('starts from the member and drops a department outside the canonical list', () => {
    expect(createMemberProfileFormValues(MEMBER)).toEqual({
      usrName: '홍길동', usrPhone: '01011112222', usrEmail: '', usrFn: '30', usrDept: '',
    });
  });

  it('accepts a valid correction and trims the request', () => {
    expect(validateMemberProfileForm(VALID)).toEqual({});
    expect(toMemberProfileInput({ ...VALID, usrName: ' 홍길동 ' }).usrName).toBe('홍길동');
  });

  it.each([
    ['usrName', { usrName: '  ' }],
    ['usrPhone', { usrPhone: '12' }],
    ['usrPhone', { usrPhone: '010-abcd-1234' }],
    ['usrEmail', { usrEmail: 'not-an-email' }],
    ['usrFn', { usrFn: '0' }],
    ['usrFn', { usrFn: '30기' }],
    ['usrFn', { usrFn: '100' }],
    ['usrDept', { usrDept: '경영학과' }],
  ] as const)('rejects an invalid %s', (field, change) => {
    expect(validateMemberProfileForm({ ...VALID, ...change })).toHaveProperty(field);
  });

  it('allows leaving phone, email, cohort and department empty', () => {
    expect(validateMemberProfileForm({ usrName: '홍길동', usrPhone: '', usrEmail: '', usrFn: '', usrDept: '' })).toEqual({});
  });
});
