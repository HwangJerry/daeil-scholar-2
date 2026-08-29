// ScholarshipBusinessSection.test — Landing scholarship hierarchy and shared-content contracts
import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ScholarshipBusinessSection } from '../components/landing/ScholarshipBusinessSection';
import {
  BUSINESS_HEADLINE,
  BUSINESS_ITEMS,
  BUSINESS_SUBHEAD,
} from '../constants/aboutContent';

describe('ScholarshipBusinessSection', () => {
  it('uses the business anchor offset and shared section copy', () => {
    render(<ScholarshipBusinessSection />);

    const section = document.getElementById('business');
    expect(section).toHaveClass('scroll-mt-[var(--landing-header-height)]');
    expect(section).toHaveAttribute('aria-labelledby', 'scholarship-business-heading');
    expect(screen.getByRole('heading', { name: BUSINESS_HEADLINE })).toBeInTheDocument();
    expect(section).toHaveTextContent(BUSINESS_SUBHEAD.replace('\n', ' '));
  });

  it('renders every shared business item as a numbered editorial list row', () => {
    render(<ScholarshipBusinessSection />);

    const list = document.querySelector('#business ol');
    expect(list).not.toBeNull();
    expect(list?.children).toHaveLength(BUSINESS_ITEMS.length);

    BUSINESS_ITEMS.forEach((item, itemIndex) => {
      const itemHeading = screen.getByRole('heading', { name: item.title });
      const listItem = itemHeading.closest('li');

      expect(listItem).not.toBeNull();
      expect(within(listItem as HTMLElement).getByText(
        String(itemIndex + 1).padStart(2, '0'),
      )).toBeInTheDocument();

      for (const bullet of item.bullets) {
        expect(within(listItem as HTMLElement).getByText(bullet)).toBeInTheDocument();
      }
    });
  });
});
