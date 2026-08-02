// Application route definitions — page routes with background-location awareness
import { Navigate, Routes, Route, useLocation } from 'react-router-dom';
import type { Location } from 'react-router-dom';
import Layout from './components/layout/Layout';
import { FeedPage } from './pages/FeedPage';
import { PostDetailPage } from './pages/PostDetailPage';
import { AboutPage } from './pages/AboutPage';
import { GreetingsPage } from './pages/GreetingsPage';
import { VisionPage } from './pages/VisionPage';
import { HistoryPage } from './pages/HistoryPage';
import { OrganizationPage } from './pages/OrganizationPage';
import { BusinessPage } from './pages/BusinessPage';
import { DisclosureListPage } from './pages/DisclosureListPage';
import { DisclosureDetailPage } from './pages/DisclosureDetailPage';
import { ModalRoutes } from './ModalRoutes';

const PUBLIC_ROUTES = [
  { path: '/', element: <FeedPage /> },
  { path: '/post/:seq', element: <PostDetailPage /> },
  { path: '/about', element: <AboutPage /> },
  { path: '/greetings', element: <GreetingsPage /> },
  { path: '/vision', element: <VisionPage /> },
  { path: '/history', element: <HistoryPage /> },
  { path: '/organization', element: <OrganizationPage /> },
  { path: '/business', element: <BusinessPage /> },
  { path: '/disclosure', element: <DisclosureListPage /> },
  { path: '/disclosure/:seq', element: <DisclosureDetailPage /> },
  { path: '*', element: <Navigate to="/" replace /> },
] as const;

export const PUBLIC_ROUTE_PATHS = PUBLIC_ROUTES.map((route) => route.path);

export default function AppRoutes() {
  const location = useLocation();
  const state = location.state as { backgroundLocation?: Location } | null;
  const backgroundLocation = state?.backgroundLocation;

  return (
    <>
      {/* Render background page when a modal is open; otherwise render current location */}
      <Routes location={backgroundLocation ?? location}>
        <Route element={<Layout />}>
          {PUBLIC_ROUTES.map((route) => (
            <Route key={route.path} path={route.path} element={route.element} />
          ))}
        </Route>
      </Routes>

      <ModalRoutes />
    </>
  );
}
