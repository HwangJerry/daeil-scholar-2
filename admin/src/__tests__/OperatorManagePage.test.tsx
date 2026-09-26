import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useAuth } from '../hooks/useAuth.ts';
import { OperatorManagePage } from '../pages/OperatorManagePage.tsx';

const OPERATORS = [
  { usrSeq: 1, usrId: 'root1', usrName: '김루트', adminRole: 'root', createdAt: '2026-09-01T00:00:00+09:00', updatedAt: '2026-09-01T00:00:00+09:00', updatedByName: '' },
  { usrSeq: 42, usrId: 'op42', usrName: '이운영', adminRole: 'operator', createdAt: '2026-09-02T00:00:00+09:00', updatedAt: '2026-09-02T00:00:00+09:00', updatedByName: '김루트' },
];

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

function renderPage(adminRole: 'root' | 'operator') {
  useAuth.setState({ user: { usrSeq: 1, usrId: 'root1', usrName: '김루트', usrStatus: 'CCC', adminRole }, isLoggedIn: true, isLoading: false });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <OperatorManagePage />
    </QueryClientProvider>,
  );
}

describe('OperatorManagePage', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('tells a non-root admin the screen is root-only without calling the API', () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    renderPage('operator');

    expect(screen.getByText('root 관리자만 이 화면을 사용할 수 있습니다.')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('locks the root’s own row and revokes another admin after confirmation', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'DELETE') return Promise.resolve(new Response(null, { status: 204 }));
      return Promise.resolve(jsonResponse({ items: OPERATORS }));
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    renderPage('root');

    const selfRow = (await screen.findByText('(본인)')).closest('tr')!;
    expect(within(selfRow).getByRole('combobox')).toBeDisabled();
    expect(within(selfRow).getByRole('button', { name: '해제' })).toBeDisabled();

    const otherRow = screen.getByText('이운영').closest('tr')!;
    await user.click(within(otherRow).getByRole('button', { name: '해제' }));
    await user.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '해제' }));

    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/admin/operators/42', expect.objectContaining({ method: 'DELETE' })));
  });

  it('sends the new role when a root changes another admin’s role', async () => {
    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      if (init?.method === 'PUT') return Promise.resolve(new Response(null, { status: 204 }));
      return Promise.resolve(jsonResponse({ items: OPERATORS }));
    });
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    renderPage('root');

    await user.selectOptions(await screen.findByLabelText('이운영 권한'), 'root');
    await user.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '지정' }));

    await vi.waitFor(() => {
      const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT');
      expect(put?.[0]).toBe('/api/admin/operators/42');
      expect(JSON.parse(String(put?.[1]?.body))).toEqual({ role: 'root' });
    });
  });
});
