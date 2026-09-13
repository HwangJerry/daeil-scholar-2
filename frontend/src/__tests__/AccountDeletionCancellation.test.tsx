import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { HelmetProvider } from 'react-helmet-async';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { AccountDeletionPage } from '../pages/AccountDeletionPage';
const receiptToken = 'a'.repeat(64);
const cancelToken = 'b'.repeat(64);
const pending = {requestId: 7, status: 'pending', canCancel: true, requestedAt: '2026-09-13T00:00:00Z', dueAt: '2026-09-26T00:00:00Z'};
afterEach(() => { cleanup(); vi.restoreAllMocks(); window.history.replaceState(null, '', '/'); });
function mount() {
 window.history.replaceState(null, '', `/account-deletion#receipt=${receiptToken}&cancel=${cancelToken}`);
 render(<HelmetProvider><MemoryRouter><AccountDeletionPage /></MemoryRouter></HelmetProvider>);
}
it('requires confirmation and sends independent cancellation capability only to cancel endpoint', async () => {
 const post = vi.spyOn(api, 'post').mockResolvedValueOnce(pending).mockResolvedValueOnce({...pending, status: 'cancelled', canCancel: false});
 const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
 const user = userEvent.setup(); mount();
 const button = await screen.findByRole('button', {name: '탈퇴 신청 취소'});
 expect(window.location.hash).toBe('');
 expect(post).toHaveBeenNthCalledWith(1, '/api/account-deletion/receipt', {receiptToken});
 await user.click(button); expect(post).toHaveBeenCalledTimes(1);
 confirm.mockReturnValue(true); await user.click(button);
 await screen.findByText('탈퇴 신청이 취소되었습니다');
 expect(post).toHaveBeenNthCalledWith(2, '/api/account-deletion/cancel', {receiptToken, cancelToken});
 expect(screen.queryByRole('button', {name:'탈퇴 신청 취소'})).not.toBeInTheDocument();
});
it('refreshes execution state after rejection without showing cancellation success', async () => {
 const post = vi.spyOn(api, 'post').mockResolvedValueOnce(pending).mockRejectedValueOnce(new Error('처리가 시작되었습니다')).mockResolvedValueOnce({...pending, status:'processing', canCancel:false});
 vi.spyOn(window, 'confirm').mockReturnValue(true); const user = userEvent.setup(); mount();
 await user.click(await screen.findByRole('button', {name:'탈퇴 신청 취소'}));
 await waitFor(() => expect(post).toHaveBeenCalledTimes(3));
 await screen.findByText('계정 삭제 처리 중입니다');
 expect(screen.queryByText('탈퇴 신청이 취소되었습니다')).not.toBeInTheDocument();
});
