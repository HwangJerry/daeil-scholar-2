import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OperatorAddPanel } from '../components/operator/OperatorAddPanel.tsx';

const MEMBERS = [
  { usrSeq: 7, usrId: 'kim', usrName: '김동문', usrStatus: 'CCC', usrFn: '30', usrPhone: null, usrEmail: null, usrDept: null, regDate: null, visitDate: null },
  { usrSeq: 8, usrId: 'lee', usrName: '김탈퇴', usrStatus: 'AAA', usrFn: null, usrPhone: null, usrEmail: null, usrDept: null, regDate: null, visitDate: null },
  { usrSeq: 42, usrId: 'op', usrName: '김운영', usrStatus: 'CCC', usrFn: '12', usrPhone: null, usrEmail: null, usrDept: null, regDate: null, visitDate: null },
];

describe('OperatorAddPanel', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('lets a root grant the chosen role only to active members who are not admins yet', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: MEMBERS, total: 3 }), {
      status: 200, headers: { 'Content-Type': 'application/json' },
    }));
    vi.stubGlobal('fetch', fetchMock);
    const onGrant = vi.fn();
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <OperatorAddPanel existingSeqs={new Set([42])} disabled={false} onGrant={onGrant} />
      </QueryClientProvider>,
    );

    await user.selectOptions(screen.getByLabelText('부여할 권한'), 'root');
    await user.type(screen.getByLabelText('회원 검색'), '김');

    expect(await screen.findByText('이미 관리자')).toBeInTheDocument();
    expect(screen.getByText('지정 불가')).toBeInTheDocument();
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/admin/member?q=');

    const addButtons = screen.getAllByRole('button', { name: '추가' });
    expect(addButtons).toHaveLength(1);
    await user.click(addButtons[0]);
    expect(onGrant).toHaveBeenCalledWith(expect.objectContaining({ usrSeq: 7 }), 'root');
  });
});
