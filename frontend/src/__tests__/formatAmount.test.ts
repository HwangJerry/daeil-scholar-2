import { describe, expect, it } from 'vitest';
import { formatAmount } from '../utils/formatAmount';

describe('formatAmount', () => {
  it.each([
    [0, '0'],
    [9_999, '9,999'],
    [10_000, '1만'],
    [123_456_789, '1억 2,345만'],
    [9_876_543_210, '98억 7,654만'],
  ])('formats %i using stable Korean monetary units', (amount, expected) => {
    expect(formatAmount(amount)).toBe(expected);
  });
});
