import { describe, expect, it } from 'vitest';
import { buildMemberListParams } from '../hooks/useMemberList.ts';

describe('buildMemberListParams', () => {
  const EMPTY = { fn: '', dept: '', regFrom: '', regTo: '' };

  it('sends cohort, department and join-date range', () => {
    const params = buildMemberListParams(2, 20, '홍', 'CCC', { fn: '30', dept: '영어', regFrom: '2026-01-01', regTo: '2026-01-31' });
    expect(Object.fromEntries(params)).toEqual({
      page: '2', size: '20', q: '홍', status: 'CCC', fn: '30', dept: '영어', regFrom: '2026-01-01', regTo: '2026-01-31',
    });
  });

  it('drops an incomplete cohort and a reversed date range instead of sending invalid filters', () => {
    const params = buildMemberListParams(1, 20, '', '', { ...EMPTY, fn: '3기', regFrom: '2026-02-01', regTo: '2026-01-01' });
    expect(Object.fromEntries(params)).toEqual({ page: '1', size: '20' });
  });
});
