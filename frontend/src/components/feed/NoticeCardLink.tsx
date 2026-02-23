// NoticeCardLink — Navigation wrapper that opens notice detail as modal on click
import { Link, useNavigate, useLocation } from 'react-router-dom';

interface NoticeCardLinkProps {
  seq: number;
  className?: string;
  children: React.ReactNode;
}

export function NoticeCardLink({ seq, className, children }: NoticeCardLinkProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const handleClick = (e: React.MouseEvent) => {
    // Allow browser-native new-tab behaviour on modifier+click
    if (!e.ctrlKey && !e.metaKey && !e.shiftKey) {
      e.preventDefault();
      navigate(`/post/${seq}`, { state: { backgroundLocation: location } });
    }
  };

  return (
    <Link to={`/post/${seq}`} onClick={handleClick} className={className}>
      {children}
    </Link>
  );
}
