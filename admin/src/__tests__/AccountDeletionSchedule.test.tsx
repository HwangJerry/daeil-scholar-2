import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { AccountDeletionsPage } from '../pages/AccountDeletionsPage';
import { AccountErasureTargets } from '../components/AccountErasureTargets';
afterEach(() => {cleanup();vi.restoreAllMocks();});
function mount(node: React.ReactNode) {return render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}})}>{node}</QueryClientProvider>);}
it('manual handoff records responsibility without inventing completion',async()=>{
 const put=vi.spyOn(api,'put').mockResolvedValue(undefined);vi.spyOn(window,'confirm').mockReturnValue(true);const user=userEvent.setup();
 mount(<AccountErasureTargets requestId={7} targets={[{target:'backups',status:'pending',evidenceReference:'',code:'',attempts:0,lastAttemptAt:null,updatedAt:'2026-09-13T00:00:00Z'}]}/>);
 const button=screen.getByRole('button',{name:'수동 처리 인계 확인'});expect(button).toBeDisabled();
 await user.type(screen.getByLabelText(/처리 인계 또는 검증 근거/),'ticket-7');await user.click(button);
 await waitFor(()=>expect(put).toHaveBeenCalledWith('/api/admin/account-deletions/7',{action:'target',target:'backups',targetStatus:'manual',evidenceReference:'ticket-7'}));
});
it('requires identity verification evidence for administrative cancellation',async()=>{
 vi.spyOn(api,'get').mockResolvedValue({items:[{requestId:7,userSeq:42,status:'pending',canCancel:true,requestedAt:'2026-09-13T00:00:00Z',targetAt:'2026-09-16T00:00:00Z',dueAt:'2026-09-26T00:00:00Z',processingMode:'manual',targets:[]}]});
 const put=vi.spyOn(api,'put').mockResolvedValue(undefined);vi.spyOn(window,'confirm').mockReturnValue(true);const user=userEvent.setup();mount(<AccountDeletionsPage/>);
 const button=await screen.findByRole('button',{name:'본인 확인 후 신청 취소'});expect(button).toBeDisabled();
 await user.type(screen.getByLabelText(/신청 취소 본인 확인 근거/),'support-ticket-7');await user.click(button);
 await waitFor(()=>expect(put).toHaveBeenCalledWith('/api/admin/account-deletions/7',{action:'cancel_verified',evidenceReference:'support-ticket-7'}));
});
it('warns when the provider unlink has stalled and offers the retry',async()=>{
 vi.spyOn(api,'get').mockResolvedValue({items:[{requestId:7,userSeq:42,status:'processing',requestedAt:'2026-09-13T00:00:00Z',scheduledAt:'2026-09-16T00:00:00Z',targetAt:'2026-09-16T00:00:00Z',dueAt:'2026-09-26T00:00:00Z',processingMode:'automatic',autoStage:'queued',autoCode:'',socialUnlinkStalled:true,targets:[]}]});
 mount(<AccountDeletionsPage/>);
 expect(await screen.findByText(/연결 해제가 15분 넘게 끝나지 않았거나 실패했습니다/)).toBeInTheDocument();
 expect(screen.getByRole('button',{name:'소셜 설정 확인 후 재시도'})).toBeInTheDocument();
});
