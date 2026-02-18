// formatAmount — converts a numeric amount to a Korean-style abbreviated string
export function formatAmount(amount: number): string {
  if (amount >= 100_000_000) {
    const eok = Math.floor(amount / 100_000_000);
    const remainder = Math.floor((amount % 100_000_000) / 10_000);
    return remainder > 0 ? `${eok}억 ${remainder.toLocaleString()}만` : `${eok}억`;
  }
  if (amount >= 10_000) {
    return `${Math.floor(amount / 10_000).toLocaleString()}만`;
  }
  return amount.toLocaleString();
}
