// ForgotPasswordPage — Email form to request a password reset link
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { requestPasswordReset } from '../api/passwordReset';
import { ApiClientError } from '../api/client';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { AlertDialog } from '../components/ui/AlertDialog';
import { useBlockBack } from '../hooks/useBlockBack';
import { AuthField, AuthFormError } from '../components/auth/AuthFormPrimitives';
import { AuthFooterLink, AuthScreen } from '../components/auth/AuthScreen';

export function ForgotPasswordPage() {
  useBlockBack();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      await requestPasswordReset(email);
      setSuccess(true);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('요청 처리에 실패했습니다. 다시 시도해주세요.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthScreen title="비밀번호 찾기" description="가입 시 등록한 이메일을 입력해주세요.">
      <AlertDialog
        open={success}
        title="이메일 발송 완료"
        message="입력하신 이메일로 비밀번호 재설정 링크를 보냈습니다. 메일함을 확인해주세요."
        onConfirm={() => navigate('/login/legacy', { replace: true })}
      />

      {!success && (
        <>
          <form onSubmit={handleSubmit} className="space-y-3">
            <AuthField label="이메일">
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                    e.preventDefault();
                    e.currentTarget.form?.requestSubmit();
                  }
                }}
                required
                placeholder="이메일을 입력하세요"
                autoComplete="email"
              />
            </AuthField>

            {error && <AuthFormError>{error}</AuthFormError>}

            <Button type="submit" disabled={submitting} className="w-full">
              {submitting ? '요청 중...' : '비밀번호 재설정 링크 받기'}
            </Button>
          </form>

          <div className="mt-4 text-center">
            <AuthFooterLink to="/login/legacy">로그인으로 돌아가기</AuthFooterLink>
          </div>
        </>
      )}
    </AuthScreen>
  );
}
