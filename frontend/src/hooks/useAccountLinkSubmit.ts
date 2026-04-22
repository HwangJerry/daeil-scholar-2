// useAccountLinkSubmit — Shared submit handler for AccountLinkNewForm and AccountLinkMergeForm.
import { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiClientError } from '../api/client';
import { useAuth } from './useAuth';
import type { SocialLinkRequest } from '../types/api';

interface Result {
  submitting: boolean;
  error: string;
  submit: (body: SocialLinkRequest) => Promise<void>;
  setError: (msg: string) => void;
}

export function useAccountLinkSubmit(): Result {
  const navigate = useNavigate();
  const fetchUser = useAuth((s) => s.fetchUser);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const submit = useCallback(
    async (body: SocialLinkRequest) => {
      setError('');
      setSubmitting(true);
      try {
        await api.post('/api/auth/social/link', body);
        await fetchUser();
        navigate('/', { replace: true });
      } catch (err) {
        if (err instanceof ApiClientError) {
          setError(err.message);
        } else {
          setError('처리에 실패했습니다. 다시 시도해주세요.');
        }
      } finally {
        setSubmitting(false);
      }
    },
    [navigate, fetchUser],
  );

  return { submitting, error, submit, setError };
}
