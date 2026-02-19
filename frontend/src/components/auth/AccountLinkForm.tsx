// AccountLinkForm — Unified form for linking Kakao to existing member or registering as new.
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, ApiClientError } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import type { KakaoLinkRequest } from '../../types/api';

interface AccountLinkFormProps {
  token: string;
}

export function AccountLinkForm({ token }: AccountLinkFormProps) {
  const navigate = useNavigate();
  const fetchUser = useAuth((s) => s.fetchUser);

  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [fn, setFn] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    const body: KakaoLinkRequest = { token, name, phone, fn };

    try {
      await api.post('/api/auth/kakao/link', body);
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
  };

  const inputClass =
    'w-full rounded-2xl border border-border px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-royal-indigo/20';

  return (
    <div className="mx-auto max-w-md">
      <div className="mb-6 rounded-2xl bg-soft-sky p-4 text-sm text-royal-indigo">
        <p>기존 회원이시면 자동으로 계정이 통합됩니다.</p>
        <p className="mt-1">신규 회원이시면 새로 가입됩니다.</p>
      </div>

      <div className="rounded-2xl bg-white p-6 shadow-sm border border-border-subtle">
        <h2 className="mb-1 text-lg font-semibold text-dark-slate">회원 정보 입력</h2>
        <p className="mb-4 text-sm text-text-muted">
          이름과 전화번호를 입력해주세요. 기수는 선택사항입니다.
        </p>

        <form onSubmit={handleSubmit}>
          <div className="mb-3">
            <label className="mb-1 block text-sm font-medium text-text-muted">이름</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className={inputClass}
              placeholder="실명을 입력하세요"
            />
          </div>

          <div className="mb-3">
            <label className="mb-1 block text-sm font-medium text-text-muted">전화번호</label>
            <input
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              required
              className={inputClass}
              placeholder="010-0000-0000"
            />
          </div>

          <div className="mb-4">
            <label className="mb-1 block text-sm font-medium text-text-muted">
              기수 <span className="text-text-placeholder">(선택)</span>
            </label>
            <input
              type="text"
              value={fn}
              onChange={(e) => setFn(e.target.value)}
              className={inputClass}
              placeholder="예: 15"
            />
          </div>

          {error && <p className="mb-3 text-sm text-error-text">{error}</p>}

          <Button type="submit" disabled={submitting} className="w-full">
            {submitting ? '처리 중...' : '확인'}
          </Button>
        </form>
      </div>

      <p className="mt-4 text-center text-xs text-text-placeholder">
        문제가 있으시면 관리자에게 문의하세요.
      </p>
    </div>
  );
}
