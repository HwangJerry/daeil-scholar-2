// ScrollReveal.test — Viewport and reduced-motion behavior contracts
import { act, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ScrollReveal } from '../components/landing/ScrollReveal';

let observerCallback: IntersectionObserverCallback;

class IntersectionObserverMock {
  disconnect = vi.fn();
  observe = vi.fn();
  takeRecords = vi.fn(() => []);
  unobserve = vi.fn();
  root = null;
  rootMargin = '';
  thresholds = [];

  constructor(callback: IntersectionObserverCallback) {
    observerCallback = callback;
  }
}

function stubMotionPreference(prefersReducedMotion: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockReturnValue({
      matches: prefersReducedMotion,
      media: '(prefers-reduced-motion: reduce)',
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  );
}

describe('ScrollReveal', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('reveals content after it enters the viewport', () => {
    stubMotionPreference(false);
    vi.stubGlobal('IntersectionObserver', IntersectionObserverMock);
    render(
      <ScrollReveal>
        <section>콘텐츠</section>
      </ScrollReveal>,
    );

    const revealBoundary = screen.getByText('콘텐츠').parentElement;
    expect(revealBoundary).toHaveAttribute('data-revealed', 'false');

    act(() => {
      observerCallback(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      );
    });

    expect(revealBoundary).toHaveAttribute('data-revealed', 'true');
  });

  it('shows content immediately when reduced motion is preferred', () => {
    stubMotionPreference(true);
    const observerConstructor = vi.fn();
    vi.stubGlobal('IntersectionObserver', observerConstructor);
    render(
      <ScrollReveal>
        <section>감소된 모션 콘텐츠</section>
      </ScrollReveal>,
    );

    expect(screen.getByText('감소된 모션 콘텐츠').parentElement).toHaveAttribute(
      'data-revealed',
      'true',
    );
    expect(observerConstructor).not.toHaveBeenCalled();
  });
});
