// formatAmount — Formats monetary amounts with Korean units (만/억)
export function formatAmount(amount: number): string {
  if (amount >= 100_000_000) {
    const eok = Math.floor(amount / 100_000_000);
    const man = Math.floor((amount % 100_000_000) / 10_000);
    return man > 0 ? `${eok}억 ${man.toLocaleString()}만` : `${eok}억`;
  }
  if (amount >= 10_000) {
    return `${Math.floor(amount / 10_000).toLocaleString()}만`;
  }
  return amount.toLocaleString();
}
