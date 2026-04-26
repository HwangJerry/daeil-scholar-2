// GreetingsPage — Letter from the foundation chair
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import { GREETINGS } from '../constants/aboutContent';

export function GreetingsPage() {
  return (
    <InfoPageShell
      title="인사말"
      subtitle="대일외고 장학회장 4기 졸업생 엄은숙"
      canonicalPath="/greetings"
    >
      <Card variant="default" padding="lg">
        <div className="space-y-6 text-text-secondary leading-relaxed text-body-md">
          <p className="font-serif text-text-primary">{GREETINGS.salutation}</p>

          <div className="space-y-4">
            {GREETINGS.paragraphs.map((paragraph) => (
              <p key={paragraph.slice(0, 24)}>{paragraph}</p>
            ))}
          </div>

          <div className="pt-2 space-y-1 text-right">
            <p className="text-text-primary">{GREETINGS.closing}</p>
            <p className="font-serif text-text-primary">— {GREETINGS.signature}</p>
          </div>
        </div>
      </Card>
    </InfoPageShell>
  );
}
