// ScholarshipBusinessSection — Numbered editorial overview of foundation scholarship programs
import {
  BUSINESS_HEADLINE,
  BUSINESS_ITEMS,
  BUSINESS_SUBHEAD,
  type BusinessItem,
} from '../../constants/aboutContent';
import { cn } from '../../lib/utils';

interface ScholarshipBusinessItemProps {
  item: BusinessItem;
  itemIndex: number;
}

function ScholarshipBusinessItem({ item, itemIndex }: ScholarshipBusinessItemProps) {
  return (
    <li className={cn('border-b border-border py-9 sm:py-11 md:py-14')}>
      <article
        className={cn(
          'grid min-w-0 gap-5 md:grid-cols-[120px_minmax(0,1fr)] md:gap-10 lg:grid-cols-[160px_minmax(0,1fr)] lg:gap-14',
        )}
      >
        <p
          aria-hidden="true"
          className={cn(
            'font-serif text-4xl font-bold leading-none tracking-tight text-primary-muted sm:text-5xl md:text-6xl',
          )}
        >
          {String(itemIndex + 1).padStart(2, '0')}
        </p>

        <div className={cn('min-w-0')}>
          <h3
            className={cn(
              'max-w-3xl font-serif text-2xl font-bold leading-snug text-text-primary sm:text-3xl sm:leading-snug [overflow-wrap:anywhere]',
            )}
          >
            {item.title}
          </h3>
          <ul
            className={cn(
              'mt-6 max-w-3xl space-y-3.5 text-sm leading-7 text-text-secondary sm:text-base sm:leading-8',
            )}
          >
            {item.bullets.map((bullet) => (
              <li key={bullet} className={cn('flex gap-3')}>
                <span
                  aria-hidden="true"
                  className={cn('mt-2.5 size-1.5 shrink-0 rounded-full bg-primary')}
                />
                <span>{bullet}</span>
              </li>
            ))}
          </ul>
        </div>
      </article>
    </li>
  );
}

export function ScholarshipBusinessSection() {
  return (
    <section
      id="business"
      aria-labelledby="scholarship-business-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] border-t border-border-subtle bg-background px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-[1080px]')}>
        <header className={cn('max-w-3xl')}>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Scholarship Programs
          </p>
          <h2
            id="scholarship-business-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            {BUSINESS_HEADLINE}
          </h2>
          <p
            className={cn(
              'mt-5 whitespace-pre-line text-sm leading-7 text-text-secondary sm:text-base sm:leading-8',
            )}
          >
            {BUSINESS_SUBHEAD}
          </p>
        </header>

        <ol className={cn('mt-10 border-t border-border sm:mt-12 md:mt-16')}>
          {BUSINESS_ITEMS.map((item, itemIndex) => (
            <ScholarshipBusinessItem
              key={item.title}
              item={item}
              itemIndex={itemIndex}
            />
          ))}
        </ol>
      </div>
    </section>
  );
}
