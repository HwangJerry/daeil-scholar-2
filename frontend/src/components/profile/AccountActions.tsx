// AccountActions — Account settings and member-since footer
import { useState } from 'react';
import { ChevronRight, Link2, Lock, LogOut } from 'lucide-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import { disconnectSocial, getAccountConnections } from '../../api/auth';
import { api, ApiClientError } from '../../api/client';
import { PasswordChangeModal } from './PasswordChangeModal';
import type { UserProfile } from '../../types/api';

type DialogType = 'kakao' | 'nopassword' | 'password' | null;
type SocialProvider = 'kakao' | 'apple';

const ACCOUNT_CONNECTIONS_QUERY_KEY = ['accountConnections'] as const;
const LAST_LOGIN_METHOD_MESSAGE =
  '마지막 로그인 수단은 해제할 수 없습니다. 먼저 비밀번호를 설정하거나 다른 로그인 수단을 추가해주세요.';

const SOCIAL_PROVIDER_DETAILS = [
  { connectionCode: 'KT', provider: 'kakao', label: '카카오' },
  { connectionCode: 'AP', provider: 'apple', label: '애플' },
] as const;

function ConnectedLoginMethods() {
  const queryClient = useQueryClient();
  const [providerToDisconnect, setProviderToDisconnect] = useState<SocialProvider | null>(null);
  const [disconnectError, setDisconnectError] = useState<string | null>(null);

  const connectionsQuery = useQuery({
    queryKey: ACCOUNT_CONNECTIONS_QUERY_KEY,
    queryFn: getAccountConnections,
  });

  const disconnectMutation = useMutation({
    mutationFn: (provider: SocialProvider) => disconnectSocial(provider),
    onSuccess: async () => {
      setProviderToDisconnect(null);
      setDisconnectError(null);
      await queryClient.invalidateQueries({ queryKey: ACCOUNT_CONNECTIONS_QUERY_KEY });
    },
    onError: (error: Error) => {
      const message =
        error instanceof ApiClientError && error.code === 'LAST_LOGIN_METHOD'
          ? LAST_LOGIN_METHOD_MESSAGE
          : '로그인 수단 연결 해제에 실패했습니다. 잠시 후 다시 시도해주세요.';
      setDisconnectError(message);
    },
  });

  const connectedSocialProviders = SOCIAL_PROVIDER_DETAILS.filter(({ connectionCode }) =>
    connectionsQuery.data?.providers.includes(connectionCode),
  );
  const selectedProvider = SOCIAL_PROVIDER_DETAILS.find(
    ({ provider }) => provider === providerToDisconnect,
  );

  function openDisconnectDialog(provider: SocialProvider) {
    setDisconnectError(null);
    setProviderToDisconnect(provider);
  }

  function closeDisconnectDialog() {
    if (disconnectMutation.isPending) return;
    setProviderToDisconnect(null);
    setDisconnectError(null);
  }

  return (
    <>
      <div className="border-b border-border">
        <div className="px-5 py-3.5 flex items-center gap-2.5">
          <Link2 className="h-3.5 w-3.5 text-text-secondary" />
          <span className="text-sm text-text-secondary">연결된 로그인 수단</span>
        </div>

        <div className="px-5 pb-4 pl-11 space-y-2">
          {connectionsQuery.isLoading && (
            <p className="text-xs text-text-tertiary" role="status">
              로그인 수단을 불러오는 중입니다.
            </p>
          )}

          {connectionsQuery.isError && (
            <div className="flex items-center justify-between gap-3" role="alert">
              <p className="text-xs text-error-text">로그인 수단을 불러오지 못했습니다.</p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-7 shrink-0 px-2"
                onClick={() => connectionsQuery.refetch()}
              >
                다시 시도
              </Button>
            </div>
          )}

          {connectionsQuery.data?.hasPassword && (
            <div className="flex min-h-8 items-center justify-between gap-3">
              <span className="text-sm text-text-primary">비밀번호</span>
              <span className="text-xs text-text-tertiary">연결됨</span>
            </div>
          )}

          {connectedSocialProviders.map(({ provider, label }) => (
            <div key={provider} className="flex min-h-8 items-center justify-between gap-3">
              <span className="text-sm text-text-primary">{label}</span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-7 px-2.5 text-error hover:text-error-text"
                onClick={() => openDisconnectDialog(provider)}
              >
                연결 해제
              </Button>
            </div>
          ))}

          {connectionsQuery.data &&
            !connectionsQuery.data.hasPassword &&
            connectedSocialProviders.length === 0 && (
              <p className="text-xs text-text-tertiary">연결된 로그인 수단이 없습니다.</p>
            )}
        </div>
      </div>

      {providerToDisconnect && selectedProvider && (
        <Modal onClose={closeDisconnectDialog} maxWidth="max-w-sm">
          <div className="p-6 space-y-4">
            <h2 className="text-base font-semibold text-text-primary font-serif">
              {selectedProvider.label} 로그인 연결 해제
            </h2>
            <p className="text-sm text-text-secondary">
              {selectedProvider.label} 로그인을 이 계정에서 해제하시겠습니까?
            </p>
            {disconnectError && (
              <p className="rounded-lg bg-error-light px-3 py-2 text-sm text-error-text" role="alert">
                {disconnectError}
              </p>
            )}
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                className="flex-1"
                disabled={disconnectMutation.isPending}
                onClick={closeDisconnectDialog}
              >
                취소
              </Button>
              <Button
                type="button"
                variant="destructive"
                className="flex-1"
                disabled={disconnectMutation.isPending}
                onClick={() => disconnectMutation.mutate(providerToDisconnect)}
              >
                {disconnectMutation.isPending ? '해제 중...' : '연결 해제'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </>
  );
}

export function AccountActions() {
  const { logout } = useAuth();
  const [dialog, setDialog] = useState<DialogType>(null);

  // Same queryKey as ProfileHeader → cache hit, no duplicate request
  const { data: profile, isLoading } = useQuery({
    queryKey: ['profile'],
    queryFn: () => api.get<UserProfile>('/api/profile'),
  });

  function handlePasswordClick() {
    if (profile?.hasPassword) {
      setDialog('password');
    } else if (profile?.hasSocialLogin) {
      setDialog('kakao');
    } else {
      setDialog('nopassword');
    }
  }

  return (
    <>
      <div className="px-4 space-y-3">
        {/* Account section */}
        <div className="rounded-[20px] bg-surface shadow-card border border-border overflow-hidden">
          <h3 className="px-5 py-3 text-sm font-semibold text-text-primary font-serif border-b border-border">
            계정
          </h3>
          <button
            onClick={handlePasswordClick}
            disabled={isLoading}
            className="w-full px-5 py-3.5 flex items-center justify-between border-b border-border hover:bg-background transition-colors disabled:opacity-50"
          >
            <div className="flex items-center gap-2.5">
              <Lock className="h-3.5 w-3.5 text-text-secondary" />
              <span className="text-sm text-text-secondary">비밀번호 변경</span>
            </div>
            <ChevronRight className="h-3.5 w-3.5 text-text-tertiary" />
          </button>
          <ConnectedLoginMethods />
          <Button
            variant="ghost"
            className="w-full justify-start px-5 text-error hover:text-error-text hover:bg-error-light rounded-none"
            onClick={() => logout()}
          >
            <LogOut className="mr-2.5 h-3.5 w-3.5" />
            로그아웃
          </Button>
        </div>

        {/* Member since */}
        {profile?.regDate && (
          <p className="text-center text-[11px] text-text-tertiary pt-1">
            Member since {profile.regDate}
          </p>
        )}
      </div>

      {dialog === 'kakao' && (
        <Modal onClose={() => setDialog(null)} maxWidth="max-w-sm">
          <div className="p-6 space-y-4">
            <h2 className="text-base font-semibold text-text-primary font-serif">비밀번호 변경 불가</h2>
            <p className="text-sm text-text-secondary">
              카카오 로그인 계정은 여기에서 비밀번호를 변경할 수 없습니다.
            </p>
            <Button className="w-full" onClick={() => setDialog(null)}>닫기</Button>
          </div>
        </Modal>
      )}

      {dialog === 'nopassword' && (
        <Modal onClose={() => setDialog(null)} maxWidth="max-w-sm">
          <div className="p-6 space-y-4">
            <h2 className="text-base font-semibold text-text-primary font-serif">비밀번호 미설정</h2>
            <p className="text-sm text-text-secondary">
              비밀번호가 설정되어 있지 않습니다. 이메일 비밀번호 재설정을 이용해주세요.
            </p>
            <Button className="w-full" onClick={() => setDialog(null)}>닫기</Button>
          </div>
        </Modal>
      )}

      {dialog === 'password' && (
        <PasswordChangeModal onClose={() => setDialog(null)} />
      )}
    </>
  );
}
