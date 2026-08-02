// ModalRoutes — Modal route overlay layer for background-location navigation pattern
import { Routes, Route, useLocation } from 'react-router-dom';
import type { Location } from 'react-router-dom';
import { NoticeDetailModal } from './components/feed/NoticeDetailModal';

export function ModalRoutes() {
  const location = useLocation();
  const state = location.state as { backgroundLocation?: Location } | null;

  if (!state?.backgroundLocation) return null;

  return (
    <Routes>
      <Route path="post/:seq" element={<NoticeDetailModal />} />
    </Routes>
  );
}
