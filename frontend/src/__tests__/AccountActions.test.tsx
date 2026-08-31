import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { disconnectSocial, getAccountConnections } from '../api/auth';
import { api, ApiClientError } from '../api/client';
import { AccountActions } from '../components/profile/AccountActions';

vi.mock('../api/auth', () => ({
  disconnectSocial: vi.fn(),
  getAccountConnections: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({ logout: vi.fn() }),
}));

function renderAccountActions() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <AccountActions />
    </QueryClientProvider>,
  );
}

describe('AccountActions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(api, 'get').mockResolvedValue({});
  });

  it('renders the connected password and social login methods', async () => {
    vi.mocked(getAccountConnections).mockResolvedValue({
      providers: ['KT', 'AP'],
      hasPassword: true,
    });

    renderAccountActions();

    expect(await screen.findByText('비밀번호')).toBeInTheDocument();
    expect(screen.getByText('카카오')).toBeInTheDocument();
    expect(screen.getByText('애플')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: '연결 해제' })).toHaveLength(2);
  });

  it('confirms a disconnect and refetches the connection list after success', async () => {
    vi.mocked(getAccountConnections)
      .mockResolvedValueOnce({ providers: ['KT'], hasPassword: true })
      .mockResolvedValueOnce({ providers: [], hasPassword: true });
    vi.mocked(disconnectSocial).mockResolvedValue({
      status: 'disconnected',
      connections: { providers: [], hasPassword: true },
    });

    renderAccountActions();

    fireEvent.click(await screen.findByRole('button', { name: '연결 해제' }));
    expect(screen.getByText('카카오 로그인 연결 해제')).toBeInTheDocument();

    const disconnectButtons = screen.getAllByRole('button', { name: '연결 해제' });
    fireEvent.click(disconnectButtons[disconnectButtons.length - 1]);

    await waitFor(() => expect(disconnectSocial).toHaveBeenCalledWith('kakao'));
    await waitFor(() => expect(getAccountConnections).toHaveBeenCalledTimes(2));
    expect(screen.queryByText('카카오')).not.toBeInTheDocument();
  });

  it('shows a clear message when the last login method cannot be disconnected', async () => {
    vi.mocked(getAccountConnections).mockResolvedValue({
      providers: ['AP'],
      hasPassword: false,
    });
    vi.mocked(disconnectSocial).mockRejectedValue(
      new ApiClientError(409, 'LAST_LOGIN_METHOD', '마지막 로그인 수단은 해제할 수 없습니다.'),
    );

    renderAccountActions();

    fireEvent.click(await screen.findByRole('button', { name: '연결 해제' }));
    const disconnectButtons = screen.getAllByRole('button', { name: '연결 해제' });
    fireEvent.click(disconnectButtons[disconnectButtons.length - 1]);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '마지막 로그인 수단은 해제할 수 없습니다. 먼저 비밀번호를 설정하거나 다른 로그인 수단을 추가해주세요.',
    );
    expect(disconnectSocial).toHaveBeenCalledWith('apple');
  });
});
