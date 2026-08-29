// GreetingSection.test — Landing greeting content and anchor contracts
import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { GreetingSection } from '../components/landing/GreetingSection';
import { GREETINGS } from '../constants/aboutContent';

describe('GreetingSection', () => {
  it('renders the shared greeting copy in an anchored editorial section', () => {
    render(<GreetingSection />);

    const section = document.getElementById('greeting');

    expect(section).toHaveClass('scroll-mt-[var(--landing-header-height)]');
    expect(section).toHaveAttribute('aria-labelledby', 'greeting-heading');
    expect(screen.getByRole('heading', { name: '인사말' })).toBeInTheDocument();
    expect(screen.getByText(GREETINGS.salutation)).toBeInTheDocument();

    for (const paragraph of GREETINGS.paragraphs) {
      expect(screen.getByText(paragraph)).toBeInTheDocument();
    }

    expect(screen.getByText(GREETINGS.closing)).toBeInTheDocument();
  });

  it('identifies the chair by role, cohort, and name', () => {
    render(<GreetingSection />);

    const section = document.getElementById('greeting');
    expect(section).not.toBeNull();

    const greetingSection = within(section as HTMLElement);
    expect(
      greetingSection.getByText(`${GREETINGS.author.role} · ${GREETINGS.author.cohort}`),
    ).toBeInTheDocument();
    expect(greetingSection.getByText(GREETINGS.author.name)).toBeInTheDocument();
  });
});
