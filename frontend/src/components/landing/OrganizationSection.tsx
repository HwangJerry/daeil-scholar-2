// OrganizationSection — Landing section header that swaps in the desktop tree or mobile accordion
import { useResponsive } from '../../hooks/useResponsive';
import { cn } from '../../lib/utils';
import { OrganizationAccordion } from './OrganizationAccordion';
import { OrganizationTree } from './OrganizationTree';

export function OrganizationSection() {
  const { isMobile } = useResponsive();

  return (
    <section
      id="organization"
      aria-labelledby="organization-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] overflow-x-clip border-t border-border-subtle bg-surface px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-[1080px]')}>
        <header className={cn('max-w-2xl')}>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Organization
          </p>
          <h2
            id="organization-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            조직도
          </h2>
          <p className={cn('mt-5 text-sm leading-7 text-text-secondary sm:text-base')}>
            동문의 뜻을 투명하고 지속 가능한 장학사업으로 연결하는 사람들입니다.
          </p>
        </header>

        <div className={cn('mt-10 sm:mt-12 md:mt-16')}>
          {isMobile ? <OrganizationAccordion /> : <OrganizationTree />}
        </div>
      </div>
    </section>
  );
}
