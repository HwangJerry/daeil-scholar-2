// MessageComposePage — Redirects to the chat thread for the recipient, or to /messages if no recipient
import { useLocation, Navigate } from 'react-router-dom';

export function MessageComposePage() {
  const location = useLocation();
  const state = location.state as { recipientSeq?: number } | null;

  if (state?.recipientSeq) {
    return <Navigate to={`/messages/${state.recipientSeq}`} replace />;
  }
  return <Navigate to="/messages" replace />;
}
