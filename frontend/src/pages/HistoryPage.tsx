// HistoryPage — Year-grouped timeline of foundation milestones
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import { HISTORY_ENTRIES } from '../constants/aboutContent';

export function HistoryPage() {
  return (
    <InfoPageShell
      title="연혁"
      subtitle="장학회가 걸어온 길"
      canonicalPath="/history"
    >
      <Card variant="default" padding="lg">
        <ol className="space-y-8">
          {HISTORY_ENTRIES.map((yearEntry) => (
            <li key={yearEntry.year} className="md:grid md:grid-cols-[80px_1fr] md:gap-6">
              <h2 className="font-serif text-2xl font-semibold text-text-primary mb-3 md:mb-0">
                {yearEntry.year}
              </h2>
              <ul className="space-y-3 border-l border-border pl-4">
                {yearEntry.items.map((item) => (
                  <li key={`${yearEntry.year}-${item.date}-${item.text.slice(0, 16)}`} className="relative">
                    <span
                      aria-hidden
                      className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-primary"
                    />
                    <p className="text-body-sm text-text-tertiary tabular-nums">{item.date}</p>
                    <p className="text-body-md text-text-secondary leading-relaxed">{item.text}</p>
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ol>
      </Card>
    </InfoPageShell>
  );
}
