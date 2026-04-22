// ScrollToTopButton — Floating button that appears past a scroll threshold and scrolls to top smoothly
import { useEffect, useState } from 'react';
import { ArrowUp } from 'lucide-react';

interface ScrollToTopButtonProps {
  threshold?: number;
}

const DEFAULT_THRESHOLD = 400;

export function ScrollToTopButton({ threshold = DEFAULT_THRESHOLD }: ScrollToTopButtonProps = {}) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    let ticking = false;

    const evaluate = () => {
      setVisible(window.scrollY > threshold);
      ticking = false;
    };

    const onScroll = () => {
      if (ticking) return;
      ticking = true;
      requestAnimationFrame(evaluate);
    };

    window.addEventListener('scroll', onScroll, { passive: true });
    evaluate();

    return () => {
      window.removeEventListener('scroll', onScroll);
    };
  }, [threshold]);

  if (!visible) return null;

  return (
    <button
      type="button"
      aria-label="맨 위로 이동"
      onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
      className="fixed bottom-24 md:bottom-8 right-4 md:right-6 z-40 w-11 h-11 rounded-full bg-primary text-white shadow-float flex items-center justify-center animate-fade-in hover:bg-primary-hover active:scale-[0.98] transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2"
    >
      <ArrowUp size={20} strokeWidth={2.2} />
    </button>
  );
}
