// usePolicyAnchor — Restores policy section links after the SPA renders, including reloads.
import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { PRIVACY_SECTIONS } from './policyContent';

export function usePolicyAnchor() {
  const { hash } = useLocation();

  useEffect(() => {
    const section = PRIVACY_SECTIONS.find((item) => `#${item.id}` === hash);
    if (!section) return;

    const frame = requestAnimationFrame(() => {
      document.getElementById(section.id)?.scrollIntoView({ behavior: 'instant' });
    });
    return () => cancelAnimationFrame(frame);
  }, [hash]);
}
