// BottomSheet — Mobile bottom sheet overlay with drag-to-dismiss and touch-outside-to-close
import { useEffect, useRef, useState } from 'react';
import { cn } from '../../lib/utils';

const DISMISS_THRESHOLD = 150; // px — half-thumb swipe feels natural
const DISMISS_VELOCITY = 0.5;  // px/ms — ~300px in 600ms, quick flick

interface BottomSheetProps {
  children: React.ReactNode;
  onClose: () => void;
  maxHeight?: 'auto' | 'full';
}

export function BottomSheet({ children, onClose, maxHeight = 'auto' }: BottomSheetProps) {
  const [dragY, setDragY] = useState(0);
  const [isSnapping, setIsSnapping] = useState(false);
  const dragStartY = useRef(0);
  const dragStartTime = useRef(0);
  const isDragging = useRef(false);

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId);
    isDragging.current = true;
    dragStartY.current = e.clientY;
    dragStartTime.current = Date.now();
    setIsSnapping(false);
  };

  const handlePointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!isDragging.current) return;
    const delta = e.clientY - dragStartY.current;
    if (delta > 0) setDragY(delta);
  };

  const handlePointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!isDragging.current) return;
    isDragging.current = false;
    const delta = e.clientY - dragStartY.current;
    const elapsed = Date.now() - dragStartTime.current;
    const velocity = delta / Math.max(elapsed, 1); // px/ms

    if (delta >= DISMISS_THRESHOLD || velocity >= DISMISS_VELOCITY) {
      onClose();
    } else {
      setIsSnapping(true);
      setDragY(0);
    }
  };

  return (
    <div className="fixed inset-0 z-50">
      {/* Backdrop — onClick fires on tap, handles touch-outside-to-close */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Sheet */}
      <div
        style={{
          transform: `translateY(${dragY}px)`,
          transition: isSnapping
            ? 'transform 0.3s cubic-bezier(0.16, 1, 0.3, 1)'
            : undefined,
        }}
        className={cn(
          'absolute bottom-0 left-0 right-0 rounded-t-2xl bg-surface shadow-float flex flex-col',
          dragY === 0 ? 'animate-slide-up' : '',
          maxHeight === 'full'
            ? 'h-[calc(100dvh-env(safe-area-inset-top)-2rem)]'
            : 'max-h-[85vh]',
        )}
      >
        {/* Handle bar — drag target only, touch-none prevents scroll interference */}
        <div
          className="flex justify-center pt-3 pb-2 flex-shrink-0 cursor-grab touch-none"
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
        >
          <div className="w-10 h-1 rounded-full bg-border" />
        </div>

        <div className="overflow-y-auto flex-1">
          {children}
        </div>
      </div>
    </div>
  );
}
