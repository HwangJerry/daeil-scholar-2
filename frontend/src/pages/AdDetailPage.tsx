// AdDetailPage — Full-screen mobile page for ad detail, navigated to without backgroundLocation
import { useNavigate, useLocation, Navigate } from 'react-router-dom';
import { AdDetailContent } from '../components/feed/AdDetailContent';
import type { AdItem } from '../types/api';

export function AdDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const adItem = (location.state as { adItem?: AdItem } | null)?.adItem;

  if (!adItem) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="flex flex-col h-dvh bg-surface">
      <AdDetailContent
        adItem={adItem}
        onClose={() => navigate(-1)}
        wrapperClassName="flex-1 min-h-0"
      />
    </div>
  );
}
