// VisionSection — Editorial landing section for the foundation's mission, vision, and core values
import {
  VISION_CORE_VALUES,
  VISION_MISSION,
  VISION_VISION,
} from '../../constants/aboutContent';
import { cn } from '../../lib/utils';

const PRIMARY_STATEMENTS = [VISION_MISSION, VISION_VISION] as const;

export function VisionSection() {
  return (
    <section
      id="vision"
      aria-labelledby="vision-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] border-t border-border-subtle bg-surface px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-[1080px]')}>
        <header className={cn('max-w-2xl')}>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Vision &amp; Core Values
          </p>
          <h2
            id="vision-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            비전과 핵심가치
          </h2>
          <p className={cn('mt-5 text-sm leading-7 text-text-secondary sm:text-base')}>
            후배들의 꿈을 현실로 이어가기 위해 장학회가 지키는 방향과 원칙입니다.
          </p>
        </header>

        <div
          className={cn(
            'mt-10 divide-y divide-border border-y border-border sm:mt-12 md:grid md:grid-cols-2 md:divide-x md:divide-y-0',
          )}
        >
          {PRIMARY_STATEMENTS.map((statement, index) => (
            <article
              key={statement.label}
              className={cn(
                'py-8 sm:py-10',
                index === 0 ? 'md:pr-12 lg:pr-16' : 'md:pl-12 lg:pl-16',
              )}
            >
              <h3
                className={cn(
                  'text-xs font-semibold uppercase tracking-[0.22em] text-text-tertiary',
                )}
              >
                {statement.label}
              </h3>
              <p
                className={cn(
                  'mt-5 font-serif text-2xl font-bold leading-snug tracking-tight text-text-primary sm:text-3xl sm:leading-snug lg:text-4xl lg:leading-tight',
                )}
              >
                {statement.body}
              </p>
            </article>
          ))}
        </div>

        <section aria-labelledby="landing-core-values-heading" className={cn('mt-12 md:mt-16')}>
          <div className={cn('max-w-xl')}>
            <h3
              id="landing-core-values-heading"
              className={cn('font-serif text-xl font-bold text-text-primary sm:text-2xl')}
            >
              핵심가치
            </h3>
            <p className={cn('mt-2 text-sm leading-7 text-text-tertiary')}>
              장학회의 미션과 비전을 구체적인 실천으로 연결합니다.
            </p>
          </div>

          <ol
            className={cn(
              'mt-7 border-t border-border md:grid md:grid-cols-3 md:divide-x md:divide-border',
            )}
          >
            {VISION_CORE_VALUES.map((value, index) => (
              <li
                key={value.title}
                className={cn(
                  'border-b border-border py-6 md:border-b-0 md:px-8 md:py-8',
                  index === 0 && 'md:pl-0',
                  index === VISION_CORE_VALUES.length - 1 && 'md:pr-0',
                )}
              >
                <p
                  aria-hidden="true"
                  className={cn('text-xs font-semibold tracking-[0.18em] text-text-placeholder')}
                >
                  {String(index + 1).padStart(2, '0')}
                </p>
                <h4 className={cn('mt-3 font-serif text-lg font-bold text-text-primary')}>
                  {value.title}
                </h4>
                <ul className={cn('mt-4 space-y-2.5 text-sm leading-6 text-text-secondary')}>
                  {value.bullets.map((bullet) => (
                    <li key={bullet} className={cn('flex gap-3')}>
                      <span
                        aria-hidden="true"
                        className={cn('mt-2.5 size-1.5 shrink-0 rounded-full bg-primary')}
                      />
                      <span>{bullet}</span>
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        </section>
      </div>
    </section>
  );
}
