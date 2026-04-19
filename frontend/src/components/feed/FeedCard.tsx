// FeedCard — Compound component providing shared design tokens for feed cards
import { cn } from '../../lib/utils';

interface RootProps {
  ref?: React.Ref<HTMLElement>;
  className?: string;
  children: React.ReactNode;
}

function Root({ ref, className, children }: RootProps) {
  return (
    <article
      ref={ref}
      className={cn(
        'rounded-2xl bg-surface border border-border-subtle shadow-card',
        'transition-all duration-200 hover:shadow-card-hover hover:border-border-hover overflow-hidden',
        className,
      )}
    >
      {children}
    </article>
  );
}

interface BodyProps {
  hasImage?: boolean;
  children: React.ReactNode;
}

function Body({ hasImage, children }: BodyProps) {
  return (
    <div className={cn('px-5 pt-5', hasImage ? 'pb-4' : 'pb-5')}>
      {children}
    </div>
  );
}

interface MetaProps {
  action?: React.ReactNode;
  children: React.ReactNode;
}

function Meta({ action, children }: MetaProps) {
  return (
    <div className="flex items-center justify-between mb-3">
      <div className="flex items-center gap-1.5 text-caption text-text-placeholder">
        {children}
      </div>
      {action}
    </div>
  );
}

function MetaDot() {
  return <span className="opacity-40">·</span>;
}

interface TitleProps {
  children: React.ReactNode;
  className?: string;
}

function Title({ children, className }: TitleProps) {
  return (
    <h3 className={cn('line-clamp-2 text-body-md font-semibold font-serif text-text-primary group-hover:text-primary transition-colors duration-150', className)}>
      {children}
    </h3>
  );
}

interface DescriptionProps {
  children: React.ReactNode;
}

function Description({ children }: DescriptionProps) {
  return (
    <div className="mb-1">
      <p className="text-body-sm text-text-tertiary leading-relaxed line-clamp-3 whitespace-pre-line">
        {children}
      </p>
    </div>
  );
}

interface ImageProps {
  src: string;
  alt: string;
}

function Image({ src, alt }: ImageProps) {
  return (
    <img
      src={src}
      alt={alt}
      className="w-full aspect-video object-cover"
      loading="lazy"
    />
  );
}

function Divider() {
  return <div className="border-t border-border-subtle mx-5" />;
}

interface FooterProps {
  children: React.ReactNode;
}

function Footer({ children }: FooterProps) {
  return (
    <div className="flex items-center px-5 py-2.5 text-xs text-text-placeholder">
      {children}
    </div>
  );
}

export const FeedCard = Object.assign(Root, {
  Body,
  Meta,
  MetaDot,
  Title,
  Description,
  Image,
  Divider,
  Footer,
});
