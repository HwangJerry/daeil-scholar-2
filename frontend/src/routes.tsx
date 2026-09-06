// Application route definitions — page routes with background-location awareness
import { Navigate, Routes, Route, useLocation } from 'react-router-dom';
import type { Location } from 'react-router-dom';
import Layout from './components/layout/Layout';
import { EditorialLayout } from './components/layout/EditorialLayout';
import { LandingPage } from './pages/LandingPage';
import { PostDetailPage } from './pages/PostDetailPage';
import { AboutPage } from './pages/AboutPage';
import { GreetingsPage } from './pages/GreetingsPage';
import { VisionPage } from './pages/VisionPage';
import { HistoryPage } from './pages/HistoryPage';
import { OrganizationPage } from './pages/OrganizationPage';
import { BusinessPage } from './pages/BusinessPage';
import { DisclosureListPage } from './pages/DisclosureListPage';
import { DisclosureDetailPage } from './pages/DisclosureDetailPage';
import { AppSupportPage } from './pages/AppSupportPage';
import { ModalRoutes } from './ModalRoutes';

const LANDING_ROUTE = { path: '/', element: <LandingPage /> } as const;

const LAYOUT_ROUTES = [
  { path: '/post/:seq', element: <PostDetailPage /> },
  { path: '/about', element: <AboutPage /> },
  { path: '/greetings', element: <GreetingsPage /> },
  { path: '/vision', element: <VisionPage /> },
  { path: '/history', element: <HistoryPage /> },
  { path: '/organization', element: <OrganizationPage /> },
  { path: '/business', element: <BusinessPage /> },
] as const;

const EDITORIAL_ROUTES = [
  { path: '/disclosure', element: <DisclosureListPage /> },
  { path: '/disclosure/:seq', element: <DisclosureDetailPage /> },
  { path: '/support', element: <AppSupportPage /> },
] as const;

const FALLBACK_ROUTE = { path: '*', element: <Navigate to="/" replace /> } as const;

const PUBLIC_ROUTES = [
  LANDING_ROUTE,
  ...LAYOUT_ROUTES,
  ...EDITORIAL_ROUTES,
  FALLBACK_ROUTE,
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
        <Route path={LANDING_ROUTE.path} element={LANDING_ROUTE.element} />
        <Route element={<Layout />}>
          {LAYOUT_ROUTES.map((route) => (
            <Route key={route.path} path={route.path} element={route.element} />
          ))}
        </Route>
        <Route element={<EditorialLayout />}>
          {EDITORIAL_ROUTES.map((route) => (
            <Route key={route.path} path={route.path} element={route.element} />
          ))}
        </Route>
        <Route path={FALLBACK_ROUTE.path} element={FALLBACK_ROUTE.element} />
      </Routes>

      <ModalRoutes />
    </>
  );
}
