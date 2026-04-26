// BusinessPage — Overview of foundation scholarship/support programs
import { CheckCircle2 } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import {
  BUSINESS_HEADLINE,
  BUSINESS_SUBHEAD,
  BUSINESS_ITEMS,
} from '../constants/aboutContent';

export function BusinessPage() {
  return (
    <InfoPageShell
      title="장학사업"
      subtitle="모교 후배들의 가능성을 여는 장학사업 운영"
      canonicalPath="/business"
    >
      <Card variant="default" padding="lg" className="space-y-2 text-center">
        <p className="font-serif text-body-md font-semibold text-text-primary">
          {BUSINESS_HEADLINE}
        </p>
        <p className="text-body-sm text-text-tertiary leading-relaxed">
          {BUSINESS_SUBHEAD}
        </p>
      </Card>

      <ol className="space-y-4">
        {BUSINESS_ITEMS.map((item, index) => (
          <li key={item.title}>
            <Card variant="default" padding="lg" className="space-y-3">
              <div className="flex items-baseline gap-3">
                <span className="font-serif text-body-sm font-semibold text-text-tertiary tabular-nums">
                  {String(index + 1).padStart(2, '0')}
                </span>
                <h2 className="font-serif text-body-md font-semibold text-text-primary leading-snug">
                  {item.title}
                </h2>
              </div>
              <ul className="space-y-2 text-body-sm text-text-secondary leading-relaxed">
                {item.bullets.map((bullet) => (
                  <li key={bullet} className="flex gap-2">
                    <CheckCircle2
                      size={14}
                      aria-hidden
                      className="mt-1 shrink-0 text-primary"
                    />
                    <span>{bullet}</span>
                  </li>
                ))}
              </ul>
            </Card>
          </li>
        ))}
      </ol>
    </InfoPageShell>
  );
}
