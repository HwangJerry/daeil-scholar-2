// PasswordChangeModal — Inline form for authenticated users to change their password.
import { useState, useRef } from 'react';
import { Modal } from '../ui/Modal';
import { Button } from '../ui/Button';
import { usePasswordChange } from '../../hooks/usePasswordChange';
import { usePasswordValidation, checkPasswordStrength } from '../../hooks/usePasswordValidation';

const DEBOUNCE_MS = 500;

interface PasswordChangeModalProps {
  onClose: () => void;
}

export function PasswordChangeModal({ onClose }: PasswordChangeModalProps) {
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState('');

  const pwDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { validate } = usePasswordValidation();
  const { loading, error: apiError, success, changePassword } = usePasswordChange();

  const validatePw = (value: string) => {
    setPasswordError(checkPasswordStrength(value) ?? '');
  };

  const handleNewPasswordChange = (value: string) => {
    setNewPassword(value);
    if (pwDebounceRef.current) clearTimeout(pwDebounceRef.current);
    pwDebounceRef.current = setTimeout(() => validatePw(value), DEBOUNCE_MS);
  };

  const handleNewPasswordBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    if (pwDebounceRef.current) clearTimeout(pwDebounceRef.current);
    validatePw(e.target.value);
  };

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setValidationError(null);

    const result = validate({ newPassword, confirmPassword });
    if (!result.ok) {
      setValidationError(result.reason);
      return;
    }

    await changePassword({ currentPassword, newPassword });
  }

  const displayError = validationError ?? apiError;

  return (
    <Modal onClose={onClose} maxWidth="max-w-sm">
      <div className="p-6">
        <h2 className="text-base font-semibold text-text-primary font-serif mb-4">비밀번호 변경</h2>

        {success ? (
          <div className="space-y-4">
            <p className="text-sm text-text-secondary">비밀번호가 성공적으로 변경되었습니다.</p>
            <Button className="w-full" onClick={onClose}>닫기</Button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label className="block text-xs text-text-secondary mb-1">현재 비밀번호</label>
              <input
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-xs text-text-secondary mb-1">새 비밀번호</label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => handleNewPasswordChange(e.target.value)}
                onBlur={handleNewPasswordBlur}
                required
                placeholder="8자 이상, 영문+숫자+특수문자 포함"
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
              {passwordError && (
                <p className="mt-1 text-xs text-error">{passwordError}</p>
              )}
            </div>
            <div>
              <label className="block text-xs text-text-secondary mb-1">새 비밀번호 확인</label>
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>

            {displayError && <p className="text-xs text-error">{displayError}</p>}

            <div className="flex gap-2 pt-1">
              <Button type="button" variant="ghost" className="flex-1" onClick={onClose} disabled={loading}>
                취소
              </Button>
              <Button type="submit" className="flex-1" disabled={loading}>
                {loading ? '변경 중...' : '변경'}
              </Button>
            </div>
          </form>
        )}
      </div>
    </Modal>
  );
}
