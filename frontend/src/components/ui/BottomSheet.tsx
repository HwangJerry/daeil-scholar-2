// BottomSheet — Mobile bottom sheet overlay with drag-to-dismiss and touch-outside-to-close
import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
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
  const activePointerId = useRef<number | null>(null);

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    if (e.pointerType === 'mouse' && e.button !== 0) return;

    e.preventDefault();
    if (!e.currentTarget.hasPointerCapture?.(e.pointerId)) {
      e.currentTarget.setPointerCapture?.(e.pointerId);
    }
    activePointerId.current = e.pointerId;
    isDragging.current = true;
    dragStartY.current = e.clientY;
    dragStartTime.current = Date.now();
    setIsSnapping(false);
  };

  const handlePointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!isDragging.current) return;
    if (activePointerId.current !== e.pointerId) return;

    e.preventDefault();
    const delta = Math.max(e.clientY - dragStartY.current, 0);
    setDragY(delta);
  };

  const handlePointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!isDragging.current) return;
    if (activePointerId.current !== e.pointerId) return;

    if (e.currentTarget.hasPointerCapture?.(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId);
    }
    isDragging.current = false;
    activePointerId.current = null;
    const delta = Math.max(e.clientY - dragStartY.current, 0);
    const elapsed = Date.now() - dragStartTime.current;
    const velocity = delta / Math.max(elapsed, 1); // px/ms

    if (delta >= DISMISS_THRESHOLD || velocity >= DISMISS_VELOCITY) {
      onClose();
    } else {
      setIsSnapping(true);
      setDragY(0);
    }
  };

  const handlePointerCancel = (e: React.PointerEvent<HTMLDivElement>) => {
    if (!isDragging.current) return;
    if (activePointerId.current !== e.pointerId) return;

    if (e.currentTarget.hasPointerCapture?.(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId);
    }
    activePointerId.current = null;
    isDragging.current = false;
    setIsSnapping(true);
    setDragY(0);
  };

  if (typeof document === 'undefined') return null;

  return createPortal(
    <div className="fixed inset-0 z-[80]">
      {/* Backdrop — onClick fires on tap, handles touch-outside-to-close */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Sheet */}
      <div
        role="dialog"
        aria-modal="true"
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
          className="flex justify-center pt-3 pb-3 flex-shrink-0 cursor-grab active:cursor-grabbing touch-none select-none"
          aria-label="바텀시트 닫기 핸들"
          data-testid="bottom-sheet-drag-handle"
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerCancel}
        >
          <div className="w-10 h-1 rounded-full bg-border" />
        </div>

        <div className="overflow-y-auto flex-1">
          {children}
        </div>
      </div>
    </div>,
    document.body,
  );
}
