// LegacyLoginPage — ID/password login form for legacy accounts.
import { useState } from 'react';
import { Navigate, Link, useNavigate, useSearchParams } from 'react-router-dom';
import { legacyLogin } from '../api/auth';
import { useAuth } from '../hooks/useAuth';
import { ApiClientError } from '../api/client';
import { Loader2 } from 'lucide-react';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { AlertDialog } from '../components/ui/AlertDialog';

export function LegacyLoginPage() {
  const { isLoggedIn, isLoading, fetchUser } = useAuth();
  const navigate = useNavigate();

  const [usrId, setUsrId] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const registeredSuccess = searchParams.get('registered') === 'true';

  const handleRegisteredDialogConfirm = () => {
    setSearchParams({}, { replace: true });
  };

  if (isLoading) return null;
  if (isLoggedIn) return <Navigate to="/" replace />;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      await legacyLogin({ usrId, password });
      await fetchUser();
      navigate('/', { replace: true });
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.code === 'PENDING_APPROVAL') {
          setError('가입 신청이 접수된 계정입니다. 관리자 승인 후 로그인 가능합니다.');
        } else {
          setError(err.message);
        }
      } else {
        setError('로그인에 실패했습니다. 다시 시도해주세요.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-[60vh] items-center justify-center animate-fade-in-up">
      <div className="w-full max-w-sm px-4">
        <h1 className="mb-6 text-center text-xl font-bold text-text-primary">기존 계정 로그인</h1>

        <AlertDialog
          open={registeredSuccess}
          title="가입 신청 완료"
          message="가입 신청이 완료되었습니다. 승인 후 로그인하세요."
          onConfirm={handleRegisteredDialogConfirm}
        />

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">아이디</label>
            <Input
              type="text"
              value={usrId}
              onChange={(e) => setUsrId(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  e.currentTarget.form?.requestSubmit();
                }
              }}
              required
              placeholder="아이디를 입력하세요"
              autoComplete="username"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">비밀번호</label>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  e.currentTarget.form?.requestSubmit();
                }
              }}
              required
              placeholder="비밀번호를 입력하세요"
              autoComplete="current-password"
            />
          </div>

          {error && <p className="text-sm text-error-text">{error}</p>}

          <Button type="submit" disabled={submitting} className="w-full">
            {submitting ? (
              <>
                <Loader2 size={14} className="animate-spin" />
                로그인 중...
              </>
            ) : '로그인'}
          </Button>
        </form>

        <div className="mt-6 text-center space-y-3">
          <div>
            <Link
              to="/forgot-password"
              className="text-xs text-primary hover:text-primary/80 transition-colors"
            >
              비밀번호를 잊으셨나요?
            </Link>
          </div>
          <div className="flex items-center justify-center gap-3">
            <Link
              to="/login"
              className="text-xs text-text-muted hover:text-text-secondary transition-colors"
            >
              카카오로 계속하기
            </Link>
            <span className="text-xs text-text-muted">|</span>
            <Link
              to="/register"
              className="text-xs text-text-muted hover:text-text-secondary transition-colors"
            >
              아이디로 회원가입
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
