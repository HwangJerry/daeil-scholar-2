// DisclosurePageHeader — Editorial introduction shared by disclosure archive pages
interface DisclosurePageHeaderProps {
  description: string;
  title: string;
}

export function DisclosurePageHeader({ description, title }: DisclosurePageHeaderProps) {
  return (
    <header className="border-b border-border-subtle bg-surface px-5 py-16 sm:px-8 md:px-6 md:py-24">
      <div className="mx-auto max-w-[1080px]">
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder">
          Public Disclosure
        </p>
        <h1 className="mt-3 max-w-3xl font-serif text-4xl font-bold tracking-tight text-text-primary sm:text-5xl md:text-6xl">
          {title}
        </h1>
        <p className="mt-5 max-w-2xl text-sm leading-7 text-text-tertiary sm:text-base">
          {description}
        </p>
      </div>
    </header>
  );
}
