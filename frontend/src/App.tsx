// Root composition — wires providers, auth init, and routes together
import { BrowserRouter } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './queryClient';
import { AuthInitializer } from './components/auth/AuthInitializer';
import AppRoutes from './routes';

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthInitializer>
          <AppRoutes />
        </AuthInitializer>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
