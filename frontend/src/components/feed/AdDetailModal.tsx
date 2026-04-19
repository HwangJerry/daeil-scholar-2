// AdDetailModal — Route-modal wrapper that renders AdDetailContent inside a centered modal
import { useNavigate, useLocation, Navigate } from 'react-router-dom';
import { Modal } from '../ui/Modal';
import { AdDetailContent } from './AdDetailContent';
import type { AdItem } from '../../types/api';

export function AdDetailModal() {
  const navigate = useNavigate();
  const location = useLocation();
  const adItem = (location.state as { adItem?: AdItem } | null)?.adItem;

  if (!adItem) {
    return <Navigate to="/" replace />;
  }

  return (
    <Modal onClose={() => navigate(-1)}>
      <AdDetailContent
        adItem={adItem}
        onClose={() => navigate(-1)}
        wrapperClassName="max-h-[85vh]"
      />
    </Modal>
  );
}
