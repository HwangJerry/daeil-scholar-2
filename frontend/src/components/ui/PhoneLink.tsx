// PhoneLink — Renders a tappable tel: link on mobile, plain text on desktop
import { useResponsive } from '../../hooks/useResponsive';
import { cn } from '../../lib/utils';

const TEL_SANITIZE_RE = /[^\d+]/g;

interface PhoneLinkProps {
  phone: string;
  className?: string;
}

export function PhoneLink({ phone, className }: PhoneLinkProps) {
  const { isMobile } = useResponsive();
  const sanitized = phone.replace(TEL_SANITIZE_RE, '');

  if (!isMobile || sanitized === '') {
    return <span className={className}>{phone}</span>;
  }

  return (
    <a
      href={`tel:${sanitized}`}
      className={cn(className, 'text-primary underline')}
    >
      {phone}
    </a>
  );
}
