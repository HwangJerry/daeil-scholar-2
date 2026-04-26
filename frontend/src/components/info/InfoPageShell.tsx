// InfoPageShell — Shared layout for static about/info pages (SEO + heading + content + footer)
import type { ReactNode } from 'react';
import { PageMeta } from '../seo/PageMeta';
import Footer from '../layout/Footer';

interface InfoPageShellProps {
  title: string;
  subtitle?: string;
  canonicalPath: string;
  children: ReactNode;
}

export function InfoPageShell({ title, subtitle, canonicalPath, children }: InfoPageShellProps) {
  return (
    <>
      <PageMeta title={title} canonicalPath={canonicalPath} />
      <div className="space-y-8 px-4 py-6 pb-20 animate-fade-in-up">
        <header className="text-center space-y-2">
          <h1 className="text-xl font-bold text-text-primary font-serif">{title}</h1>
          {subtitle && <p className="text-text-tertiary">{subtitle}</p>}
        </header>

        <main className="mx-auto w-full max-w-3xl space-y-6">{children}</main>

        <Footer />
      </div>
    </>
  );
}
