// AboutPage — Foundation overview hub linking to greetings/vision/history/organization/business
import { Link } from 'react-router-dom';
import { ArrowRight } from 'lucide-react';
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import { ABOUT_LINKS } from '../constants/aboutContent';

export function AboutPage() {
  return (
    <InfoPageShell
      title="대일외고 장학회"
      subtitle="후배들의 아름다운 꿈과 높은 이상을 함께 응원합니다."
      canonicalPath="/about"
    >
      <section aria-labelledby="about-links-heading" className="space-y-4">
        <h2
          id="about-links-heading"
          className="text-base font-semibold text-text-primary font-serif"
        >
          더 자세히 보기
        </h2>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {ABOUT_LINKS.map((link) => (
            <Link
              key={link.to}
              to={link.to}
              className="group block rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              <Card
                variant="interactive"
                padding="lg"
                className="h-full flex items-start justify-between gap-3"
              >
                <div className="space-y-1">
                  <h3 className="text-body-md font-semibold font-serif text-text-primary group-hover:text-primary transition-colors">
                    {link.title}
                  </h3>
                  <p className="text-body-sm text-text-tertiary leading-relaxed">
                    {link.description}
                  </p>
                </div>
                <ArrowRight
                  size={18}
                  className="mt-1 shrink-0 text-text-tertiary group-hover:text-primary transition-colors"
                  aria-hidden
                />
              </Card>
            </Link>
          ))}
        </div>
      </section>
    </InfoPageShell>
  );
}
