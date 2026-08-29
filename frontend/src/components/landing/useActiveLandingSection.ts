// useActiveLandingSection — Tracks the visible landing section for anchor navigation
import { useEffect, useState } from 'react';

const ACTIVE_VIEWPORT_ROOT_MARGIN = '0px 0px -50% 0px';
const ACTIVE_SECTION_THRESHOLDS = [0, 0.15, 0.3, 0.6];

export function useActiveLandingSection(sectionIds: readonly string[]) {
  const [activeSectionId, setActiveSectionId] = useState<string | null>(null);

  useEffect(() => {
    const sections = sectionIds
      .map((sectionId) => document.getElementById(sectionId))
      .filter((section): section is HTMLElement => section !== null);

    if (sections.length === 0 || typeof IntersectionObserver === 'undefined') {
      return;
    }

    const visibleSections = new Map<string, IntersectionObserverEntry>();
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            visibleSections.set(entry.target.id, entry);
            return;
          }

          visibleSections.delete(entry.target.id);
        });

        const nextActiveSectionId = sectionIds.find((sectionId) =>
          visibleSections.has(sectionId),
        );
        setActiveSectionId(nextActiveSectionId ?? null);
      },
      {
        rootMargin: ACTIVE_VIEWPORT_ROOT_MARGIN,
        threshold: ACTIVE_SECTION_THRESHOLDS,
      },
    );

    sections.forEach((section) => observer.observe(section));

    return () => observer.disconnect();
  }, [sectionIds]);

  return activeSectionId;
}
