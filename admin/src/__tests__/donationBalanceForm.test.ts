// donationBalanceForm tests — calendar validity and optional balance semantics.
import { describe, expect, it } from 'vitest';
import { parseDonationBalance } from '../components/donation/donationBalanceForm.ts';

describe('donation balance calendar validation', () => {
  it.each(['2026-02-29', '2026-04-31', '2026-9-23', '2026-09-23T00:00:00Z', '0000-01-01'])(
    'rejects %s', (date) => {
      expect(parseDonationBalance('1000', date).ok).toBe(false);
    },
  );

  it('accepts a leap day and preserves zero', () => {
    expect(parseDonationBalance('0', '2024-02-29')).toEqual({
      ok: true, balanceAmount: 0, balanceAsOf: '2024-02-29',
    });
  });

  it('clears the date with an empty balance', () => {
    expect(parseDonationBalance('', '2026-09-23')).toEqual({
      ok: true, balanceAmount: null, balanceAsOf: null,
    });
  });
});
