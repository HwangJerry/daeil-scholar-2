// Toast — Radix Toast provider and viewport with variant-based styling
import * as ToastPrimitive from '@radix-ui/react-toast';
import { X } from 'lucide-react';
import { useToast } from '../../hooks/useToast.ts';
import { cn } from '../../lib/utils.ts';

const VARIANT_STYLES = {
  success: 'bg-success-subtle text-success-text border-success',
  error: 'bg-error-subtle text-error-text border-error-border',
  info: 'bg-soft-sky text-royal-indigo border-royal-indigo/20',
} as const;

function ToastItem({ id, variant, title, description }: {
  id: string;
  variant: 'success' | 'error' | 'info';
  title: string;
  description?: string;
}) {
  const removeToast = useToast((s) => s.removeToast);

  return (
    <ToastPrimitive.Root
      className={cn(
        'rounded-xl border px-4 py-3 shadow-lg',
        'data-[state=open]:animate-in data-[state=open]:fade-in data-[state=open]:slide-in-from-top-2',
        'data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=closed]:slide-out-to-right-full',
        VARIANT_STYLES[variant],
      )}
      duration={4000}
      onOpenChange={(open) => { if (!open) removeToast(id); }}
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <ToastPrimitive.Title className="text-sm font-semibold">
            {title}
          </ToastPrimitive.Title>
          {description && (
            <ToastPrimitive.Description className="mt-1 text-xs opacity-80">
              {description}
            </ToastPrimitive.Description>
          )}
        </div>
        <ToastPrimitive.Close aria-label="닫기" className="shrink-0 rounded-lg p-1 opacity-60 hover:opacity-100">
          <X className="h-3.5 w-3.5" />
        </ToastPrimitive.Close>
      </div>
    </ToastPrimitive.Root>
  );
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const toasts = useToast((s) => s.toasts);

  return (
    <ToastPrimitive.Provider swipeDirection="right">
      {children}
      {toasts.map((toast) => (
        <ToastItem key={toast.id} {...toast} />
      ))}
      <ToastPrimitive.Viewport className="fixed top-4 right-4 z-50 flex w-80 flex-col gap-2" />
    </ToastPrimitive.Provider>
  );
}
