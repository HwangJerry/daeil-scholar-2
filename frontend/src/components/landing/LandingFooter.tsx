// LandingFooter — Editorial landing footer backed by shared foundation contact data
import { Mail, MapPin, Phone } from 'lucide-react';
import { Link } from 'react-router-dom';
import {
  FOOTER_CONTACT_INFO,
  FOOTER_INFO_LINKS,
} from '../../constants/aboutContent';
import { cn } from '../../lib/utils';

export function LandingFooter() {
  const year = new Date().getFullYear();

  return (
    <footer className={cn('border-t border-primary-muted/20 bg-primary px-5 py-12 text-primary-muted sm:px-8 md:px-6 md:py-16')}>
      <div className={cn('mx-auto max-w-[1080px]')}>
        <div
          className={cn(
            'grid gap-10 border-b border-primary-muted/20 pb-10 md:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] md:gap-16 md:pb-12',
          )}
        >
          <div>
            <p className={cn('font-serif text-xl font-bold text-surface sm:text-2xl')}>
              {FOOTER_CONTACT_INFO.organizationName}
            </p>
            <p className={cn('mt-3 max-w-sm text-body-sm leading-6')}>
              동문의 마음을 모아 후배들의 배움과 성장을 이어갑니다.
            </p>
          </div>

          <address className={cn('grid gap-5 not-italic sm:grid-cols-2')}>
            <div className={cn('flex gap-3')}>
              <span
                aria-hidden="true"
                className={cn('flex min-h-11 shrink-0 self-start items-center')}
              >
                <Phone className={cn('size-4 text-surface')} />
              </span>
              <p className={cn('text-body-sm leading-6')}>
                <a
                  href={`tel:${FOOTER_CONTACT_INFO.telephone}`}
                  className={cn(
                    'inline-flex min-h-11 items-center rounded-sm transition-colors hover:text-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-muted focus-visible:ring-offset-2 focus-visible:ring-offset-primary',
                  )}
                >
                  Tel {FOOTER_CONTACT_INFO.telephone}
                </a>
                <br />
                Fax {FOOTER_CONTACT_INFO.fax}
              </p>
            </div>
            <div className={cn('flex items-start gap-3')}>
              <span
                aria-hidden="true"
                className={cn('flex min-h-11 shrink-0 items-center')}
              >
                <Mail className={cn('size-4 text-surface')} />
              </span>
              <a
                href={`mailto:${FOOTER_CONTACT_INFO.email}`}
                className={cn(
                  'inline-flex min-h-11 items-center rounded-sm break-all text-body-sm leading-6 transition-colors hover:text-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-muted focus-visible:ring-offset-2 focus-visible:ring-offset-primary',
                )}
              >
                {FOOTER_CONTACT_INFO.email}
              </a>
            </div>
            <div className={cn('flex gap-3 sm:col-span-2')}>
              <MapPin aria-hidden="true" className={cn('mt-0.5 size-4 shrink-0 text-surface')} />
              <p className={cn('text-body-sm leading-6')}>{FOOTER_CONTACT_INFO.address}</p>
            </div>
          </address>
        </div>

        <div className={cn('grid gap-8 py-10 md:grid-cols-[minmax(0,1fr)_auto] md:items-start')}>
          <nav aria-label="장학회 정보">
            <ul className={cn('flex flex-wrap gap-x-5 gap-y-3')}>
              {FOOTER_INFO_LINKS.map((link) => (
                <li key={link.to}>
                  <Link
                    to={link.to}
                    className={cn(
                      'inline-flex min-h-12 items-center rounded-full border border-primary-muted/50 px-5 text-base font-semibold text-surface transition-colors hover:border-surface hover:bg-surface/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-muted focus-visible:ring-offset-2 focus-visible:ring-offset-primary',
                    )}
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>

          <dl className={cn('grid gap-2 text-2xs leading-5 md:text-right')}>
            <div className={cn('flex flex-wrap gap-x-2 md:justify-end')}>
              <dt>회장</dt>
              <dd className={cn('text-surface')}>{FOOTER_CONTACT_INFO.chair}</dd>
              <dt className={cn('ml-2')}>총괄이사</dt>
              <dd className={cn('text-surface')}>{FOOTER_CONTACT_INFO.executiveDirector}</dd>
            </div>
            <div className={cn('flex flex-wrap gap-x-2 md:justify-end')}>
              <dt>고유등록번호</dt>
              <dd className={cn('text-surface')}>{FOOTER_CONTACT_INFO.registrationNumber}</dd>
            </div>
            <div className={cn('flex flex-wrap gap-x-2 md:justify-end')}>
              <dt>기부영수증 발급단체명</dt>
              <dd className={cn('text-surface')}>
                {FOOTER_CONTACT_INFO.donationReceiptOrganizationName}
              </dd>
            </div>
          </dl>
        </div>

        <p className={cn('border-t border-primary-muted/20 pt-6 text-2xs')}>
          COPYRIGHT ⓒ {year} {FOOTER_CONTACT_INFO.organizationName} ALL RIGHT RESERVED.
        </p>
      </div>
    </footer>
  );
}
