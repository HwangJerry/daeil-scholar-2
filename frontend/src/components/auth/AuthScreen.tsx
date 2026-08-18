// AuthScreen — Shared tokenized shell for authentication and onboarding pages.
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { cn } from '../../lib/utils';

type AuthScreenAlign = 'center' | 'start';

interface AuthScreenProps {
  title: string;
  description?: string;
  align?: AuthScreenAlign;
  children: ReactNode;
}

const AUTH_SCREEN_ALIGN_CLASS: Record<AuthScreenAlign, string> = {
  center: 'items-center',
  start: 'items-start pt-10 md:pt-16',
};

export function AuthScreen({
  title,
  description,
  align = 'center',
  children,
}: AuthScreenProps) {
  return (
    <div
      className={cn(
        'flex min-h-[60vh] justify-center animate-fade-in-up',
        AUTH_SCREEN_ALIGN_CLASS[align],
      )}
    >
      <div className="w-full max-w-sm px-4">
        <h1 className="mb-2 text-center text-title font-bold text-text-primary">{title}</h1>
        {description && (
          <p className="mb-6 text-center text-body-xs text-text-tertiary">{description}</p>
        )}
        {!description && <div className="mb-6" />}
        {children}
      </div>
    </div>
  );
}

export function AuthFooterLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link
      to={to}
      className="text-caption text-text-muted transition-colors hover:text-text-secondary"
    >
      {children}
    </Link>
  );
}

export function AuthTextLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link
      to={to}
      className="text-caption text-primary transition-colors hover:text-primary-hover"
    >
      {children}
    </Link>
  );
}

export function AuthNotice({
  tone,
  children,
}: {
  tone: 'warning' | 'error';
  children: ReactNode;
}) {
  const toneClass =
    tone === 'warning'
      ? 'bg-warning-subtle text-warning-text'
      : 'bg-error-subtle text-error-text';

  return (
    <div className={cn('mb-4 rounded-sm px-4 py-3 text-body-xs', toneClass)}>
      {children}
    </div>
  );
}
