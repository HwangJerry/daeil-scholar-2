// routes — admin SPA route definitions with auth guard and layout wrapper
import { Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthGuard } from './components/auth/AdminAuthGuard.tsx';
import { AdminLayout } from './components/layout/AdminLayout.tsx';
import { DashboardPage } from './pages/DashboardPage.tsx';
import { NoticeListPage } from './pages/NoticeListPage.tsx';
import { NoticeEditPage } from './pages/NoticeEditPage.tsx';
import { AdManagePage } from './pages/AdManagePage.tsx';
import { AdEditPage } from './pages/AdEditPage.tsx';
import { DonationMonitorPage } from './pages/DonationMonitorPage.tsx';
import { MemberListPage } from './pages/MemberListPage.tsx';
import { MemberDetailPage } from './pages/MemberDetailPage.tsx';
import { AdminLoginPage } from './pages/AdminLoginPage.tsx';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<AdminLoginPage />} />
      <Route
        element={
          <AdminAuthGuard>
            <AdminLayout />
          </AdminAuthGuard>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="notice" element={<NoticeListPage />} />
        <Route path="notice/new" element={<NoticeEditPage />} />
        <Route path="notice/:seq/edit" element={<NoticeEditPage />} />
        <Route path="ad" element={<AdManagePage />} />
        <Route path="ad/new" element={<AdEditPage />} />
        <Route path="ad/:maSeq/edit" element={<AdEditPage />} />
        <Route path="donation" element={<DonationMonitorPage />} />
        <Route path="member" element={<MemberListPage />} />
        <Route path="member/:seq" element={<MemberDetailPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
