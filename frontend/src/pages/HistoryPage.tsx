// HistoryPage — Year-grouped timeline of foundation milestones
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';

interface HistoryItem {
  heSeq: number;
  eventDate: string; // "YYYY-MM-DD"
  text: string;
  sortOrder: number;
}

interface HistoryYearGroup {
  year: number;
  items: HistoryItem[];
}

function displayDate(date: string) {
  const parts = date.split('-');
  if (parts.length < 3) return date;
  return `${parts[1]}.${parts[2]}`;
}

export function HistoryPage() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['history'],
    queryFn: () => api.get<HistoryYearGroup[]>('/api/history'),
    staleTime: 5 * 60_000,
  });

  return (
    <InfoPageShell
      title="연혁"
      subtitle="장학회가 걸어온 길"
      canonicalPath="/history"
    >
      <Card variant="default" padding="lg">
        {isLoading ? (
          <div className="space-y-8">
            {[1, 2, 3].map((n) => (
              <div key={n} className="h-24 skeleton-shimmer rounded-lg" />
            ))}
          </div>
        ) : isError ? (
          <p className="text-center text-sm text-error-text py-8">연혁을 불러올 수 없습니다.</p>
        ) : (
          <ol className="space-y-8">
            {(data ?? []).map((yearEntry) => (
              <li key={yearEntry.year} className="md:grid md:grid-cols-[80px_1fr] md:gap-6">
                <h2 className="font-serif text-2xl font-semibold text-text-primary mb-3 md:mb-0">
                  {yearEntry.year}
                </h2>
                <ul className="space-y-3 border-l border-border pl-4">
                  {yearEntry.items.map((item) => (
                    <li key={item.heSeq} className="relative">
                      <span
                        aria-hidden
                        className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-primary"
                      />
                      <p className="text-body-sm text-text-tertiary tabular-nums">{displayDate(item.eventDate)}</p>
                      <p className="text-body-md text-text-secondary leading-relaxed">{item.text}</p>
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        )}
      </Card>
    </InfoPageShell>
  );
}
