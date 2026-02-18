// SnapshotHistoryTable — read-only table of donation snapshot history for the last 30 days
import { useDonationHistory } from '../../hooks/useDonationHistory.ts';
import { formatAmount } from '../../lib/formatAmount.ts';

export function SnapshotHistoryTable() {
  const { data: history } = useDonationHistory();

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-6 shadow-sm">
      <h3 className="mb-4 font-semibold text-dark-slate">스냅샷 이력 (최근 30일)</h3>
      {history?.length ? (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100 text-left text-cool-gray">
                <th className="px-3 py-2 font-medium">일자</th>
                <th className="px-3 py-2 font-medium text-right">누적 총액</th>
                <th className="px-3 py-2 font-medium text-right">수동 조정</th>
                <th className="px-3 py-2 font-medium text-right">기부자 수</th>
                <th className="px-3 py-2 font-medium text-right">목표</th>
              </tr>
            </thead>
            <tbody>
              {history.map((s) => (
                <tr key={s.dsDate} className="border-b border-slate-50">
                  <td className="px-3 py-2 text-dark-slate">{s.dsDate}</td>
                  <td className="px-3 py-2 text-right text-dark-slate">₩{formatAmount(s.dsTotal)}</td>
                  <td className="px-3 py-2 text-right text-cool-gray">₩{formatAmount(s.dsManualAdj)}</td>
                  <td className="px-3 py-2 text-right text-cool-gray">{s.dsDonorCnt}명</td>
                  <td className="px-3 py-2 text-right text-cool-gray">₩{formatAmount(s.dsGoal)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="text-sm text-cool-gray">이력이 없습니다.</p>
      )}
    </div>
  );
}
