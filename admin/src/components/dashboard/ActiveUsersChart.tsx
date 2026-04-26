// ActiveUsersChart — 30-day DAU/MAU trend line chart for the admin dashboard
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useActiveUsers } from '../../hooks/useActiveUsers.ts';

function formatDate(iso: string): string {
  // "2026-04-22" → "04-22"
  return iso.length >= 10 ? iso.slice(5) : iso;
}

export function ActiveUsersChart() {
  const { data, isLoading } = useActiveUsers();
  const points = data?.points ?? [];

  return (
    <div className="rounded-2xl border border-border-light bg-white p-5 shadow-sm">
      <div className="mb-4 flex items-baseline justify-between">
        <h3 className="font-semibold text-dark-slate">활성 사용자 추이 (최근 30일)</h3>
        <span className="text-xs text-cool-gray">DAU · MAU</span>
      </div>
      {isLoading ? (
        <div className="flex h-64 items-center justify-center text-sm text-cool-gray">
          불러오는 중...
        </div>
      ) : (
        <div className="h-64 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={points} margin={{ top: 8, right: 16, bottom: 0, left: -8 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
              <XAxis
                dataKey="date"
                tickFormatter={formatDate}
                tick={{ fontSize: 11, fill: '#6B7280' }}
              />
              <YAxis
                allowDecimals={false}
                tick={{ fontSize: 11, fill: '#6B7280' }}
                width={40}
              />
              <Tooltip
                labelFormatter={(v) => String(v)}
                contentStyle={{ fontSize: 12, borderRadius: 8, borderColor: '#E5E7EB' }}
              />
              <Legend wrapperStyle={{ fontSize: 12 }} />
              <Line
                type="monotone"
                dataKey="dau"
                name="DAU"
                stroke="#4F46E5"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="mau"
                name="MAU (30일)"
                stroke="#10B981"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
