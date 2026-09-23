// donationBalanceForm — validates the optional account balance and its calendar date.
type DonationBalanceResult =
  | { ok: true; balanceAmount: number | null; balanceAsOf: string | null }
  | { ok: false; message: string };

export function parseDonationBalance(amount: string, asOf: string): DonationBalanceResult {
  const balanceAmount = amount === '' ? null : Number(amount);
  if (balanceAmount !== null && (!/^\d+$/.test(amount) || !Number.isSafeInteger(balanceAmount))) {
    return { ok: false, message: '계좌 잔액은 0 이상의 정수여야 합니다.' };
  }
  if (balanceAmount !== null && asOf === '') {
    return { ok: false, message: '계좌 잔액을 입력하면 잔액 기준일이 필요합니다.' };
  }
  if (asOf !== '') {
    const date = new Date(`${asOf}T00:00:00Z`);
    const isValidDate = /^\d{4}-\d{2}-\d{2}$/.test(asOf)
      && !Number.isNaN(date.getTime())
      && date.getUTCFullYear() >= 1000
      && date.toISOString().slice(0, 10) === asOf;
    if (!isValidDate) return { ok: false, message: '잔액 기준일은 올바른 YYYY-MM-DD 날짜여야 합니다.' };
  }
  return { ok: true, balanceAmount, balanceAsOf: balanceAmount === null ? null : asOf };
}
