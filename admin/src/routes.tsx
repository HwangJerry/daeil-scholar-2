// routes — admin SPA route definitions with auth guard and layout wrapper
import { Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthGuard } from './components/auth/AdminAuthGuard.tsx';
import { AdminLayout } from './components/layout/AdminLayout.tsx';
import { DashboardPage } from './pages/DashboardPage.tsx';
import { NoticeListPage } from './pages/NoticeListPage.tsx';
import { NoticeEditPage } from './pages/NoticeEditPage.tsx';
import { AdManagePage } from './pages/AdManagePage.tsx';
import { DonationConfigPage } from './pages/DonationConfigPage.tsx';
import { MemberListPage } from './pages/MemberListPage.tsx';
import { MemberDetailPage } from './pages/MemberDetailPage.tsx';
import { DonationOrderListPage } from './pages/DonationOrderListPage.tsx';
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
        <Route path="donation" element={<DonationConfigPage />} />
        <Route path="donation/orders" element={<DonationOrderListPage />} />
        <Route path="member" element={<MemberListPage />} />
        <Route path="member/:seq" element={<MemberDetailPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
