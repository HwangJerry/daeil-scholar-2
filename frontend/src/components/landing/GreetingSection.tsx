// GreetingSection — Editorial landing-page letter from the foundation chair
import { GREETINGS } from '../../constants/aboutContent';
import { cn } from '../../lib/utils';

export function GreetingSection() {
  return (
    <section
      id="greeting"
      aria-labelledby="greeting-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] border-t border-border-subtle bg-background px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-2xl')}>
        <header>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Greeting
          </p>
          <h2
            id="greeting-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            인사말
          </h2>
        </header>

        <article className={cn('mt-10 sm:mt-12')}>
          <p
            className={cn(
              'font-serif text-lg font-semibold leading-8 text-text-primary sm:text-xl sm:leading-9',
            )}
          >
            {GREETINGS.salutation}
          </p>

          <div
            className={cn(
              'mt-8 space-y-6 text-base leading-8 text-text-secondary sm:mt-10 sm:space-y-7 sm:text-[17px] sm:leading-9',
            )}
          >
            {GREETINGS.paragraphs.map((paragraph) => (
              <p key={paragraph}>{paragraph}</p>
            ))}
          </div>

          <footer className={cn('mt-10 border-t border-border pt-7 text-right sm:mt-12 sm:pt-8')}>
            <p className={cn('text-base leading-7 text-text-primary')}>{GREETINGS.closing}</p>
            <div className={cn('mt-5 flex flex-wrap items-baseline justify-end gap-x-3 gap-y-1')}>
              <p className={cn('text-sm leading-6 text-text-tertiary')}>
                {GREETINGS.author.role} · {GREETINGS.author.cohort}
              </p>
              <p className={cn('font-serif text-xl font-bold text-text-primary')}>
                {GREETINGS.author.name}
              </p>
            </div>
          </footer>
        </article>
      </div>
    </section>
  );
}
