// Sanitized HTML renderer — wraps DOMPurify as a second defense layer after server-side sanitization
import DOMPurify from 'dompurify';
import { cn } from '../../lib/utils';

const ALLOWED_TAGS = [
  'h1','h2','h3','h4','h5','h6','p','a','img','ul','ol','li',
  'blockquote','code','pre','em','strong','br','hr','table',
  'thead','tbody','tr','th','td','del','sup','sub',
];

const ALLOWED_ATTR = [
  'href','src','alt','title','class','target','rel','loading',
];

interface Props {
  html: string;
  className?: string;
}

export function HtmlContent({ html, className }: Props) {
  const clean = DOMPurify.sanitize(html, {
    ALLOWED_TAGS,
    ALLOWED_ATTR,
  });

  return (
    <div
      className={cn('prose prose-lg max-w-none', className)}
      dangerouslySetInnerHTML={{ __html: clean }}
    />
  );
}
