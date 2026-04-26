// VisionPage — Mission, Vision, and Core Values of the foundation
import { Badge } from '../components/ui/Badge';
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import {
  VISION_MISSION,
  VISION_VISION,
  VISION_CORE_VALUES,
} from '../constants/aboutContent';

export function VisionPage() {
  return (
    <InfoPageShell
      title="비전가치체계"
      subtitle="대일외고 장학회가 향하는 방향"
      canonicalPath="/vision"
    >
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card variant="default" padding="lg" className="space-y-3">
          <Badge variant="default">{VISION_MISSION.label}</Badge>
          <p className="text-body-md font-serif text-text-primary leading-relaxed">
            {VISION_MISSION.body}
          </p>
        </Card>

        <Card variant="default" padding="lg" className="space-y-3">
          <Badge variant="default">{VISION_VISION.label}</Badge>
          <p className="text-body-md font-serif text-text-primary leading-relaxed">
            {VISION_VISION.body}
          </p>
          <p className="text-body-sm text-text-tertiary">{VISION_VISION.goal}</p>
        </Card>
      </div>

      <section aria-labelledby="core-values-heading" className="space-y-4">
        <h2
          id="core-values-heading"
          className="text-base font-semibold text-text-primary font-serif text-center"
        >
          핵심가치 (Core Value)와 추진 방향
        </h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {VISION_CORE_VALUES.map((value) => (
            <Card
              key={value.title}
              variant="default"
              padding="lg"
              className="space-y-3"
            >
              <h3 className="text-body-md font-semibold font-serif text-text-primary">
                {value.title}
              </h3>
              <ul className="space-y-2 text-body-sm text-text-secondary leading-relaxed">
                {value.bullets.map((bullet) => (
                  <li key={bullet} className="flex gap-2">
                    <span aria-hidden className="mt-1 h-1 w-1 shrink-0 rounded-full bg-primary" />
                    <span>{bullet}</span>
                  </li>
                ))}
              </ul>
            </Card>
          ))}
        </div>
      </section>
    </InfoPageShell>
  );
}
