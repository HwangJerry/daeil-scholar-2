import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { AccountDeletionsPage } from '../pages/AccountDeletionsPage';
import { AccountErasureTargets } from '../components/AccountErasureTargets';
afterEach(() => {cleanup();vi.restoreAllMocks();});
function mount(node: React.ReactNode) {return render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}})}>{node}</QueryClientProvider>);}
it('only requests expedited common processing after confirmation',async()=>{
 vi.spyOn(api,'get').mockResolvedValue({items:[{requestId:7,userSeq:42,status:'pending',requestedAt:'2026-09-13T00:00:00Z',scheduledAt:'2026-09-16T00:00:00Z',targetAt:'2026-09-16T00:00:00Z',dueAt:'2026-09-26T00:00:00Z',processingMode:'automatic',targets:[]}]});
 const put=vi.spyOn(api,'put').mockResolvedValue(undefined);
 const confirm=vi.spyOn(window,'confirm').mockReturnValue(false);const user=userEvent.setup();mount(<AccountDeletionsPage/>);
 const button=await screen.findByRole('button',{name:'지금 탈퇴 처리'});await user.click(button);expect(put).not.toHaveBeenCalled();
 confirm.mockReturnValue(true);await user.click(button);
 await waitFor(()=>expect(put).toHaveBeenCalledWith('/api/admin/account-deletions/7',{action:'expedite'}));
});
it('manual handoff records responsibility without inventing completion',async()=>{
 const put=vi.spyOn(api,'put').mockResolvedValue(undefined);vi.spyOn(window,'confirm').mockReturnValue(true);const user=userEvent.setup();
 mount(<AccountErasureTargets requestId={7} targets={[{target:'backups',status:'pending',evidenceReference:'',code:'',attempts:0,lastAttemptAt:null,updatedAt:'2026-09-13T00:00:00Z'}]}/>);
 const button=screen.getByRole('button',{name:'수동 처리 인계 확인'});expect(button).toBeDisabled();
 await user.type(screen.getByLabelText(/처리 인계 또는 검증 근거/),'ticket-7');await user.click(button);
 await waitFor(()=>expect(put).toHaveBeenCalledWith('/api/admin/account-deletions/7',{action:'target',target:'backups',targetStatus:'manual',evidenceReference:'ticket-7'}));
});
