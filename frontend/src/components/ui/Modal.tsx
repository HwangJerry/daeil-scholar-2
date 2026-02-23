// Modal — Reusable modal shell with scroll container and body scroll lock
import { useEffect } from 'react';

interface ModalProps {
  children: React.ReactNode;
  onClose: () => void;
  maxWidth?: string;
}

export function Modal({ children, onClose, maxWidth = 'max-w-2xl' }: ModalProps) {
  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/50 backdrop-blur-sm">
      <div
        className="flex min-h-full items-start justify-center px-4 py-10"
        onClick={(e) => {
          if (e.target === e.currentTarget) onClose();
        }}
      >
        <div className={`w-full ${maxWidth} rounded-2xl bg-surface shadow-float animate-fade-in-up`}>
          {children}
        </div>
      </div>
    </div>
  );
}
