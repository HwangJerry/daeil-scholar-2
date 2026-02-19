// App — root component wiring providers, router, error boundary, and auth initialization
import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { queryClient } from './queryClient.ts';
import { AuthInitializer } from './components/auth/AuthInitializer.tsx';
import { ErrorBoundary } from './components/error/ErrorBoundary.tsx';
import { ToastProvider } from './components/ui/Toast.tsx';
import { AppRoutes } from './routes.tsx';

export default function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter basename="/admin">
          <AuthInitializer>
            <ToastProvider>
              <AppRoutes />
            </ToastProvider>
          </AuthInitializer>
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
