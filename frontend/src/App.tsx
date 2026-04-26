// App — Root composition wiring providers, auth init, and routes together
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './queryClient';
import { AuthInitializer } from './components/auth/AuthInitializer';
import { RealtimeProvider } from './components/realtime/RealtimeProvider';
import { ScrollPolicy } from './hooks/useScrollPolicy';
import { useVisitBeacon } from './hooks/useVisitBeacon';
import AppRoutes from './routes';

const SCROLL_RESTORE_ROUTES = ['/', '/alumni'] as const;

function VisitTracker() {
  useVisitBeacon();
  return null;
}

export default function App() {
  return (
    <HelmetProvider>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <ScrollPolicy restoreAllowlist={SCROLL_RESTORE_ROUTES} />
          <VisitTracker />
          <AuthInitializer>
            <RealtimeProvider>
              <AppRoutes />
            </RealtimeProvider>
          </AuthInitializer>
        </BrowserRouter>
      </QueryClientProvider>
    </HelmetProvider>
  );
}
