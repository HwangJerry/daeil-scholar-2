// LandingFooter.test — Shared foundation contact data and disclosure navigation contracts
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { LandingFooter } from '../components/landing/LandingFooter';
import Footer from '../components/layout/Footer';
import {
  FOOTER_CONTACT_INFO,
  FOOTER_INFO_LINKS,
} from '../constants/aboutContent';

function renderWithRouter(component: React.ReactNode) {
  return render(<MemoryRouter>{component}</MemoryRouter>);
}

describe('LandingFooter', () => {
  it('renders the shared contact details and mandatory disclosure link', () => {
    const { container } = renderWithRouter(<LandingFooter />);
    const disclosureLink = FOOTER_INFO_LINKS.find(
      (link) => link.label === '의무공시',
    );

    expect(disclosureLink).toBeDefined();
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.organizationName);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.telephone);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.fax);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.address);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.registrationNumber);
    expect(screen.getByRole('link', { name: FOOTER_CONTACT_INFO.email })).toHaveAttribute(
      'href',
      `mailto:${FOOTER_CONTACT_INFO.email}`,
    );
    expect(screen.getByRole('link', { name: '의무공시' })).toHaveAttribute(
      'href',
      disclosureLink?.to,
    );
  });

  it('keeps the site-wide footer backed by the same contact data', () => {
    const { container } = renderWithRouter(<Footer />);

    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.organizationName);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.telephone);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.fax);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.address);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.email);
    expect(container).toHaveTextContent(FOOTER_CONTACT_INFO.registrationNumber);
    expect(container).toHaveTextContent(
      FOOTER_CONTACT_INFO.donationReceiptOrganizationName,
    );
  });
});
