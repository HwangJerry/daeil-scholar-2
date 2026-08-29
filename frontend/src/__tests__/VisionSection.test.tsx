// VisionSection.test — Landing vision hierarchy and shared-content contracts
import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { VisionSection } from '../components/landing/VisionSection';
import {
  VISION_CORE_VALUES,
  VISION_MISSION,
  VISION_VISION,
} from '../constants/aboutContent';

describe('VisionSection', () => {
  it('renders the shared mission and vision as primary statements', () => {
    render(<VisionSection />);

    const section = document.getElementById('vision');
    expect(section).toHaveClass('scroll-mt-[var(--landing-header-height)]');
    expect(section).toHaveAttribute('aria-labelledby', 'vision-heading');
    expect(screen.getByRole('heading', { name: '비전과 핵심가치' })).toBeInTheDocument();

    for (const statement of [VISION_MISSION, VISION_VISION]) {
      const label = screen.getByRole('heading', { name: statement.label });
      const article = label.closest('article');

      expect(article).not.toBeNull();
      expect(within(article as HTMLElement).getByText(statement.body)).toHaveClass('text-2xl');
    }
  });

  it('renders every shared core value as a subordinate bullet list', () => {
    render(<VisionSection />);

    const coreValuesHeading = screen.getByRole('heading', { name: '핵심가치' });
    const coreValuesSection = coreValuesHeading.closest('section');
    expect(coreValuesSection).not.toBeNull();

    const coreValues = within(coreValuesSection as HTMLElement);

    for (const value of VISION_CORE_VALUES) {
      const valueHeading = coreValues.getByRole('heading', { name: value.title });
      const valueItem = valueHeading.closest('li');

      expect(valueItem).not.toBeNull();
      for (const bullet of value.bullets) {
        expect(within(valueItem as HTMLElement).getByText(bullet)).toBeInTheDocument();
      }
    }
  });
});
