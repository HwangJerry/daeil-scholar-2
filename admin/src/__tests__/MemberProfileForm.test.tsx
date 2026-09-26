import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemberProfileForm } from '../components/member/MemberProfileForm.tsx';
import type { AdminMemberDetail } from '../types/api.ts';

const MEMBER: AdminMemberDetail = {
  usrSeq: 42, usrId: 'hong', usrName: '홍길동', usrStatus: 'CCC', usrFn: '30', usrDept: '영어',
  usrPhone: '01011112222', usrEmail: 'old@example.com', usrNick: null, usrPhoto: null, regDate: null, visitCnt: 0, visitDate: null,
};

function renderForm(onOpenChange = vi.fn()) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemberProfileForm open member={MEMBER} onOpenChange={onOpenChange} />
    </QueryClientProvider>,
  );
  return onOpenChange;
}

describe('MemberProfileForm', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('sends the corrected profile and closes on success', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    const onOpenChange = renderForm();

    const phone = screen.getByLabelText(/휴대폰/);
    await user.clear(phone);
    await user.type(phone, '010-3333-4444');
    await user.click(screen.getByRole('button', { name: '저장' }));

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe('/api/admin/member/42/profile');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(String(init.body))).toEqual({
      usrName: '홍길동', usrPhone: '010-3333-4444', usrEmail: 'old@example.com', usrFn: '30', usrDept: '영어',
    });
    await vi.waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it('blocks an invalid cohort before sending', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    renderForm();

    const cohort = screen.getByLabelText('기수');
    await user.clear(cohort);
    await user.type(cohort, '30기');
    await user.click(screen.getByRole('button', { name: '저장' }));

    expect(fetchMock).not.toHaveBeenCalled();
    expect(screen.getByText('기수는 1~99 사이 숫자로 입력해 주세요.')).toBeInTheDocument();
  });

  it('shows the server message when the phone belongs to another member', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ code: 'PHONE_TAKEN', message: '다른 회원이 사용 중인 휴대폰 번호입니다' }),
      { status: 409, headers: { 'Content-Type': 'application/json' } },
    )));
    const user = userEvent.setup();
    const onOpenChange = renderForm();

    await user.click(screen.getByRole('button', { name: '저장' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('다른 회원이 사용 중인 휴대폰 번호입니다');
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });
});
