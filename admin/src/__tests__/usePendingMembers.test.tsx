import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { usePendingMembers, type PendingVerification } from '../hooks/usePendingMembers';

const member: PendingVerification = { userSeq: 42, userName: '테스트', status: 'pending', graduationYear: 2000, cohort: '14', department: '영어', submittedAt: null, updatedAt: '2026-09-08T11:00:00+09:00' };
afterEach(() => vi.restoreAllMocks());
function setup() {
 const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
 const wrapper = ({children}: PropsWithChildren) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
 return { client, ...renderHook(usePendingMembers, { wrapper }) };
}
it('includes reapproval requests and accepts empty server lists', async () => {
 vi.spyOn(api, 'get').mockImplementation(async (url) => ({ items: url.endsWith('=pending') ? null : [{...member,status:'reapproval_pending'}] }) as never);
 const {result}=setup();
 await waitFor(()=>expect(result.current.data?.items).toHaveLength(1));
 expect(result.current.data?.items[0].status).toBe('reapproval_pending');
});
it('approves the displayed version through the canonical API', async () => {
 vi.spyOn(api,'get').mockResolvedValue({items:[]});
 const post=vi.spyOn(api,'post').mockResolvedValue(undefined);
 const legacy=vi.spyOn(api,'put');
 const {result}=setup();
 act(()=>result.current.approve(member));
 await waitFor(()=>expect(post).toHaveBeenCalledWith('/api/admin/alumni-verifications/42/approve',{expectedUpdatedAt:member.updatedAt}));
 expect(legacy).not.toHaveBeenCalled();
});
it('sends a reason and refreshes stale decisions instead of retrying them', async () => {
 vi.spyOn(api,'get').mockResolvedValue({items:[]});
 const post=vi.spyOn(api,'post').mockRejectedValue(new Error('stale'));
 const {result,client}=setup(); const refresh=vi.spyOn(client,'invalidateQueries');
 act(()=>result.current.reject({member,reason:' 정보 확인 필요 '}));
 await waitFor(()=>expect(refresh).toHaveBeenCalled());
 expect(post).toHaveBeenCalledExactlyOnceWith('/api/admin/alumni-verifications/42/reject',{expectedUpdatedAt:member.updatedAt,reason:'정보 확인 필요'});
});
