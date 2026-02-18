// App — root component wiring providers, router, and auth initialization
import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { queryClient } from './queryClient.ts';
import { AuthInitializer } from './components/auth/AuthInitializer.tsx';
import { AppRoutes } from './routes.tsx';

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename="/admin">
        <AuthInitializer>
          <AppRoutes />
        </AuthInitializer>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
