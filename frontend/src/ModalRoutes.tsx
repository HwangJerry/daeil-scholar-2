// ModalRoutes — Modal route overlay layer for background-location navigation pattern
import { Routes, Route, useLocation } from 'react-router-dom';
import type { Location } from 'react-router-dom';
import { NoticeDetailModal } from './components/feed/NoticeDetailModal';
import { AdDetailModal } from './components/feed/AdDetailModal';

export function ModalRoutes() {
  const location = useLocation();
  const state = location.state as { backgroundLocation?: Location } | null;

  if (!state?.backgroundLocation) return null;

  return (
    <Routes>
      <Route path="post/:seq" element={<NoticeDetailModal />} />
      <Route path="ad/:maSeq" element={<AdDetailModal />} />
    </Routes>
  );
}
