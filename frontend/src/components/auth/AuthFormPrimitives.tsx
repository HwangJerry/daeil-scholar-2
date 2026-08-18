// AuthFormPrimitives — Small DS-backed form helpers for auth/onboarding flows.
import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';

export function AuthField({
  label,
  children,
}: {
  label: ReactNode;
  children: ReactNode;
}) {
  return (
    <div>
      <label className="mb-1 block text-body-xs font-medium text-text-muted">{label}</label>
      {children}
    </div>
  );
}

export function AuthInlineField({
  label,
  action,
  children,
}: {
  label: ReactNode;
  action: ReactNode;
  children: ReactNode;
}) {
  return (
    <div>
      <div className="mb-1 flex items-center justify-between">
        <label className="text-body-xs font-medium text-text-muted">{label}</label>
        {action}
      </div>
      {children}
    </div>
  );
}

export function AuthFieldMessage({
  tone = 'muted',
  children,
}: {
  tone?: 'muted' | 'success' | 'error';
  children: ReactNode;
}) {
  const toneClass = {
    muted: 'text-text-placeholder',
    success: 'text-success-text',
    error: 'text-error-text',
  }[tone];

  return <p className={cn('mt-1 text-caption', toneClass)}>{children}</p>;
}

export function AuthFormError({ children }: { children: ReactNode }) {
  return <p className="text-body-xs text-error-text">{children}</p>;
}

export function AuthSectionText({
  tone = 'secondary',
  children,
}: {
  tone?: 'secondary' | 'muted' | 'warning';
  children: ReactNode;
}) {
  const toneClass = {
    secondary: 'text-text-secondary',
    muted: 'text-text-placeholder',
    warning: 'text-warning-text',
  }[tone];

  return <p className={cn('text-body-xs leading-6', toneClass)}>{children}</p>;
}

export function AuthActionLink({
  tone = 'primary',
  children,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: 'primary' | 'muted' | 'danger';
}) {
  const toneClass = {
    primary: 'text-primary hover:text-primary-hover',
    muted: 'text-text-placeholder hover:text-text-secondary',
    danger: 'text-text-placeholder hover:text-error-text',
  }[tone];

  return (
    <button
      type="button"
      {...props}
      className={cn(
        'text-caption transition-colors disabled:text-text-placeholder disabled:no-underline',
        toneClass,
        props.className,
      )}
    >
      {children}
    </button>
  );
}

export function AuthDividerMark() {
  return <span className="text-caption text-border">|</span>;
}

export function AuthSelect(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={cn(
        'w-full resize-none rounded-sm border border-border bg-surface px-3 py-2 text-body-xs text-text-secondary outline-none transition-shadow duration-150 focus:border-primary/30 focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-50',
        props.className,
      )}
    />
  );
}

export function AuthTextarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={cn(
        'w-full rounded-sm border border-border bg-surface px-3 py-2 text-body-xs text-text-secondary outline-none transition-shadow duration-150 focus:border-primary/30 focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-50',
        props.className,
      )}
    />
  );
}
