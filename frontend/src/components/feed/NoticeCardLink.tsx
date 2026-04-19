// NoticeCardLink — Navigation wrapper: modal on desktop, full-page on mobile
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { useResponsive } from '../../hooks/useResponsive';

interface NoticeCardLinkProps {
  seq: number;
  className?: string;
  children: React.ReactNode;
}

export function NoticeCardLink({ seq, className, children }: NoticeCardLinkProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { isMobile } = useResponsive();

  const handleClick = (e: React.MouseEvent) => {
    // Allow browser-native new-tab behaviour on modifier+click
    if (!e.ctrlKey && !e.metaKey && !e.shiftKey) {
      e.preventDefault();
      if (isMobile) {
        navigate(`/post/${seq}`);
      } else {
        navigate(`/post/${seq}`, { state: { backgroundLocation: location } });
      }
    }
  };

  return (
    <Link to={`/post/${seq}`} onClick={handleClick} className={className}>
      {children}
    </Link>
  );
}
