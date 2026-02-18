// HtmlContent — renders sanitized HTML with DOMPurify as a second defense layer
import DOMPurify from 'dompurify';
import { cn } from '../../lib/utils.ts';

const PURIFY_CONFIG = {
  ALLOWED_TAGS: [
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'a', 'img', 'ul', 'ol', 'li',
    'blockquote', 'code', 'pre', 'em', 'strong', 'br', 'hr', 'table',
    'thead', 'tbody', 'tr', 'th', 'td', 'del', 'sup', 'sub',
  ],
  ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'target', 'rel', 'loading'],
};

interface Props {
  html: string;
  className?: string;
}

export function HtmlContent({ html, className }: Props) {
  const clean = DOMPurify.sanitize(html, PURIFY_CONFIG);

  return (
    <div
      className={cn('prose prose-lg max-w-none', className)}
      dangerouslySetInnerHTML={{ __html: clean }}
    />
  );
}
