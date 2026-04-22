// RegisterForm — ID/password registration form. Profile fields are shared via ProfileFieldsSection.
import { useState, useRef } from 'react';
import { ProfileFieldsSection, defaultProfileFieldValues } from './ProfileFieldsSection';
import type { ProfileFieldValues } from './ProfileFieldsSection';
import { useRegisterFormValidation } from '../../hooks/useRegisterFormValidation';
import { useRegisterSubmit } from '../../hooks/useRegisterSubmit';
import { useCheckUsrId } from '../../hooks/useCheckUsrId';
import { useCheckPhone } from '../../hooks/useCheckPhone';
import { useCheckEmail } from '../../hooks/useCheckEmail';
import { checkPasswordStrength } from '../../hooks/usePasswordValidation';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';

const DEBOUNCE_MS = 500;

export function RegisterForm() {
  const [usrId, setUsrId] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [profile, setProfile] = useState<ProfileFieldValues>(defaultProfileFieldValues);
  const [validationError, setValidationError] = useState('');

  const pwDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { validate } = useRegisterFormValidation();
  const { submit, error: submitError, submitting } = useRegisterSubmit();

  const idCheck = useCheckUsrId(usrId);
  const phoneCheck = useCheckPhone(profile.phone);
  const emailCheck = useCheckEmail(profile.email);

  const displayError = validationError || submitError;
  const passwordMismatch = passwordConfirm !== '' && password !== passwordConfirm;

  const validatePw = (value: string) => {
    setPasswordError(checkPasswordStrength(value) ?? '');
  };

  const handlePasswordChange = (value: string) => {
    setPassword(value);
    if (pwDebounceRef.current) clearTimeout(pwDebounceRef.current);
    pwDebounceRef.current = setTimeout(() => validatePw(value), DEBOUNCE_MS);
  };

  const handlePasswordBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    if (pwDebounceRef.current) clearTimeout(pwDebounceRef.current);
    validatePw(e.target.value);
  };

  const handleProfileChange = <K extends keyof ProfileFieldValues>(
    key: K,
    value: ProfileFieldValues[K],
  ) => {
    setProfile((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (idCheck.status !== 'available') {
      setValidationError('아이디 중복 확인을 완료해주세요.');
      return;
    }
    const validErr = validate({
      usrId,
      password,
      passwordConfirm,
      email: profile.email,
      fn: profile.fn,
      fmDept: profile.fmDept,
    });
    if (validErr) {
      setValidationError(validErr);
      return;
    }
    setValidationError('');
    await submit({
      usrId,
      password,
      name: profile.name,
      phone: profile.phone,
      fn: profile.fn,
      email: profile.email,
      fmDept: profile.fmDept,
      jobCat: profile.jobCat,
      bizName: profile.bizName,
      bizDesc: profile.bizDesc,
      bizAddr: profile.bizAddr,
      position: profile.position,
      tags: profile.tags,
      usrPhonePublic: profile.usrPhonePublic,
      usrEmailPublic: profile.usrEmailPublic,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">아이디 *</label>
        <Input
          type="text"
          value={usrId}
          onChange={(e) => setUsrId(e.target.value)}
          onBlur={idCheck.onBlur}
          required
          placeholder="영문자+숫자, 4~20자"
          autoComplete="username"
        />
        {idCheck.status === 'checking' && (
          <p className="mt-1 text-xs text-text-placeholder">확인 중...</p>
        )}
        {idCheck.status === 'available' && (
          <p className="mt-1 text-xs text-success-text">사용 가능한 아이디입니다.</p>
        )}
        {idCheck.status === 'unavailable' && (
          <p className="mt-1 text-xs text-error-text">이미 사용 중인 아이디입니다.</p>
        )}
        {idCheck.status === 'error' && (
          <p className="mt-1 text-xs text-error-text">확인에 실패했습니다. 다시 시도해주세요.</p>
        )}
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">비밀번호 *</label>
        <Input
          type="password"
          value={password}
          onChange={(e) => handlePasswordChange(e.target.value)}
          onBlur={handlePasswordBlur}
          required
          placeholder="8자 이상, 영문+숫자+특수문자 포함"
          autoComplete="new-password"
        />
        {passwordError && (
          <p className="mt-1 text-xs text-error-text">{passwordError}</p>
        )}
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">비밀번호 확인 *</label>
        <Input
          type="password"
          value={passwordConfirm}
          onChange={(e) => setPasswordConfirm(e.target.value)}
          required
          placeholder="비밀번호를 다시 입력하세요"
          autoComplete="new-password"
        />
        {passwordMismatch && (
          <p className="mt-1 text-xs text-error-text">비밀번호가 일치하지 않습니다.</p>
        )}
      </div>

      <ProfileFieldsSection
        values={profile}
        onChange={handleProfileChange}
        phoneCheck={phoneCheck}
        emailCheck={emailCheck}
      />

      {displayError && <p className="text-sm text-error-text">{displayError}</p>}

      <Button type="submit" disabled={submitting || passwordMismatch} className="w-full">
        {submitting ? '가입 신청 중...' : '가입 신청'}
      </Button>
    </form>
  );
}
