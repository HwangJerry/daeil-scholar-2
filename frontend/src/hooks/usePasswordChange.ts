// usePasswordChange — API mutation for authenticated password change.
import { useState } from 'react';
import { api, ApiClientError } from '../api/client';

interface PasswordChangeState {
  loading: boolean;
  error: string | null;
  success: boolean;
}

interface ChangePasswordArgs {
  currentPassword: string;
  newPassword: string;
}

export function usePasswordChange() {
  const [state, setState] = useState<PasswordChangeState>({
    loading: false,
    error: null,
    success: false,
  });

  async function changePassword({ currentPassword, newPassword }: ChangePasswordArgs) {
    setState({ loading: true, error: null, success: false });
    try {
      await api.post('/api/profile/password', { currentPassword, newPassword });
      setState({ loading: false, error: null, success: true });
    } catch (err) {
      const message =
        err instanceof ApiClientError && err.code === 'WRONG_PASSWORD'
          ? '현재 비밀번호가 올바르지 않습니다.'
          : '비밀번호 변경에 실패했습니다. 다시 시도해주세요.';
      setState({ loading: false, error: message, success: false });
    }
  }

  return { ...state, changePassword };
}
