// Footer — Site-wide footer with foundation info nav, contact details, and copyright
import { Fragment } from 'react';
import { Link } from 'react-router-dom';
import {
  FOOTER_CONTACT_INFO,
  FOOTER_INFO_LINKS,
} from '../../constants/aboutContent';

export default function Footer() {
  const year = new Date().getFullYear();

  return (
    <footer className="mt-12 border-t border-border-subtle px-6 py-8 text-center">
      <nav
        aria-label="장학회 소개"
        className="mb-4 flex flex-wrap items-center justify-center gap-x-2 gap-y-1 text-2xs"
      >
        {FOOTER_INFO_LINKS.map((link, index) => (
          <Fragment key={link.to}>
            <Link
              to={link.to}
              className="text-text-secondary underline-offset-4 hover:text-text-primary hover:underline transition-colors"
            >
              {link.label}
            </Link>
            {index < FOOTER_INFO_LINKS.length - 1 && (
              <span aria-hidden className="text-border-subtle">
                |
              </span>
            )}
          </Fragment>
        ))}
      </nav>

      <ul className="space-y-1 text-2xs text-text-secondary">
        <li>
          <span className="font-medium">{FOOTER_CONTACT_INFO.organizationName}</span>
          <span className="mx-2 text-border-subtle">|</span>
          <span>Tel: {FOOTER_CONTACT_INFO.telephone}</span>
          <span className="mx-2 text-border-subtle">|</span>
          <span>Fax: {FOOTER_CONTACT_INFO.fax}</span>
        </li>
        <li>
          <span>회장: {FOOTER_CONTACT_INFO.chair}</span>
          <span className="mx-2 text-border-subtle">|</span>
          <span>총괄이사: {FOOTER_CONTACT_INFO.executiveDirector}</span>
        </li>
        <li>{FOOTER_CONTACT_INFO.address}</li>
        <li>
          <a
            href={`mailto:${FOOTER_CONTACT_INFO.email}`}
            className="underline hover:text-text-primary transition-colors"
          >
            {FOOTER_CONTACT_INFO.email}
          </a>
          <span className="mx-2 text-border-subtle">|</span>
          <span>고유등록번호: {FOOTER_CONTACT_INFO.registrationNumber}</span>
        </li>
        <li>
          기부영수증 발급단체명: {FOOTER_CONTACT_INFO.donationReceiptOrganizationName}
        </li>
      </ul>
      <p className="mt-3 text-2xs text-text-placeholder">
        COPYRIGHT ⓒ {year} {FOOTER_CONTACT_INFO.organizationName} ALL RIGHT RESERVED.
      </p>
    </footer>
  );
}
