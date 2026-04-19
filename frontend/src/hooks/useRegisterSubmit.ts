// useRegisterSubmit.ts — API integration and navigation logic for the registration form.
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { register } from '../api/auth';
import { ApiClientError } from '../api/client';
import type { RegisterRequest } from '../api/auth';

interface SubmitState {
  error: string;
  submitting: boolean;
}

/** Handles POST /api/auth/register and redirects on success.
 *  Returns submit function plus error/submitting state for the form to render. */
export function useRegisterSubmit() {
  const navigate = useNavigate();
  const [state, setState] = useState<SubmitState>({ error: '', submitting: false });

  const submit = async (req: RegisterRequest) => {
    setState({ error: '', submitting: true });
    try {
      await register(req);
      navigate('/login/legacy?registered=true', { replace: true });
    } catch (err) {
      const message = (() => {
        if (!(err instanceof ApiClientError)) return '회원가입에 실패했습니다. 다시 시도해주세요.';
        if (err.code === 'ID_TAKEN') return '이미 사용 중인 아이디입니다.';
        if (err.code === 'PHONE_TAKEN') return '이미 등록된 전화번호입니다.';
        if (err.code === 'EMAIL_TAKEN') return '이미 등록된 이메일입니다.';
        return err.message;
      })();
      setState({ error: message, submitting: false });
    }
  };

  return { submit, error: state.error, submitting: state.submitting };
}
