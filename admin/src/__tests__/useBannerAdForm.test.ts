import { describe, expect, it } from 'vitest';
import { utcToDatetimeLocal } from '../hooks/useBannerAdForm.ts';

describe('utcToDatetimeLocal', () => {
  it('converts an RFC3339 UTC timestamp to KST without appending a second Z', () => {
    expect(utcToDatetimeLocal('2026-01-01T00:00:00Z')).toBe('2026-01-01T09:00');
  });

  it('respects an existing numeric time-zone offset', () => {
    expect(utcToDatetimeLocal('2026-01-01T09:00:00+09:00')).toBe('2026-01-01T09:00');
  });

  it('continues to treat legacy time-zone-less values as UTC', () => {
    expect(utcToDatetimeLocal('2026-01-01 00:00:00')).toBe('2026-01-01T09:00');
  });
});
