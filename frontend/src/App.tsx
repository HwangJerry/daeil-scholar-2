// App — Root composition wiring providers, auth init, and routes together
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './queryClient';
import { AuthInitializer } from './components/auth/AuthInitializer';
import { RealtimeProvider } from './components/realtime/RealtimeProvider';
import { ScrollPolicy } from './hooks/useScrollPolicy';
import AppRoutes from './routes';

const SCROLL_RESTORE_ROUTES = ['/', '/alumni'] as const;

export default function App() {
  return (
    <HelmetProvider>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <ScrollPolicy restoreAllowlist={SCROLL_RESTORE_ROUTES} />
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
