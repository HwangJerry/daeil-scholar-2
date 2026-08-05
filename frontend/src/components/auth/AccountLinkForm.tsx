// AccountLinkForm — Reauthenticates an existing member before attaching a social identity.
import { useState } from 'react';
import { useAccountLinkSubmit } from '../../hooks/useAccountLinkSubmit';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { AuthField, AuthFormError, AuthSectionText } from './AuthFormPrimitives';

interface AccountLinkFormProps {
  token: string;
}

export function AccountLinkForm({ token }: AccountLinkFormProps) {
  const { submitting, error, submit } = useAccountLinkSubmit();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    void submit({ linkToken: token, email: email.trim(), password });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <AuthSectionText>
        카카오 계정을 연결할 기존 회원 계정의 이메일과 비밀번호를 입력해주세요.
      </AuthSectionText>
      <AuthField label="이메일">
        <Input
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          autoComplete="email"
          required
        />
      </AuthField>
      <AuthField label="비밀번호">
        <Input
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
          required
        />
      </AuthField>
      {error && <AuthFormError>{error}</AuthFormError>}
      <Button type="submit" disabled={submitting} className="w-full">
        {submitting ? '연결 중...' : '계정 연결'}
      </Button>
    </form>
  );
}
